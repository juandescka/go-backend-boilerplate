package cmd

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"hris-core-service/internal/infrastructure/seeding"
)

func newSeedCommand() *cobra.Command {
	seed := &cobra.Command{Use: "seed", Short: "Manage named local development seeders"}
	list := &cobra.Command{
		Use: "list", Short: "List registered seeders", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			for _, name := range seeding.Names() {
				dataset, _ := seeding.Lookup(name)
				fmt.Fprintf(out, "%s\t%s\n", name, dataset.Description)
			}
			return out.Flush()
		},
	}
	var expected string
	run := &cobra.Command{
		Use: "run <name>", Short: "Run one named seeder", Args: cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			dataset, ok := seeding.Lookup(name)
			if !ok {
				return errors.New("unknown seeder; run seed list to see available names")
			}
			target, err := seeding.Open(cmd.Context(), expected)
			if err != nil {
				return err
			}
			defer target.Close()
			var created bool
			if err := target.Write(cmd.Context(), func(ctx context.Context, tx *sql.Tx) error {
				var err error
				created, err = dataset.Run(ctx, tx)
				return err
			}); err != nil {
				return fmt.Errorf("seeder %s failed: %w", name, err)
			}
			if created {
				fmt.Fprintf(cmd.OutOrStdout(), "Seeder %s completed. Created: 1\n", name)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Seeder %s completed. Already exists: 1. Created: 0\n", name)
			}
			return nil
		},
	}
	run.Flags().StringVar(&expected, "target", "", "Confirm the effective PostgreSQL host:port/database")
	seed.AddCommand(list, run)
	return seed
}

func init() { RootCmd.AddCommand(newSeedCommand()) }
