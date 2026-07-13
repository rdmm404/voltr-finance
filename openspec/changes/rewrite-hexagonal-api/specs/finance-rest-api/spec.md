## ADDED Requirements

### Requirement: Finance operations are exposed through a versioned JSON API
The system SHALL expose all supported transaction, user, household, category, and budget application operations through authenticated JSON endpoints under `/v1`.

#### Scenario: Client executes a supported finance operation
- **WHEN** an authenticated client sends a valid request to the corresponding `/v1` endpoint
- **THEN** the system executes the application operation and returns a JSON representation of its result

#### Scenario: Unsupported route is requested
- **WHEN** a client requests a route or method not defined by the versioned API
- **THEN** the system returns a not-found or method-not-allowed response without executing an application operation

### Requirement: Transaction resources support the existing lifecycle
The API SHALL support creating one or multiple transactions, retrieving transactions by identifier, listing transactions with filters and pagination, updating one or multiple transactions, soft-deleting transactions, and restoring soft-deleted transactions.

#### Scenario: Transaction is created
- **WHEN** a client submits a valid transaction create request
- **THEN** the API creates the transaction and returns its result with a successful creation status

#### Scenario: Transactions are listed with filters
- **WHEN** a client supplies supported transaction filters, sorting, deletion visibility, limit, and offset parameters
- **THEN** the API returns only matching transactions in the requested deterministic order

#### Scenario: Transaction is soft-deleted and restored
- **WHEN** a client soft-deletes an existing transaction and subsequently restores it with the required audit inputs
- **THEN** the transaction is excluded from ordinary reads after deletion and is available again after restoration

### Requirement: User and identity operations are exposed
The API SHALL support creating, listing, retrieving, updating, and resolving users by exactly one supported identity selector, including internal user ID, Discord ID, Telegram ID, phone number, or WhatsApp ID.

#### Scenario: User is resolved by one identity
- **WHEN** a client submits exactly one identity selector belonging to an existing user
- **THEN** the API returns that user

#### Scenario: User resolution has an invalid selector count
- **WHEN** a client submits no identity selector or more than one identity selector
- **THEN** the API returns a validation error

### Requirement: Household operations are exposed
The API SHALL support listing households, retrieving a household by internal ID, resolving a household by exactly one supported external selector, and listing users belonging to a household.

#### Scenario: Household users are listed
- **WHEN** a client requests users for an existing household ID
- **THEN** the API returns the household's users as a JSON array

#### Scenario: Household is resolved by selector
- **WHEN** a client supplies exactly one supported household selector such as name or guild ID
- **THEN** the API returns the matching household or a not-found error

### Requirement: Category operations preserve deactivation semantics
The API SHALL support creating, listing, retrieving, updating, and deactivating categories. Deleting a category resource through the API SHALL deactivate it rather than physically remove its database row.

#### Scenario: Category is deactivated
- **WHEN** a client deletes an active category resource
- **THEN** the category becomes inactive and its historical references remain intact

#### Scenario: Inactive categories are requested
- **WHEN** a client lists categories with the include-inactive option
- **THEN** the response includes both active and inactive categories

### Requirement: Monthly-budget reads and creation are separate
The API SHALL provide a read-only monthly-budget retrieval operation and a separate idempotent operation that ensures a monthly budget exists for exactly one household or user owner.

#### Scenario: Existing monthly budget is retrieved
- **WHEN** a client reads a month for an owner with an existing budget
- **THEN** the API returns the budget without mutating database state

#### Scenario: Missing monthly budget is read
- **WHEN** a client reads a month for an owner without a budget
- **THEN** the API returns a not-found error and does not create a budget

#### Scenario: Missing monthly budget is ensured
- **WHEN** a client invokes the ensure operation for an owner and month without a budget
- **THEN** the API creates the budget using the existing prior-budget copy rules and returns a creation response

#### Scenario: Existing monthly budget is ensured
- **WHEN** a client invokes the ensure operation for an owner and month that already has a budget
- **THEN** the API returns the existing budget without creating a duplicate

### Requirement: Budget lines and reports are exposed
The API SHALL support creating, updating, and deleting budget lines and retrieving budget reports while preserving all existing budget owner scoping and unmapped-transaction reporting requirements.

#### Scenario: Budget line categories are replaced
- **WHEN** a client updates a budget line with a category collection
- **THEN** the system atomically replaces that line's category mappings while preserving budget category invariants

#### Scenario: Budget report is retrieved
- **WHEN** a client requests a report for an existing budget
- **THEN** the API returns budget metadata, report lines, unmapped transactions, and totals consistent with the `budget-reporting` specification

### Requirement: API responses use stable status and error semantics
The API SHALL distinguish validation, authentication, not-found, conflict, and internal failures with appropriate HTTP status codes and a JSON error containing a stable machine-readable code and safe human-readable message.

#### Scenario: Request validation fails
- **WHEN** a request has malformed JSON, unknown fields, invalid parameters, or violates an application validation rule
- **THEN** the API returns HTTP 400 with a validation error and does not expose infrastructure details

#### Scenario: Resource is absent
- **WHEN** a requested user, household, category, transaction, budget, or budget line does not exist
- **THEN** the API returns HTTP 404 with a resource-specific not-found error

#### Scenario: Application invariant conflicts
- **WHEN** a valid request conflicts with uniqueness or another application invariant
- **THEN** the API returns HTTP 409 with a stable conflict error

#### Scenario: Unexpected failure occurs
- **WHEN** an unexpected database or server failure prevents an operation
- **THEN** the API returns HTTP 500 without exposing SQL, stack, secret, or internal implementation details

### Requirement: Collection representations are deterministic
The API SHALL encode empty collections as JSON arrays and SHALL preserve explicitly defined sorting or input ordering rather than relying on map iteration order.

#### Scenario: Collection has no items
- **WHEN** a successful list operation has no matching items
- **THEN** the API returns an empty JSON array rather than `null`
