## ADDED Requirements

### Requirement: Detailed reports classify every in-scope transaction
The system SHALL provide a detailed monthly budget report that associates each in-scope transaction with exactly one mapped budget line or with the unmapped transaction collection according to the budget's category mappings. The existing aggregate report behavior SHALL remain available unchanged.

#### Scenario: Mapped transaction appears under its line
- **WHEN** an in-scope transaction's category is assigned to a budget line
- **THEN** the detailed report includes the transaction under that line and does not include it as unmapped

#### Scenario: Categorized transaction has no line mapping
- **WHEN** an in-scope transaction has a category that is not assigned to any line in the budget
- **THEN** the detailed report includes the transaction in the unmapped collection and under no budget line

#### Scenario: Uncategorized transaction has no line mapping
- **WHEN** an in-scope transaction has no category
- **THEN** the detailed report includes the transaction in the unmapped collection and under no budget line

#### Scenario: Out-of-scope transaction is absent
- **WHEN** a transaction is deleted, outside the budget period, or outside the budget owner's exact personal or household scope
- **THEN** the detailed report excludes it from every line and from the unmapped collection

### Requirement: Detailed report transaction information
Each transaction in a detailed report SHALL include its identifier, transaction date, exact decimal amount, description, notes when present, category reference when present, and useful author identity.

#### Scenario: Transaction detail is returned
- **WHEN** a detailed report includes a mapped or unmapped transaction
- **THEN** the report provides the transaction fields required to identify and explain that spending without a separate transaction-list query

### Requirement: Detailed report is internally consistent
The detailed report SHALL load its budget structure, category mappings, aggregate amounts, mapped transactions, and unmapped transactions from one consistent read snapshot.

#### Scenario: Concurrent write occurs during report loading
- **WHEN** transaction or budget data changes while a detailed report is being loaded
- **THEN** all sections of the returned report reflect one database snapshot rather than a mixture of states

### Requirement: Detailed report totals preserve reporting semantics
Detailed report line actuals and aggregate totals SHALL equal the authoritative budget-reporting calculations, and mapped transaction classification SHALL not alter the existing definitions of allocation, mapped actual, remaining, unmapped actual, or uncategorized actual.

#### Scenario: Detailed and aggregate reports are compared
- **WHEN** aggregate and detailed reports are loaded for the same unchanged budget
- **THEN** their line actuals and aggregate total fields are equal
