package cmd

import (
	"strings"
	"testing"
)

func TestMigrationUsesExistingConfigurationLoader(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, err := newMigrationRunner(); err == nil || !strings.Contains(err.Error(), "config.json") {
		t.Fatal("migration must require the existing configuration loader")
	}
}

func TestForceRejectsInvalidVersionBeforeConfiguration(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, arg := range []string{"invalid-secret-sentinel", "-2", "9999999999999999999999999999999999"} {
		err := migrateForceCmd.RunE(migrateForceCmd, []string{arg})
		if err == nil || err.Error() != "invalid migration version: expected an integer >= -1" {
			t.Fatal("invalid force input must fail safely before loading configuration")
		}
	}
}
