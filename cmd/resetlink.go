package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/outlet-sh/outlet/internal/config"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/logic/auth"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/zeromicro/go-zero/core/conf"
)

var resetLinkCmd = &cobra.Command{
	Use:   "reset-link <email>",
	Short: "Print a one-time password reset link for a user",
	Long: `Print a password reset link for a user, good for an hour and once: the
same link "Forgot your password?" emails, for when the server can't send it
(platform SMTP not set up). Run it on the server, e.g. with kubectl exec.`,
	Args: cobra.ExactArgs(1),
	RunE: runResetLink,
}

func init() {
	rootCmd.AddCommand(resetLinkCmd)
}

func runResetLink(cmd *cobra.Command, args []string) error {
	godotenv.Load()
	var c config.Config
	if err := conf.LoadFromYamlBytes([]byte(os.ExpandEnv(string(EmbeddedConfig))), &c); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	dbPath := c.Database.Path
	if dbPath == "" {
		dbPath = "./data/outlet.db"
	}
	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("database not found at %s", dbPath)
	}
	conn, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return err
	}
	defer conn.Close()
	store := db.NewStore(conn)

	ctx := context.Background()
	user, err := store.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(args[0])))
	if err == sql.ErrNoRows {
		return fmt.Errorf("no user with the email %s", args[0])
	}
	if err != nil {
		return err
	}
	token, err := auth.IssuePasswordReset(ctx, store, user.ID)
	if err != nil {
		return err
	}
	fmt.Println(auth.ResetURL(c.App.BaseURL, token))
	return nil
}
