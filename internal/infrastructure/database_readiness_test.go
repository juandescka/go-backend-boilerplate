package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestDatabaseReadinessUnavailable(t *testing.T) {
	for _, checker := range []*DatabaseReadinessChecker{nil, NewDatabaseReadinessChecker(nil)} {
		if checker.Check(context.Background()) == nil {
			t.Fatal("unavailable pool must fail readiness")
		}
	}
}

func TestDatabaseReadinessCanceledAndClosedPool(t *testing.T) {
	// sql.Open is lazy. A canceled context and a closed pool must fail without
	// making a network connection or replacing the registry-owned pool.
	pool, err := sql.Open("pgx", "postgres://unused:unused@127.0.0.1:1/unused?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	checker := NewDatabaseReadinessChecker(pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := checker.Check(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if err := pool.Close(); err != nil {
		t.Fatal(err)
	}
	if err := checker.Check(context.Background()); err == nil {
		t.Fatal("closed application pool must fail readiness")
	}
}
