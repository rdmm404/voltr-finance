package budgets

import (
	"context"
	"time"
)

// Repository exposes use-case-level persistence operations. Implementations own
// transaction boundaries, locking, aggregate loading, and join-table mechanics.
type Repository interface {
	FindMonthly(context.Context, Owner, time.Time, time.Time) (Budget, error)
	CreateMonthlyFromTemplate(context.Context, CreateMonthlyFromTemplateInput) (Budget, error)
	CreateLineWithCategories(context.Context, CreateLineInput) (Line, error)
	UpdateLineWithCategories(context.Context, UpdateLineInput) (Line, error)
	DeleteLine(context.Context, int64) error
	LoadReportSnapshot(context.Context, int64) (ReportSnapshot, error)
	LoadDetailedMonthlySnapshot(context.Context, Owner, time.Time, time.Time) (DetailedReportSnapshot, error)
}
