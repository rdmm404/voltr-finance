package api

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestVersionedRouteContracts(t *testing.T) {
	routes := []string{
		TransactionsPath, TransactionsBulkPath, TransactionsRestorePath, TransactionPath,
		UsersPath, UserPath, UserResolvePath,
		HouseholdsPath, HouseholdPath, HouseholdUsersPath, HouseholdResolvePath,
		CategoriesPath, CategoryPath,
		MonthlyBudgetsPath, BudgetReportPath, BudgetLinesPath, BudgetLinePath,
	}
	for _, route := range routes {
		if !strings.HasPrefix(route, APIPrefix+"/") {
			t.Errorf("route %q is outside %s", route, APIPrefix)
		}
	}
	if strings.HasPrefix(LivePath, APIPrefix+"/") {
		t.Fatalf("liveness route %q must be outside authenticated API prefix", LivePath)
	}
}

func TestErrorResponseContract(t *testing.T) {
	encoded, err := json.Marshal(ErrorResponse{Error: Error{Code: "validation_error", Message: "invalid request"}})
	if err != nil {
		t.Fatalf("marshal error response: %v", err)
	}
	if got, want := string(encoded), `{"error":{"code":"validation_error","message":"invalid request"}}`; got != want {
		t.Fatalf("error response = %s, want %s", got, want)
	}
}

func TestBulkResultContractUsesIndexesAndOmitsUnknownID(t *testing.T) {
	result := BulkResult{
		Succeeded: []BulkSucceeded{{Index: 1, ID: 42}},
		Failed:    []BulkFailed{{Index: 0, Error: Error{Code: "validation_error", Message: "amount is required"}}},
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal bulk result: %v", err)
	}
	got := string(encoded)
	if strings.Contains(got, `"id":0`) || !strings.Contains(got, `"index":0`) || !strings.Contains(got, `"index":1`) {
		t.Fatalf("bulk result does not preserve index/optional id contract: %s", got)
	}
}

func TestCollectionContractsEncodeEmptyArrays(t *testing.T) {
	encoded, err := json.Marshal(BulkResult{Succeeded: []BulkSucceeded{}, Failed: []BulkFailed{}})
	if err != nil {
		t.Fatalf("marshal empty bulk result: %v", err)
	}
	if got, want := string(encoded), `{"succeeded":[],"failed":[]}`; got != want {
		t.Fatalf("empty bulk result = %s, want %s", got, want)
	}
}
