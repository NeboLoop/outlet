package workers

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

// ImportWorkerConfig configures the import worker
type ImportWorkerConfig struct {
	// How often to poll for pending imports
	PollInterval time.Duration

	// Number of concurrent import workers
	Workers int

	// Batch size for database operations
	BatchSize int

	// Upload directory for CSV files
	UploadDir string
}

// DefaultImportWorkerConfig returns sensible defaults
func DefaultImportWorkerConfig() ImportWorkerConfig {
	return ImportWorkerConfig{
		PollInterval: 5 * time.Second,
		Workers:      2,
		BatchSize:    500,
		UploadDir:    "./data/uploads",
	}
}

// ImportWorker handles background CSV imports
type ImportWorker struct {
	config ImportWorkerConfig
	store  *db.Store

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Metrics
	processed atomic.Int64
	failed    atomic.Int64
}

// NewImportWorker creates a new import worker
func NewImportWorker(store *db.Store, config ImportWorkerConfig) *ImportWorker {
	ctx, cancel := context.WithCancel(context.Background())

	return &ImportWorker{
		config: config,
		store:  store,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start begins the import worker
func (w *ImportWorker) Start() {
	logx.Infof("Starting import worker: %d workers, batch=%d",
		w.config.Workers, w.config.BatchSize)

	// Ensure upload directory exists
	if err := os.MkdirAll(w.config.UploadDir, 0755); err != nil {
		logx.Errorf("Failed to create upload directory: %v", err)
	}

	// Start import workers
	for i := 0; i < w.config.Workers; i++ {
		w.wg.Add(1)
		go w.worker(i)
	}
}

// Stop gracefully shuts down the import worker
func (w *ImportWorker) Stop() {
	logx.Info("Stopping import worker...")
	w.cancel()

	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logx.Info("Import worker stopped gracefully")
	case <-time.After(30 * time.Second):
		logx.Error("Import worker shutdown timed out")
	}
}

// worker processes pending import jobs
func (w *ImportWorker) worker(id int) {
	defer w.wg.Done()
	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	logx.Infof("Import worker %d started", id)

	for {
		select {
		case <-w.ctx.Done():
			logx.Infof("Import worker %d stopped", id)
			return
		case <-ticker.C:
			w.processPendingImports()
		}
	}
}

// processPendingImports finds and processes pending import jobs
func (w *ImportWorker) processPendingImports() {
	jobs, err := w.store.ListPendingImportJobs(w.ctx)
	if err != nil {
		if err != sql.ErrNoRows {
			logx.Errorf("Failed to list pending import jobs: %v", err)
		}
		return
	}

	for _, job := range jobs {
		select {
		case <-w.ctx.Done():
			return
		default:
		}

		if err := w.processImportJob(job); err != nil {
			logx.Errorf("Failed to process import job %s: %v", job.ID, err)
		}
	}
}

// processImportJob processes a single import job
func (w *ImportWorker) processImportJob(job db.ImportJob) error {
	logx.Infof("Processing import job %s: %s (%s)", job.ID, job.Filename, job.Type)

	// Update status to processing
	err := w.store.UpdateImportJobStatus(w.ctx, db.UpdateImportJobStatusParams{
		ID:     job.ID,
		Status: sql.NullString{String: "processing", Valid: true},
	})
	if err != nil {
		return err
	}

	// Open CSV file; it holds people's addresses, so it goes when the job ends
	filePath := ImportFile(w.config.UploadDir, job.ID)
	file, err := os.Open(filePath)
	if err != nil {
		return w.failJob(job.ID, "Failed to open file: "+err.Error())
	}
	defer os.Remove(filePath)
	defer file.Close()

	// Parse CSV
	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	// Read header
	header, err := reader.Read()
	if err != nil {
		return w.failJob(job.ID, "Failed to read CSV header: "+err.Error())
	}

	// Map column indices
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.ToLower(strings.TrimSpace(col))] = i
	}

	// Count total rows first
	var totalRows int64
	for {
		_, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		totalRows++
	}

	// Reset file position
	file.Seek(0, 0)
	reader = csv.NewReader(file)
	reader.TrimLeadingSpace = true
	reader.Read() // Skip header

	// Update total rows
	err = w.store.SetImportJobTotalRows(w.ctx, db.SetImportJobTotalRowsParams{
		ID:        job.ID,
		TotalRows: sql.NullInt64{Int64: totalRows, Valid: true},
	})
	if err != nil {
		logx.Errorf("Failed to set total rows: %v", err)
	}

	// Process rows
	var processed, success, errors, skipped int64
	var errorMessages []string

	for {
		select {
		case <-w.ctx.Done():
			return w.ctx.Err()
		default:
		}

		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errors++
			errorMessages = append(errorMessages, "Row parse error: "+err.Error())
			continue
		}

		processed++

		// Process based on import type
		var importErr error
		switch job.Type {
		case "subscribers":
			importErr = w.importSubscriber(job, record, colMap)
		case "suppression":
			importErr = w.importSuppression(job, record, colMap)
		case "blocked_domains":
			importErr = w.importBlockedDomain(job, record, colMap)
		default:
			importErr = nil
			skipped++
		}

		if importErr != nil {
			errors++
			if len(errorMessages) < 100 { // Limit error messages
				errorMessages = append(errorMessages, importErr.Error())
			}
		} else {
			success++
		}

		// Update progress every batch
		if processed%int64(w.config.BatchSize) == 0 {
			w.store.UpdateImportJobProgress(w.ctx, db.UpdateImportJobProgressParams{
				ID:            job.ID,
				ProcessedRows: sql.NullInt64{Int64: processed, Valid: true},
				SuccessCount:  sql.NullInt64{Int64: success, Valid: true},
				ErrorCount:    sql.NullInt64{Int64: errors, Valid: true},
				SkipCount:     sql.NullInt64{Int64: skipped, Valid: true},
			})
		}
	}

	// Final progress update
	w.store.UpdateImportJobProgress(w.ctx, db.UpdateImportJobProgressParams{
		ID:            job.ID,
		ProcessedRows: sql.NullInt64{Int64: processed, Valid: true},
		SuccessCount:  sql.NullInt64{Int64: success, Valid: true},
		ErrorCount:    sql.NullInt64{Int64: errors, Valid: true},
		SkipCount:     sql.NullInt64{Int64: skipped, Valid: true},
	})

	// Mark as completed
	err = w.store.UpdateImportJobStatus(w.ctx, db.UpdateImportJobStatusParams{
		ID:     job.ID,
		Status: sql.NullString{String: "completed", Valid: true},
	})
	if err != nil {
		return err
	}

	w.processed.Add(1)
	logx.Infof("Import job %s completed: %d processed, %d success, %d errors, %d skipped",
		job.ID, processed, success, errors, skipped)

	return nil
}

// importSubscriber imports a single subscriber row
func (w *ImportWorker) importSubscriber(job db.ImportJob, record []string, colMap map[string]int) error {
	// Get email (required)
	emailIdx, ok := colMap["email"]
	if !ok {
		return nil // Skip if no email column
	}
	if emailIdx >= len(record) {
		return nil
	}
	email := strings.TrimSpace(strings.ToLower(record[emailIdx]))
	if email == "" {
		return nil
	}

	// Get name (optional)
	var name string
	if nameIdx, ok := colMap["name"]; ok && nameIdx < len(record) {
		name = strings.TrimSpace(record[nameIdx])
	}

	if _, err := w.store.AddSubscriber(w.ctx, job.OrgID, job.ListID.Int64, email, name, "import"); err != nil {
		return fmt.Errorf("%s: %w", email, err)
	}
	return nil
}

// importSuppression imports a suppression list entry
func (w *ImportWorker) importSuppression(job db.ImportJob, record []string, colMap map[string]int) error {
	emailIdx, ok := colMap["email"]
	if !ok {
		return nil
	}
	if emailIdx >= len(record) {
		return nil
	}
	email := strings.TrimSpace(strings.ToLower(record[emailIdx]))
	if email == "" {
		return nil
	}

	// Get reason (optional)
	var reason string
	if reasonIdx, ok := colMap["reason"]; ok && reasonIdx < len(record) {
		reason = strings.TrimSpace(record[reasonIdx])
	}
	if reason == "" {
		reason = "imported"
	}

	// Add to suppression list
	_, err := w.store.AddToSuppressionList(w.ctx, db.AddToSuppressionListParams{
		OrgID:  job.OrgID,
		Email:  email,
		Reason: sql.NullString{String: reason, Valid: true},
		Source: sql.NullString{String: "import", Valid: true},
	})
	if err != nil {
		// Ignore duplicate errors
		if !strings.Contains(err.Error(), "UNIQUE constraint") {
			return err
		}
	}

	return nil
}

// importBlockedDomain imports a blocked domain entry
func (w *ImportWorker) importBlockedDomain(job db.ImportJob, record []string, colMap map[string]int) error {
	domainIdx, ok := colMap["domain"]
	if !ok {
		return nil
	}
	if domainIdx >= len(record) {
		return nil
	}
	domain := strings.TrimSpace(strings.ToLower(record[domainIdx]))
	if domain == "" {
		return nil
	}

	// Add to blocked domains
	_, err := w.store.AddBlockedDomain(w.ctx, db.AddBlockedDomainParams{
		ID:     time.Now().UnixNano(),
		OrgID:  job.OrgID,
		Domain: domain,
	})
	if err != nil {
		// Ignore duplicate errors
		if !strings.Contains(err.Error(), "UNIQUE constraint") {
			return err
		}
	}

	return nil
}

// failJob marks an import job as failed
func (w *ImportWorker) failJob(id, reason string) error {
	w.failed.Add(1)
	return w.store.UpdateImportJobStatus(w.ctx, db.UpdateImportJobStatusParams{
		ID:     id,
		Status: sql.NullString{String: "failed", Valid: true},
	})
}

// Stats returns worker statistics
func (w *ImportWorker) Stats() (processed, failed int64) {
	return w.processed.Load(), w.failed.Load()
}

// ImportFile is where an import job's CSV waits for the worker.
func ImportFile(uploadDir, jobID string) string {
	return filepath.Join(uploadDir, jobID+".csv")
}

// StartImportWorker starts the import worker from service context
func StartImportWorker(svcCtx *svc.ServiceContext) *ImportWorker {
	config := DefaultImportWorkerConfig()
	config.UploadDir = svcCtx.UploadDir

	worker := NewImportWorker(svcCtx.DB, config)
	worker.Start()

	return worker
}
