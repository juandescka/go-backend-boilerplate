package cmd

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/spf13/cobra"

	"hris-core-service/config"
	"hris-core-service/internal/infrastructure/migration"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Manage PostgreSQL database migrations",
}

var migrateUpCmd = &cobra.Command{
	Use:   "up",
	Short: "Run all pending database migrations",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		runner, err := newMigrationRunner()
		if err != nil {
			return err
		}
		defer func() { err = errors.Join(err, runner.Close()) }()

		if err := runner.Up(); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				fmt.Fprintln(cmd.OutOrStdout(), "No pending migrations")
				return nil
			}
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Migrations applied successfully")
		return nil
	},
}

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Roll back the latest database migration",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		runner, err := newMigrationRunner()
		if err != nil {
			return err
		}
		defer func() { err = errors.Join(err, runner.Close()) }()

		if err := runner.Down(); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				fmt.Fprintln(cmd.OutOrStdout(), "No migration to roll back")
				return nil
			}
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Migration rolled back successfully")
		return nil
	},
}

var migrateVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the current database migration version",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		runner, err := newMigrationRunner()
		if err != nil {
			return err
		}
		defer func() { err = errors.Join(err, runner.Close()) }()

		version, dirty, err := runner.Version()
		if err != nil {
			return err
		}

		if version == database.NilVersion {
			if dirty {
				fmt.Fprintln(cmd.OutOrStdout(), "Migration version: none dirty: true")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Migration version: none")
			}
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Migration version: %d dirty: %t\n", version, dirty)
		return nil
	},
}

var migrateForceCmd = &cobra.Command{
	Use:   "force <version>",
	Short: "Force the database migration version",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		version, err := strconv.Atoi(args[0])
		if err != nil || version < database.NilVersion {
			return fmt.Errorf("invalid migration version: expected an integer >= -1")
		}

		runner, err := newMigrationRunner()
		if err != nil {
			return err
		}
		defer func() { err = errors.Join(err, runner.Close()) }()

		if err := runner.Force(version); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Migration version forced to %d\n", version)
		return nil
	},
}

func init() {
	migrateCmd.AddCommand(migrateUpCmd, migrateDownCmd, migrateVersionCmd, migrateForceCmd)
	RootCmd.AddCommand(migrateCmd)
}

func newMigrationRunner() (*migration.Runner, error) {
	if err := config.LoadConfig(); err != nil {
		return nil, err
	}
	return migration.NewRunner(migration.NewConfig())
}
