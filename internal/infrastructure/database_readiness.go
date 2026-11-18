package infrastructure

import (
	"context"
	"database/sql"
	"errors"

	"hris-core-service/internal/usecases/health"
)

type DatabaseReadinessChecker struct {
	db *sql.DB
}

var _ health.ReadinessChecker = (*DatabaseReadinessChecker)(nil)

func NewDatabaseReadinessChecker(db *sql.DB) *DatabaseReadinessChecker {
	return &DatabaseReadinessChecker{db: db}
}

func (c *DatabaseReadinessChecker) Check(ctx context.Context) error {
	if c == nil || c.db == nil {
		return errors.New("database readiness pool is unavailable")
	}
	return c.db.PingContext(ctx)
}
