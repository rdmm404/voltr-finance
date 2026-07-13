# budget-reporting Specification

## Purpose
TBD - created by syncing change fix-budget-report-scope-and-unmapped-categories.

## Requirements

### Requirement: Report transactions match the budget owner scope
The system SHALL include only non-deleted transactions within the budget's inclusive date range whose ownership scope exactly matches the budget. A household budget SHALL include transactions assigned to that household regardless of which household member authored them. A personal budget SHALL include transactions authored by the budget owner only when the transaction is not assigned to any household.

#### Scenario: Household report excludes personal transactions
- **WHEN** a household budget report is generated and its owner has personal transactions in the budget period
- **THEN** the report excludes transactions whose household is absent or differs from the budget household

#### Scenario: Household report includes transactions by household members
- **WHEN** a household budget report is generated for transactions assigned to that household in the budget period
- **THEN** the report includes those transactions regardless of which household member authored them

#### Scenario: Personal report excludes household transactions
- **WHEN** a personal budget report is generated and the owner authored both personal and household transactions in the budget period
- **THEN** the report includes only transactions with no household assignment

### Requirement: Report unmapped transactions explicitly
The system SHALL list every in-scope report transaction whose category is absent or is not mapped to any line in the reported budget. Each listed transaction SHALL include its identifier, date, description, amount, and category reference when a category exists.

#### Scenario: Uncategorized transaction is listed
- **WHEN** an in-scope transaction in the budget period has no category
- **THEN** the report lists the transaction as unmapped with no category reference

#### Scenario: Categorized transaction without a budget-line mapping is listed
- **WHEN** an in-scope transaction's category is not assigned to any line in the reported budget
- **THEN** the report lists the transaction as unmapped with its category reference

#### Scenario: Mapped transaction is not listed as unmapped
- **WHEN** an in-scope transaction's category is assigned to a line in the reported budget
- **THEN** the transaction contributes to that line and does not appear in the unmapped transaction list

#### Scenario: Out-of-scope transaction is not listed as unmapped
- **WHEN** a transaction is outside the budget period, deleted, or does not match the budget owner scope
- **THEN** the transaction does not appear in the unmapped transaction list

### Requirement: Report unmapped spending totals
The system SHALL expose the summed amount of all unmapped transactions as `unmappedActualAmount`. The existing `uncategorizedActualAmount` SHALL remain available and SHALL continue to sum only in-scope transactions with no category.

#### Scenario: Unmapped total includes both unmapped forms
- **WHEN** a report contains both an uncategorized transaction and a categorized transaction not mapped to a budget line
- **THEN** `unmappedActualAmount` equals the sum of both transactions and `uncategorizedActualAmount` equals only the uncategorized transaction amount

#### Scenario: No transactions are unmapped
- **WHEN** every in-scope transaction is mapped to a budget line
- **THEN** the unmapped transaction list is empty and `unmappedActualAmount` is zero
