# Transaction Categories Design

## Purpose

Add transaction categories as global, database-backed reference data. Categories let transactions be grouped for entry, listing, and reporting without requiring every transaction to be categorized.

This design is intentionally limited to categories. It does not define budgeting behavior.

## Goals

- Store categories in the database rather than hardcoding them in Go.
- Keep categories global across the app, not scoped to users or households.
- Allow transactions to remain uncategorized.
- Support CLI-friendly category assignment by stable code.
- Allow category names to change without breaking transaction history or automation.
- Replace the old `budget_category` categorization path with a dedicated category model.

## Non-Goals

- No new budget design or implementation.
- No household-specific category customization.
- No custom display ordering.
- No hard deletion of categories that are referenced by transactions.

## Data Model

Create a global `category` table:

```sql
CREATE TABLE category (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code VARCHAR UNIQUE NOT NULL,
    name VARCHAR NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

Add a nullable category reference to transactions:

```sql
ALTER TABLE transaction
    ADD COLUMN category_id BIGINT REFERENCES category(id);

CREATE INDEX idx_transaction_category_id ON transaction(category_id);
```

`transaction.category_id` is nullable. Existing transactions and ambiguous future transactions can stay uncategorized.

Drop the old budget-coupled categorization schema as part of this change:

```sql
ALTER TABLE transaction
    DROP COLUMN budget_category_id;

DROP TABLE budget_category;
```

The existing `budget` table is outside this category design. This spec only removes `budget_category` because it currently mixes category assignment with budget allocation and would conflict with the dedicated category model.

## Category Rules

Categories have two user-visible identifiers:

- `name`: display label, such as `Groceries`.
- `code`: stable slug, such as `groceries`.

Creation accepts `name` and optional `code`.

If `code` is omitted, generate it from `name`:

- Lowercase letters and numbers.
- Whitespace and separators become single hyphens.
- Leading and trailing hyphens are removed.
- Generated code must still pass uniqueness validation.

Examples:

```text
Groceries -> groceries
Home Utilities -> home-utilities
Restaurants & Takeout -> restaurants-takeout
```

`name` can be updated freely. `code` should be treated as stable because CLI commands and scripts may rely on it. If code updates are supported, they should be explicit rather than part of an ordinary rename.

Categories should be deactivated by setting `is_active = false`. Hard deletion should only be allowed when no transaction references the category.

Default listing order is `name ASC, id ASC`.

## API and Service Behavior

Add category operations:

- Create category.
- List categories, with an option to include inactive categories.
- Get category by id.
- Get category by code.
- Update category name and description.
- Deactivate category.

Transaction create and update requests should accept optional category assignment by either:

- `categoryId`
- `categoryCode`

If both are provided, they must refer to the same category. If they conflict, return a validation error.

Inactive categories should not be assignable to new or updated transactions by default. Existing transactions may still reference inactive categories.

Transaction get/list responses should include category details when present:

```json
{
  "id": 123,
  "amount": 42.5,
  "category": {
    "id": 1,
    "code": "groceries",
    "name": "Groceries"
  }
}
```

Filtering transactions by category can be added as part of the implementation if it stays small. Assignment and response rendering are the core scope.

## CLI Behavior

Add category-oriented commands using the existing CLI style:

```bash
voltr category create "Groceries"
voltr category create "Restaurants & Takeout" --code restaurants
voltr category list
voltr category rename groceries "Food & Groceries"
voltr category deactivate restaurants
```

Transaction commands should use category codes for ergonomic assignment:

```bash
voltr tx create --amount 42.50 --description Costco --category groceries
voltr tx update 123 --category utilities
voltr tx update 123 --clear-category
```

When a category is created with an auto-generated code, the CLI should print the resulting code so it can be reused.

## Validation and Errors

Validate category creation:

- Name is required.
- Code is required after generation.
- Code matches the slug format.
- Code is unique.

Validate transaction category assignment:

- Referenced category exists.
- Referenced category is active.
- `categoryId` and `categoryCode` do not conflict when both are provided.

Use existing app error patterns for validation and database errors.

## Testing

Add focused tests for:

- Category code generation.
- Category create/list/update/deactivate service behavior.
- Transaction create with `categoryCode`.
- Transaction update with `categoryCode`.
- Clearing a transaction category.
- Rejection of unknown or inactive categories.
- Response rendering with category details.
- CLI command parsing for category create/list/deactivate and transaction category flags.

## Open Decisions

None. The first implementation should stay focused on optional transaction categorization with global dynamic categories.
