package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/joho/godotenv"
	"github.com/outlet-sh/outlet/internal/config"
	"github.com/outlet-sh/outlet/internal/db"
	"github.com/outlet-sh/outlet/internal/db/migrations"
	"github.com/outlet-sh/outlet/internal/services/signalnet"
	"github.com/spf13/cobra"
	"github.com/zeromicro/go-zero/core/conf"

	_ "modernc.org/sqlite"
)

var networkKeyName string

var networkCmd = &cobra.Command{
	Use:   "network",
	Short: "Manage the Signal Network hub",
	Long: `Manage the Outlet Signal Network hub (run with SignalNet.Hub: true).

An instance key is a subscription: an instance configured with SignalNet.HubURL and
SignalNet.InstanceKey reports hashed outcomes to this hub and can look up pooled signals.`,
}

var networkKeyCmd = &cobra.Command{
	Use:   "key",
	Short: "Create, list and revoke instance keys",
}

var networkKeyCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Issue a new instance key (printed once)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := openNetworkStore()
		if err != nil {
			return err
		}
		inst, key, err := signalnet.NewHub(store).CreateInstance(cmd.Context(), networkKeyName)
		if err != nil {
			return fmt.Errorf("create key: %w", err)
		}
		fmt.Printf("Instance: %s (%s)\nKey:      %s\n\nStore the key now; it cannot be shown again.\n", inst.Name, inst.ID, key)
		return nil
	},
}

var networkKeyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List instance keys with usage",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := openNetworkStore()
		if err != nil {
			return err
		}
		insts, err := store.ListNetworkInstances(cmd.Context())
		if err != nil {
			return err
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tACTIVE\tLOOKUPS\tREPORTS\tCREATED\tLAST SEEN")
		for _, i := range insts {
			fmt.Fprintf(w, "%s\t%s\t%t\t%d\t%d\t%s\t%s\n", i.ID, i.Name, i.Active == 1, i.LookupsCount, i.ReportsCount, i.CreatedAt, i.LastSeenAt.String)
		}
		return w.Flush()
	},
}

var networkKeyRevokeCmd = &cobra.Command{
	Use:   "revoke <id>",
	Short: "Revoke an instance key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := openNetworkStore()
		if err != nil {
			return err
		}
		n, err := store.RevokeNetworkInstance(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("no instance with id %s", args[0])
		}
		fmt.Printf("Revoked %s\n", args[0])
		return nil
	},
}

func init() {
	networkKeyCreateCmd.Flags().StringVar(&networkKeyName, "name", "", "customer or instance name (required)")
	_ = networkKeyCreateCmd.MarkFlagRequired("name")
	networkKeyCmd.AddCommand(networkKeyCreateCmd, networkKeyListCmd, networkKeyRevokeCmd)
	networkCmd.AddCommand(networkKeyCmd)
	rootCmd.AddCommand(networkCmd)
}

// openNetworkStore opens the configured database and applies pending migrations.
func openNetworkStore() (*db.Store, error) {
	godotenv.Load()
	var c config.Config
	if err := conf.LoadFromYamlBytes([]byte(os.ExpandEnv(string(EmbeddedConfig))), &c); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	dbPath := c.Database.Path
	if dbPath == "" {
		dbPath = "./data/outlet.db"
	}
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("database not found at %s", dbPath)
	}
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	if _, err := conn.Exec("PRAGMA busy_timeout=5000"); err != nil {
		return nil, err
	}
	if err := migrations.Run(conn); err != nil {
		return nil, err
	}
	return db.NewStore(conn), nil
}
