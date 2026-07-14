# Purpose

Define deterministic item-level outcomes and commit semantics for bulk finance mutations.

## Requirements
### Requirement: Bulk mutations report successes and failures explicitly
Every bulk create, update, delete, and restore response SHALL contain separate `succeeded` and `failed` arrays that account for every submitted input item exactly once.

#### Scenario: Bulk request partially succeeds
- **WHEN** some valid input items succeed and other input items fail
- **THEN** the response lists each successful item in `succeeded` and each unsuccessful item in `failed`

#### Scenario: All bulk items succeed
- **WHEN** every submitted input item succeeds
- **THEN** `succeeded` accounts for every item and `failed` is an empty array

#### Scenario: All bulk items fail individually
- **WHEN** the request envelope is valid but every submitted item fails
- **THEN** `failed` accounts for every item and `succeeded` is an empty array

### Requirement: Bulk outcomes retain original input identity
Each bulk result item SHALL include its zero-based original input index and SHALL include the affected or known resource ID when one is available.

#### Scenario: Invalid item precedes a valid item
- **WHEN** item zero fails validation and item one succeeds
- **THEN** the failed result has index zero and the successful result has index one regardless of internal filtering or processing order

#### Scenario: Resource ID is unknown
- **WHEN** an item fails before a resource ID can be assigned or resolved
- **THEN** its failed result retains the input index and omits the unknown ID

### Requirement: Bulk results are deterministic
The `succeeded` and `failed` arrays SHALL each be ordered by ascending original input index and SHALL NOT depend on Go map iteration or database return order.

#### Scenario: Database returns resources out of input order
- **WHEN** the persistence adapter returns successful resources in a different order than the submitted items
- **THEN** the API returns bulk result arrays ordered by original input index

### Requirement: Item failures have structured errors
Each failed bulk result SHALL contain a stable machine-readable error code and a safe human-readable message describing that item's failure.

#### Scenario: Duplicate transaction fails
- **WHEN** one transaction in a bulk create conflicts with an existing transaction
- **THEN** that item is returned in `failed` with the duplicate-transaction error while independent items continue processing

#### Scenario: Requested delete ID does not exist
- **WHEN** one ID in a bulk delete request does not identify a deletable transaction
- **THEN** that input is explicitly returned in `failed` rather than silently omitted

#### Scenario: Infrastructure failure prevents further attribution
- **WHEN** an infrastructure failure prevents remaining items from being processed or individually attributed
- **THEN** every unprocessed input is returned in `failed` with a safe internal-error code

### Requirement: Valid bulk requests use operation-level success status
A syntactically valid bulk envelope SHALL return HTTP 200 whether all items succeed, some items succeed, or all items fail individually. A malformed or envelope-level invalid request SHALL return a request-level HTTP 400 error instead of an item result.

#### Scenario: Mixed item result is returned
- **WHEN** a syntactically valid bulk request has mixed outcomes
- **THEN** the API returns HTTP 200 with both result arrays

#### Scenario: Bulk envelope is malformed
- **WHEN** a bulk request body is invalid JSON or does not contain the required collection
- **THEN** the API returns HTTP 400 with a top-level error and executes no items

### Requirement: Bulk items are independently committed
Bulk mutation items SHALL retain partial-success semantics and SHALL NOT be rolled back solely because another item in the same request fails.

#### Scenario: Later item fails
- **WHEN** an earlier item succeeds and a later independent item fails
- **THEN** the earlier item's mutation remains committed and is reported as successful
