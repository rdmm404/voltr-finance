# Test Baseline

Recorded before implementation on 2026-07-13 from commit `3cdbe7e` (`main`).

## Command

```text
go test ./...
```

## Result

```text
PASS: 104 tests across 12 packages
```

The baseline completed without failures. Characterization added in phase 1 protects transaction input indexing and deterministic result ordering, missing delete/restore IDs, monthly-budget owner and prior-copy behavior, transactional budget-line category changes, and the existing budget-report owner/unmapped-transaction SQL requirements.
