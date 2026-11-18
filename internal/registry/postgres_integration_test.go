package registry

import (
	"context"
	"os"
	"testing"

	"hris-core-service/config"
	"hris-core-service/internal/infrastructure"
)

func TestPostgresSharedPoolIntegration(t *testing.T) {
	if os.Getenv("M4A_INTEGRATION") != "1" {
		t.Skip("set M4A_INTEGRATION=1 with local PostgreSQL configuration")
	}
	t.Chdir("../..")
	t.Setenv("METRICS_ENABLED", "true")
	reg, err := NewRegistry()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reg.Close() })
	pool, err := reg.DB.DB()
	if err != nil || pool != reg.sqlDB || reg.DB.Dialector.Name() != "postgres" {
		t.Fatal("GORM must use registry-owned PostgreSQL SQL pool")
	}
	if pool.Stats().MaxOpenConnections != config.AppConfig.Database.MaxOpenConnections {
		t.Fatal("pool limits not applied")
	}
	checker := infrastructure.NewDatabaseReadinessChecker(pool)
	if err := checker.Check(context.Background()); err != nil {
		t.Fatal("live PostgreSQL readiness failed")
	}
	var result int
	if err := reg.DB.Raw("SELECT 1").Scan(&result).Error; err != nil || result != 1 {
		t.Fatal("GORM PostgreSQL read-only query failed")
	}
	families, err := reg.Metrics.Gatherer().Gather()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, family := range families {
		if family.GetName() == config.AppConfig.Metrics.Namespace+"_db_connections_max_open" {
			found = true
			if len(family.Metric) != 1 || family.Metric[0].GetGauge().GetValue() != float64(pool.Stats().MaxOpenConnections) {
				t.Fatal("shared-pool metrics mismatch")
			}
		}
	}
	if !found {
		t.Fatal("shared-pool metric missing")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if checker.Check(ctx) == nil {
		t.Fatal("canceled readiness succeeded")
	}
	if err := reg.Close(); err != nil {
		t.Fatal(err)
	}
	if err := reg.Close(); err != nil {
		t.Fatal(err)
	}
	if pool.Stats().OpenConnections != 0 || checker.Check(context.Background()) == nil {
		t.Fatal("registry did not close its shared pool")
	}
}
