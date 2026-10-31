package backup

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/outlet-sh/outlet/internal/access"
	"github.com/outlet-sh/outlet/internal/db"
	backupService "github.com/outlet-sh/outlet/internal/services/backup"
	"github.com/outlet-sh/outlet/internal/svc"
	"github.com/outlet-sh/outlet/internal/types"
	"github.com/outlet-sh/outlet/internal/websocket"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateBackupLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateBackupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateBackupLogic {
	return &CreateBackupLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateBackupLogic) CreateBackup(req *types.CreateBackupRequest) (resp *types.CreateBackupResponse, err error) {
	if _, err := access.Require(l.ctx, l.svcCtx.DB, "", access.SuperAdmin); err != nil {
		return nil, err
	}

	// Generate backup ID and filename
	backupID := uuid.New().String()
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("outlet-backup-%s.zip", timestamp)

	// Determine backup directory - use absolute path for SQLite VACUUM INTO
	backupDir, err := filepath.Abs(filepath.Join(".", "backups"))
	if err != nil {
		return nil, fmt.Errorf("failed to determine backup directory: %w", err)
	}
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	backupPath := filepath.Join(backupDir, filename)

	// Create backup record in database
	storageType := "local"
	if req.UploadToS3 {
		storageType = "s3"
	}

	record, err := l.svcCtx.DB.CreateBackupRecord(l.ctx, db.CreateBackupRecordParams{
		ID:          backupID,
		Filename:    filename,
		FilePath:    sql.NullString{String: backupPath, Valid: true},
		FileSize:    0,
		BackupType:  "manual",
		StorageType: storageType,
		S3Bucket:    sql.NullString{String: req.S3Bucket, Valid: req.S3Bucket != ""},
		S3Key:       sql.NullString{},
		Status:      "in_progress",
		CreatedBy:   sql.NullString{String: l.getUserID(), Valid: l.getUserID() != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create backup record: %w", err)
	}

	// Get S3 config if uploading to S3
	var s3Config *backupService.S3Config
	if req.UploadToS3 {
		s3Config = l.getS3Config()
		if s3Config == nil {
			return nil, fmt.Errorf("S3 not configured or credentials missing")
		}
	}

	// Get database path for use in goroutine - must be absolute
	dbPath, err := filepath.Abs(l.svcCtx.Config.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve database path: %w", err)
	}

	// Perform the backup - create zip with both .db and .sql files
	go func() {
		ctx := context.Background()
		var finalErr error
		logger := logx.WithContext(ctx)

		// Temp files
		tempDbPath := filepath.Join(backupDir, fmt.Sprintf("temp-%s.db", backupID))
		tempSqlPath := filepath.Join(backupDir, fmt.Sprintf("temp-%s.sql", backupID))

		defer func() {
			// Clean up temp files
			os.Remove(tempDbPath)
			os.Remove(tempSqlPath)

			if r := recover(); r != nil {
				errMsg := fmt.Sprintf("panic: %v", r)
				logger.Errorf("Panic during backup: %v", r)
				l.svcCtx.DB.UpdateBackupStatus(ctx, db.UpdateBackupStatusParams{
					ID:           backupID,
					Status:       "failed",
					ErrorMessage: sql.NullString{String: errMsg, Valid: true},
				})
				// Broadcast failure via websocket
				if l.svcCtx.WebSocketHub != nil {
					l.svcCtx.WebSocketHub.Broadcast(websocket.NewBackupUpdate(backupID, "failed", filename, 0, errMsg))
				}
			}
			if finalErr != nil {
				logger.Errorf("Backup failed: %v", finalErr)
				l.svcCtx.DB.UpdateBackupStatus(ctx, db.UpdateBackupStatusParams{
					ID:           backupID,
					Status:       "failed",
					ErrorMessage: sql.NullString{String: finalErr.Error(), Valid: true},
				})
				// Broadcast failure via websocket
				if l.svcCtx.WebSocketHub != nil {
					l.svcCtx.WebSocketHub.Broadcast(websocket.NewBackupUpdate(backupID, "failed", filename, 0, finalErr.Error()))
				}
			}
		}()

		logger.Infof("Starting backup to: %s (DB: %s)", backupPath, dbPath)

		// 1. Create binary backup using VACUUM INTO
		logger.Infof("Creating binary backup...")
		_, finalErr = l.svcCtx.DB.GetDB().ExecContext(ctx, fmt.Sprintf("VACUUM INTO '%s'", tempDbPath))
		if finalErr != nil {
			logger.Errorf("VACUUM INTO failed: %v", finalErr)
			return
		}

		// 2. Create SQL dump
		logger.Infof("Creating SQL dump...")
		finalErr = createSQLDump(dbPath, tempSqlPath)
		if finalErr != nil {
			logger.Errorf("SQL dump failed: %v", finalErr)
			return
		}

		// 3. Create zip archive containing both files
		logger.Infof("Creating zip archive...")
		finalErr = createZipArchive(backupPath, timestamp, tempDbPath, tempSqlPath)
		if finalErr != nil {
			logger.Errorf("Zip creation failed: %v", finalErr)
			return
		}
		logger.Infof("Backup archive created successfully")

		// Get file size
		info, err := os.Stat(backupPath)
		if err != nil {
			logger.Errorf("Failed to stat backup file: %v", err)
			finalErr = err
			return
		}
		logger.Infof("Backup file size: %d bytes", info.Size())

		// Upload to S3 if requested
		var s3Key string
		if s3Config != nil {
			s3Key, finalErr = backupService.UploadToS3(ctx, *s3Config, backupPath, filename)
			if finalErr != nil {
				return
			}
		}

		// Update backup record with completion
		_, finalErr = l.svcCtx.DB.UpdateBackupComplete(ctx, db.UpdateBackupCompleteParams{
			ID:       backupID,
			FileSize: info.Size(),
		})
		if finalErr != nil {
			logger.Errorf("Failed to update backup record: %v", finalErr)
			return
		}
		logger.Infof("Backup completed successfully: %s (%d bytes)", filename, info.Size())

		// Broadcast success via websocket
		if l.svcCtx.WebSocketHub != nil {
			l.svcCtx.WebSocketHub.Broadcast(websocket.NewBackupUpdate(backupID, "completed", filename, info.Size(), ""))
		}

		// Update S3 key if uploaded
		if s3Key != "" {
			l.svcCtx.DB.UpdateBackupS3Key(ctx, db.UpdateBackupS3KeyParams{
				ID:    backupID,
				S3Key: sql.NullString{String: s3Key, Valid: true},
			})
		}
	}()

	return &types.CreateBackupResponse{
		Backup: dbBackupToType(record),
	}, nil
}

// createZipArchive creates a zip file containing both the .db and .sql backup files
func createZipArchive(zipPath, timestamp, dbPath, sqlPath string) error {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Add the .db file
	if err := addFileToZip(zipWriter, dbPath, fmt.Sprintf("outlet-backup-%s.db", timestamp)); err != nil {
		return fmt.Errorf("failed to add db file to zip: %w", err)
	}

	// Add the .sql file
	if err := addFileToZip(zipWriter, sqlPath, fmt.Sprintf("outlet-backup-%s.sql", timestamp)); err != nil {
		return fmt.Errorf("failed to add sql file to zip: %w", err)
	}

	return nil
}

// addFileToZip adds a single file to a zip archive
func addFileToZip(zipWriter *zip.Writer, filePath, archiveName string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = archiveName
	header.Method = zip.Deflate // Use compression

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}

// createSQLDump creates a SQL dump of the database using pure Go
func createSQLDump(dbPath, outputPath string) error {
	// Open a separate connection to the database for dumping
	dumpDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database for dump: %w", err)
	}
	defer dumpDB.Close()

	// Create the output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Write header
	fmt.Fprintln(outFile, "-- Outlet.sh Database Dump")
	fmt.Fprintf(outFile, "-- Generated at: %s\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintln(outFile, "PRAGMA foreign_keys=OFF;")
	fmt.Fprintln(outFile, "BEGIN TRANSACTION;")
	fmt.Fprintln(outFile)

	// Get all table names
	rows, err := dumpDB.Query("SELECT name, sql FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return fmt.Errorf("failed to query tables: %w", err)
	}

	type tableInfo struct {
		name string
		sql  string
	}
	var tables []tableInfo

	for rows.Next() {
		var name string
		var tableSql sql.NullString
		if err := rows.Scan(&name, &tableSql); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan table info: %w", err)
		}
		if tableSql.Valid && tableSql.String != "" {
			tables = append(tables, tableInfo{name: name, sql: tableSql.String})
		}
	}
	rows.Close()

	// Dump each table
	for _, table := range tables {
		// Write CREATE TABLE statement
		fmt.Fprintf(outFile, "%s;\n", table.sql)

		// Get all rows from the table
		dataRows, err := dumpDB.Query(fmt.Sprintf("SELECT * FROM %q", table.name))
		if err != nil {
			return fmt.Errorf("failed to query table %s: %w", table.name, err)
		}

		columns, err := dataRows.Columns()
		if err != nil {
			dataRows.Close()
			return fmt.Errorf("failed to get columns for %s: %w", table.name, err)
		}

		// Create a slice of interface{} to hold the values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		for dataRows.Next() {
			if err := dataRows.Scan(valuePtrs...); err != nil {
				dataRows.Close()
				return fmt.Errorf("failed to scan row from %s: %w", table.name, err)
			}

			// Build INSERT statement
			fmt.Fprintf(outFile, "INSERT INTO %q VALUES(", table.name)
			for i, val := range values {
				if i > 0 {
					fmt.Fprint(outFile, ",")
				}
				writeValue(outFile, val)
			}
			fmt.Fprintln(outFile, ");")
		}
		dataRows.Close()
		fmt.Fprintln(outFile)
	}

	// Dump indexes
	indexRows, err := dumpDB.Query("SELECT sql FROM sqlite_master WHERE type='index' AND sql IS NOT NULL ORDER BY name")
	if err != nil {
		return fmt.Errorf("failed to query indexes: %w", err)
	}
	for indexRows.Next() {
		var indexSql string
		if err := indexRows.Scan(&indexSql); err != nil {
			indexRows.Close()
			return fmt.Errorf("failed to scan index: %w", err)
		}
		fmt.Fprintf(outFile, "%s;\n", indexSql)
	}
	indexRows.Close()

	fmt.Fprintln(outFile)
	fmt.Fprintln(outFile, "COMMIT;")

	return nil
}

// writeValue writes a SQL-escaped value to the output
func writeValue(w io.Writer, val interface{}) {
	if val == nil {
		fmt.Fprint(w, "NULL")
		return
	}

	switch v := val.(type) {
	case int, int32, int64, float32, float64:
		fmt.Fprintf(w, "%v", v)
	case bool:
		if v {
			fmt.Fprint(w, "1")
		} else {
			fmt.Fprint(w, "0")
		}
	case []byte:
		fmt.Fprintf(w, "'%s'", escapeSQLString(string(v)))
	case string:
		fmt.Fprintf(w, "'%s'", escapeSQLString(v))
	default:
		fmt.Fprintf(w, "'%s'", escapeSQLString(fmt.Sprintf("%v", v)))
	}
}

// escapeSQLString escapes single quotes for SQL
func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func dbBackupToType(b db.BackupHistory) types.BackupInfo {
	return types.BackupInfo{
		Id:           b.ID,
		Filename:     b.Filename,
		FilePath:     b.FilePath.String,
		FileSize:     b.FileSize,
		BackupType:   b.BackupType,
		StorageType:  b.StorageType,
		S3Bucket:     b.S3Bucket.String,
		S3Key:        b.S3Key.String,
		Status:       b.Status,
		ErrorMessage: b.ErrorMessage.String,
		CreatedBy:    b.CreatedBy.String,
		StartedAt:    b.StartedAt.String,
		CompletedAt:  b.CompletedAt.String,
		CreatedAt:    b.CreatedAt.String,
	}
}

// getUserID retrieves the user ID from auth context
func (l *CreateBackupLogic) getUserID() string {
	if user, ok := access.User(l.ctx); ok {
		return user.ID
	}
	return ""
}

// getS3Config retrieves S3 configuration from platform settings
func (l *CreateBackupLogic) getS3Config() *backupService.S3Config {
	settings, err := l.svcCtx.DB.GetPlatformSettingsByCategory(l.ctx, "backup")
	if err != nil {
		l.Errorf("Failed to get backup settings: %v", err)
		return nil
	}

	cfg := &backupService.S3Config{}
	var accessKeyEncrypted, secretKeyEncrypted []byte

	for _, s := range settings {
		switch s.Key {
		case "backup.s3_enabled":
			if s.ValueText.String != "true" {
				return nil // S3 not enabled
			}
		case "backup.s3_bucket":
			cfg.Bucket = s.ValueText.String
		case "backup.s3_region":
			cfg.Region = s.ValueText.String
		case "backup.s3_prefix":
			cfg.Prefix = s.ValueText.String
		case "backup.s3_access_key":
			if s.ValueEncrypted.Valid {
				var err error
				accessKeyEncrypted, err = hex.DecodeString(s.ValueEncrypted.String)
				if err != nil {
					l.Errorf("Failed to decode S3 access key: %v", err)
				}
			}
		case "backup.s3_secret_key":
			if s.ValueEncrypted.Valid {
				var err error
				secretKeyEncrypted, err = hex.DecodeString(s.ValueEncrypted.String)
				if err != nil {
					l.Errorf("Failed to decode S3 secret key: %v", err)
				}
			}
		}
	}

	// Decrypt credentials if crypto service is available
	if l.svcCtx.CryptoService != nil {
		if len(accessKeyEncrypted) > 0 {
			decrypted, err := l.svcCtx.CryptoService.DecryptString(accessKeyEncrypted)
			if err == nil {
				cfg.AccessKey = decrypted
			} else {
				l.Errorf("Failed to decrypt S3 access key: %v", err)
			}
		}
		if len(secretKeyEncrypted) > 0 {
			decrypted, err := l.svcCtx.CryptoService.DecryptString(secretKeyEncrypted)
			if err == nil {
				cfg.SecretKey = decrypted
			} else {
				l.Errorf("Failed to decrypt S3 secret key: %v", err)
			}
		}
	}

	// Validate required fields
	if cfg.Bucket == "" {
		l.Errorf("S3 bucket not configured")
		return nil
	}

	return cfg
}
