package postgres_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	appbudgets "rdmm404/voltr-finance/internal/app/budgets"
	appcategories "rdmm404/voltr-finance/internal/app/categories"
	apperrors "rdmm404/voltr-finance/internal/app/errors"
	"rdmm404/voltr-finance/internal/app/patch"
	apptransactions "rdmm404/voltr-finance/internal/app/transactions"
	appusers "rdmm404/voltr-finance/internal/app/users"
	"rdmm404/voltr-finance/internal/database/sqlc"
	postgresbudgets "rdmm404/voltr-finance/internal/postgres/budgets"
	postgrescategories "rdmm404/voltr-finance/internal/postgres/categories"
	postgreshouseholds "rdmm404/voltr-finance/internal/postgres/households"
	postgrestransactions "rdmm404/voltr-finance/internal/postgres/transactions"
	postgresusers "rdmm404/voltr-finance/internal/postgres/users"
)

type identityResolver struct{ id int64 }

func (r identityResolver) ResolveUserID(context.Context, apptransactions.IdentitySelector) (int64, error) {
	return r.id, nil
}

type categoryResolver struct{ service *appcategories.Service }

func (r categoryResolver) ResolveActiveCategoryID(ctx context.Context, id *int64, code *string) (*int64, error) {
	if id == nil && code == nil {
		return nil, nil
	}
	item, err := r.service.ResolveActive(ctx, id, code)
	if err != nil {
		return nil, err
	}
	return &item.ID, nil
}

func TestPostgresAdaptersEndToEnd(t *testing.T) {
	if os.Getenv("VOLTR_INTEGRATION_TEST") == "" {
		t.Skip("set VOLTR_INTEGRATION_TEST=1 to run against PostgreSQL")
	}
	ctx := context.Background()
	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		connString = "postgres://voltr:voltr@127.0.0.1:5432/voltr_finance?sslmode=disable&search_path=transactions"
	}
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	queries := sqlc.New(pool)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	userRepo := postgresusers.NewRepository(queries)
	userService := appusers.NewService(userRepo)
	user, err := userService.Create(ctx, appusers.CreateInput{Name: "Adapter User " + suffix, TelegramID: stringPointer("tg-" + suffix)})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, user.ID) })
	resolved, err := userService.Resolve(ctx, appusers.Selector{TelegramID: stringPointer("tg-" + suffix + "|display")})
	if err != nil || resolved.ID != user.ID {
		t.Fatalf("resolve user=%+v error=%v", resolved, err)
	}
	if _, err := userService.Create(ctx, appusers.CreateInput{Name: "Duplicate", TelegramID: stringPointer("tg-" + suffix)}); !apperrors.IsKind(err, apperrors.KindConflict) {
		t.Fatalf("duplicate user error=%v", err)
	}

	var householdID int64
	err = pool.QueryRow(ctx, `INSERT INTO household(name,guild_id) VALUES($1,$2) RETURNING id`, "Adapter Household "+suffix, "guild-"+suffix).Scan(&householdID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM household_user WHERE household_id=$1`, householdID)
		pool.Exec(context.Background(), `DELETE FROM household WHERE id=$1`, householdID)
	})
	if _, err := pool.Exec(ctx, `INSERT INTO household_user(household_id,user_id) VALUES($1,$2)`, householdID, user.ID); err != nil {
		t.Fatal(err)
	}
	householdRepo := postgreshouseholds.NewRepository(queries)
	household, err := householdRepo.GetByID(ctx, householdID)
	if err != nil || household.GuildID != "guild-"+suffix {
		t.Fatalf("household=%+v error=%v", household, err)
	}
	members, err := householdRepo.ListUsers(ctx, householdID)
	if err != nil || len(members) != 1 || members[0].ID != user.ID {
		t.Fatalf("members=%+v error=%v", members, err)
	}

	categoryRepo := postgrescategories.NewRepository(queries)
	categoryService := appcategories.NewService(categoryRepo)
	category, err := categoryService.Create(ctx, appcategories.CreateInput{Name: "Adapter Category " + suffix, Code: stringPointer("adapter-" + suffix)})
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), `DELETE FROM category WHERE id=$1`, category.ID) })
	if _, err := categoryService.Create(ctx, appcategories.CreateInput{Name: "Duplicate", Code: &category.Code}); !apperrors.IsKind(err, apperrors.KindConflict) {
		t.Fatalf("duplicate category error=%v", err)
	}

	transactionRepo := postgrestransactions.NewRepository(pool)
	transactionService := apptransactions.NewService(transactionRepo, identityResolver{id: user.ID}, categoryResolver{service: categoryService})
	transaction, err := transactionService.Create(ctx, apptransactions.CreateInput{Amount: 25.50, TransactionDate: time.Now().UTC(), HouseholdID: &householdID, CategoryID: &category.ID})
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), `DELETE FROM "transaction" WHERE id=$1`, transaction.ID) })
	if transaction.AuthorName == "" || transaction.Category == nil || transaction.Category.ID != category.ID {
		t.Fatalf("transaction details=%+v", transaction)
	}
	if _, err := transactionService.Create(ctx, apptransactions.CreateInput{Amount: 25.50, TransactionDate: transaction.TransactionDate, HouseholdID: &householdID, CategoryID: &category.ID}); !apperrors.IsKind(err, apperrors.KindConflict) {
		t.Fatalf("duplicate transaction error=%v", err)
	}
	updatedAmount, updatedDescription := float32(30.75), "concurrent update"
	updateErrors := make([]error, 2)
	var updateWait sync.WaitGroup
	updateWait.Add(2)
	go func() {
		defer updateWait.Done()
		_, updateErrors[0] = transactionService.Update(ctx, apptransactions.UpdateInput{ID: transaction.ID, Amount: &updatedAmount})
	}()
	go func() {
		defer updateWait.Done()
		_, updateErrors[1] = transactionService.Update(ctx, apptransactions.UpdateInput{ID: transaction.ID, Description: patch.Set(updatedDescription)})
	}()
	updateWait.Wait()
	finalTransaction, err := transactionService.Get(ctx, transaction.ID, true)
	if updateErrors[0] != nil || updateErrors[1] != nil || err != nil || finalTransaction.Amount != updatedAmount || finalTransaction.Description == nil || *finalTransaction.Description != updatedDescription {
		t.Fatalf("concurrent transaction=%+v updateErrors=%v getError=%v", finalTransaction, updateErrors, err)
	}
	wantHash, _ := apptransactions.Hash(finalTransaction.Description, finalTransaction.TransactionDate, finalTransaction.AuthorID, finalTransaction.HouseholdID, finalTransaction.CategoryID, finalTransaction.Amount)
	if finalTransaction.Hash != wantHash {
		t.Fatalf("hash=%q want=%q", finalTransaction.Hash, wantHash)
	}
	if _, err := transactionService.SoftDelete(ctx, apptransactions.DeleteInput{ID: transaction.ID, DeletedByUserID: user.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := transactionService.Restore(ctx, apptransactions.RestoreInput{ID: transaction.ID, RestoredByUserID: user.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := transactionService.SoftDelete(ctx, apptransactions.DeleteInput{ID: -1, DeletedByUserID: user.ID}); !apperrors.IsKind(err, apperrors.KindNotFound) {
		t.Fatalf("missing delete error=%v", err)
	}

	budgetRepo := postgresbudgets.NewRepository(pool)
	budgetService := appbudgets.NewService(budgetRepo)
	now := transaction.TransactionDate
	monthly := appbudgets.MonthlyInput{Owner: appbudgets.Owner{HouseholdID: &householdID}, Year: now.Year(), Month: int(now.Month())}
	ensured, err := budgetService.EnsureMonthly(ctx, monthly)
	if err != nil || !ensured.Created {
		t.Fatalf("ensure budget=%+v error=%v", ensured, err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), `DELETE FROM budget WHERE id=$1`, ensured.Budget.ID) })
	line, err := budgetService.CreateLine(ctx, appbudgets.CreateLineInput{BudgetID: ensured.Budget.ID, Name: "Food", AllocationAmount: "100.00", CategoryCodes: []string{category.Code}})
	if err != nil || len(line.Categories) != 1 || line.Categories[0].Code != category.Code {
		t.Fatalf("create line=%+v error=%v", line, err)
	}
	categoryCodes := []string{category.Code}
	line, err = budgetService.UpdateLine(ctx, appbudgets.UpdateLineInput{LineID: line.ID, CategoryCodes: &categoryCodes})
	if err != nil || len(line.Categories) != 1 || line.Categories[0].Code != category.Code {
		t.Fatalf("replace line categories=%+v error=%v", line, err)
	}
	if _, err := budgetService.CreateLine(ctx, appbudgets.CreateLineInput{BudgetID: ensured.Budget.ID, Name: "Duplicate mapping", AllocationAmount: "1.00", CategoryIDs: []int64{category.ID}}); !apperrors.IsKind(err, apperrors.KindConflict) || apperrors.MessageOf(err) != "category already mapped to another budget line" {
		t.Fatalf("category mapping conflict=%v", err)
	}
	lineResults := make([]appbudgets.Line, 2)
	lineErrors := make([]error, 2)
	var lineWait sync.WaitGroup
	for index := range lineResults {
		lineWait.Add(1)
		go func(i int) {
			defer lineWait.Done()
			lineResults[i], lineErrors[i] = budgetService.CreateLine(ctx, appbudgets.CreateLineInput{BudgetID: ensured.Budget.ID, Name: fmt.Sprintf("Concurrent %d", i), AllocationAmount: "1.00"})
		}(index)
	}
	lineWait.Wait()
	if lineErrors[0] != nil || lineErrors[1] != nil || lineResults[0].SortOrder == lineResults[1].SortOrder {
		t.Fatalf("concurrent lines=%+v errors=%v", lineResults, lineErrors)
	}
	report, err := budgetService.Report(ctx, ensured.Budget.ID)
	if err != nil || report.Totals.ActualAmount != "30.75" || report.Totals.RemainingAmount != "71.25" {
		t.Fatalf("report=%+v error=%v", report, err)
	}

	unmappedDescription, unmappedNotes := "Unmapped detail", "shown in dashboard"
	unmappedTransaction, err := transactionService.Create(ctx, apptransactions.CreateInput{
		Amount: 4.25, TransactionDate: now.Add(time.Second), HouseholdID: &householdID,
		Description: &unmappedDescription, Notes: &unmappedNotes,
	})
	if err != nil {
		t.Fatalf("create unmapped transaction: %v", err)
	}
	deletedTransaction, err := transactionService.Create(ctx, apptransactions.CreateInput{Amount: 8, TransactionDate: now.Add(2 * time.Second), HouseholdID: &householdID})
	if err != nil {
		t.Fatalf("create deleted transaction: %v", err)
	}
	if _, err := transactionService.SoftDelete(ctx, apptransactions.DeleteInput{ID: deletedTransaction.ID, DeletedByUserID: user.ID}); err != nil {
		t.Fatalf("delete detailed-report transaction: %v", err)
	}
	outOfScopeTransaction, err := transactionService.Create(ctx, apptransactions.CreateInput{Amount: 9, TransactionDate: now.Add(3 * time.Second)})
	if err != nil {
		t.Fatalf("create out-of-scope transaction: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM "transaction" WHERE id = ANY($1)`, []int64{unmappedTransaction.ID, deletedTransaction.ID, outOfScopeTransaction.ID})
	})

	detailed, err := budgetService.DetailedMonthlyReport(ctx, monthly)
	if err != nil {
		t.Fatalf("detailed report: %v", err)
	}
	if len(detailed.Lines) != 3 || len(detailed.Lines[0].Transactions) != 1 || detailed.Lines[0].Transactions[0].ID != transaction.ID {
		t.Fatalf("mapped detailed transactions=%+v", detailed.Lines)
	}
	if len(detailed.UnmappedTransactions) != 1 || detailed.UnmappedTransactions[0].ID != unmappedTransaction.ID || detailed.UnmappedTransactions[0].Notes == nil || detailed.UnmappedTransactions[0].Author.Name == "" {
		t.Fatalf("unmapped detailed transactions=%+v", detailed.UnmappedTransactions)
	}
	if detailed.Totals.ActualAmount != report.Totals.ActualAmount || detailed.Totals.UnmappedActualAmount != "4.25" || detailed.Totals.UncategorizedActualAmount != "4.25" {
		t.Fatalf("detailed totals=%+v aggregate before unmapped=%+v", detailed.Totals, report.Totals)
	}

	next := now.AddDate(0, 1, 0)
	nextMonthly := appbudgets.MonthlyInput{Owner: monthly.Owner, Year: next.Year(), Month: int(next.Month())}
	results := make([]appbudgets.EnsureResult, 2)
	errs := make([]error, 2)
	var wait sync.WaitGroup
	for index := range results {
		wait.Add(1)
		go func(i int) { defer wait.Done(); results[i], errs[i] = budgetService.EnsureMonthly(ctx, nextMonthly) }(index)
	}
	wait.Wait()
	if errs[0] != nil || errs[1] != nil || results[0].Budget.ID != results[1].Budget.ID || results[0].Created == results[1].Created || len(results[0].Budget.Lines) != 3 {
		t.Fatalf("concurrent ensure results=%+v errors=%v", results, errs)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), `DELETE FROM budget WHERE id=$1`, results[0].Budget.ID) })
}

func stringPointer(value string) *string { return &value }
