# Voltr Finance CLI Repurpose Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a host-installed `voltr-finance` CLI that exposes transaction CRUD, user CRUD, and read-only household operations through a direct Postgres connection.

**Architecture:** Keep the existing Discord/Genkit bot entrypoint working while adding a new CLI entrypoint. Add an application service layer above sqlc and the existing transaction service so the CLI stays thin and future API/MCP transports can reuse the same use cases. Use typed DTOs, deterministic identity resolution, soft-delete transaction behavior, JSON/CSV renderers, and config loaded from JSON.

**Tech Stack:** Go 1.24, pgx/sqlc, `github.com/alecthomas/kong`, stdlib `encoding/json`, existing dbmate migrations.

---

## File Structure

- Create `db/migrations/20260505000000_cli_identity_and_soft_delete.sql`: schema changes for provider identities and soft delete.
- Modify `internal/database/query.sql`: add user, household, transaction list/delete/restore sqlc queries.
- Regenerate `internal/database/sqlc/*`: generated models/query methods after sqlc changes.
- Create `internal/cli/config.go` and `internal/cli/config_test.go`: CLI config lookup and JSON parsing.
- Create `internal/app/errors.go`: stable application error codes and write error DTOs.
- Create `internal/app/identity.go` and `internal/app/identity_test.go`: provider-specific identity selector validation and resolution.
- Create `internal/app/users.go` and `internal/app/users_test.go`: user create/update/get/resolve/list use cases.
- Create `internal/app/households.go`: read-only household use cases.
- Create `internal/app/transactions.go` and `internal/app/transactions_test.go`: transaction create/update/get/list/delete/restore use cases.
- Create `internal/cli/render.go` and `internal/cli/render_test.go`: compact text, JSON, and CSV output.
- Create `internal/cli/commands.go`, `internal/cli/commands_test.go`, and `cmd/voltr-finance/main.go`: Kong command structs, wiring, exit codes.
- Modify `Dockerfile`: keep bot target, add CLI build/export target.
- Add `docs/cli.md`: install, config, and command examples.

## Task 1: CLI Config

**Files:**
- Create: `internal/cli/config.go`
- Create: `internal/cli/config_test.go`
- [ ] **Step 1: Write config tests**

Create `internal/cli/config_test.go` with tests for:

```go
func TestResolveConfigPathPrefersFlag(t *testing.T) {}
func TestResolveConfigPathFallsBackToEnv(t *testing.T) {}
func TestResolveConfigPathDefaultsToHomeConfig(t *testing.T) {}
func TestLoadConfigParsesDatabaseFields(t *testing.T) {}
func TestLoadConfigRejectsUnknownFields(t *testing.T) {}
func TestLoadConfigRequiresDatabaseFields(t *testing.T) {}
```

Expected behavior:

```text
--config wins over VOLTR_CONFIG
VOLTR_CONFIG wins over $HOME/.config/voltr-finance/config.json
database fields parse into DBConfig
unknown JSON fields return validation errors
missing host, port, name, user, or password returns validation errors
```

- [ ] **Step 2: Implement config loading**

Create `internal/cli/config.go` with:

```go
type Config struct {
	Database DBConfig `json:"database"`
}

type DBConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
	PoolSize int    `json:"poolSize"`
}
```

Decode with `json.Decoder.DisallowUnknownFields()` so the config is a strict structured model, not a loose map.

Implement:

```go
func ResolveConfigPath(flagPath string) (string, error)
func LoadConfig(path string) (Config, error)
func (c Config) Validate() error
func (c DBConfig) ConnString() string
```

`Validate` must require `database.host`, `database.port`, `database.name`, `database.user`, and `database.password`. `ConnString` must include `search_path=transactions` and default `PoolSize` to `5` when zero.

- [ ] **Step 3: Verify**

Run:

```bash
go test ./internal/cli -run 'TestResolveConfigPath|TestLoadConfig'
```

Expected: tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/cli/config.go internal/cli/config_test.go
git commit -m "feat: add CLI config loading"
```

## Task 2: Schema and sqlc Queries

**Files:**
- Create: `db/migrations/20260505000000_cli_identity_and_soft_delete.sql`
- Modify: `internal/database/query.sql`
- Modify: `internal/database/sqlc/models.go`
- Modify: `internal/database/sqlc/query.sql.go`
- Modify: `internal/database/sqlc/db.go`

- [ ] **Step 1: Add migration**

Create migration with this up/down shape:

```sql
-- migrate:up
ALTER TABLE users ALTER COLUMN discord_id DROP NOT NULL;
ALTER TABLE users ADD COLUMN telegram_id VARCHAR;
ALTER TABLE users ADD COLUMN phone_number VARCHAR;
ALTER TABLE users ADD COLUMN whatsapp_id VARCHAR;

CREATE UNIQUE INDEX idx_users_discord_id_unique_not_null ON users(discord_id) WHERE discord_id IS NOT NULL;
CREATE UNIQUE INDEX idx_users_telegram_id_unique_not_null ON users(telegram_id) WHERE telegram_id IS NOT NULL;
CREATE UNIQUE INDEX idx_users_phone_number_unique_not_null ON users(phone_number) WHERE phone_number IS NOT NULL;
CREATE UNIQUE INDEX idx_users_whatsapp_id_unique_not_null ON users(whatsapp_id) WHERE whatsapp_id IS NOT NULL;

ALTER TABLE transaction ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE transaction ADD COLUMN deleted_by_user_id BIGINT REFERENCES users(id);
ALTER TABLE transaction ADD COLUMN delete_reason TEXT;
CREATE INDEX idx_transaction_deleted_at ON transaction(deleted_at);

-- migrate:down
DROP INDEX IF EXISTS idx_transaction_deleted_at;
ALTER TABLE transaction DROP COLUMN IF EXISTS delete_reason;
ALTER TABLE transaction DROP COLUMN IF EXISTS deleted_by_user_id;
ALTER TABLE transaction DROP COLUMN IF EXISTS deleted_at;

DROP INDEX IF EXISTS idx_users_whatsapp_id_unique_not_null;
DROP INDEX IF EXISTS idx_users_phone_number_unique_not_null;
DROP INDEX IF EXISTS idx_users_telegram_id_unique_not_null;
DROP INDEX IF EXISTS idx_users_discord_id_unique_not_null;
ALTER TABLE users DROP COLUMN IF EXISTS whatsapp_id;
ALTER TABLE users DROP COLUMN IF EXISTS phone_number;
ALTER TABLE users DROP COLUMN IF EXISTS telegram_id;
ALTER TABLE users ALTER COLUMN discord_id SET NOT NULL;
```

Before applying, check the initial migration’s existing `discord_id` unique constraint/index name. If sqlc/dbmate reports duplicate index names, adjust the new Discord index name only; do not change the spec semantics.

- [ ] **Step 2: Add user and household sqlc queries**

Add queries for:

```text
CreateUser
UpdateUser
GetUserById
ListUsers
GetUserByDiscordId
GetUserByTelegramId
GetUserByPhoneNumber
GetUserByWhatsappId
GetHouseholdById
GetHouseholdByGuildId
GetHouseholdByName
ListHouseholds
GetHouseholdUsers
```

`UpdateUser` must use boolean `set_*` args so fields can be unchanged, set to value, or cleared to null.

- [ ] **Step 3: Add transaction sqlc queries**

Add or replace transaction read/write queries for:

```text
GetTransactionByIdActive
GetTransactionsByIdActive
ListTransactions
SoftDeleteTransactionsById
RestoreTransactionsById
```

`ListTransactions` must support filters from the spec and use explicit CASE ordering for the allowed sort/order values instead of string-concatenated SQL.

- [ ] **Step 4: Regenerate sqlc**

Run:

```bash
sqlc generate
```

Expected: generated Go files update successfully.

- [ ] **Step 5: Verify generated fields**

Run:

```bash
rg "TelegramID|PhoneNumber|WhatsappID|DeletedAt|DeleteReason" internal/database/sqlc
```

Expected: generated models include nullable fields for the new columns.

- [ ] **Step 6: Commit**

```bash
git add db/migrations/20260505000000_cli_identity_and_soft_delete.sql internal/database/query.sql internal/database/sqlc
git commit -m "feat: add CLI database queries"
```

## Task 3: Application Error and Identity Layer

**Files:**
- Create: `internal/app/errors.go`
- Create: `internal/app/identity.go`
- Create: `internal/app/identity_test.go`

- [ ] **Step 1: Write identity tests**

Cover these cases in `internal/app/identity_test.go`:

```go
func TestIdentitySelectorRequiresExactlyOneField(t *testing.T) {}
func TestIdentitySelectorAcceptsAuthorID(t *testing.T) {}
func TestIdentitySelectorNormalizesTelegramSenderID(t *testing.T) {}
```

Expected:

```text
zero selector fields -> validation_error
two selector fields -> validation_error
"123456|rafael" normalizes to "123456" for telegram ID
```

- [ ] **Step 2: Implement app errors**

Create `internal/app/errors.go`:

```go
type ErrorCode string

const (
	CodeValidationError     ErrorCode = "validation_error"
	CodeUserNotFound        ErrorCode = "user_not_found"
	CodeTransactionNotFound ErrorCode = "transaction_not_found"
	CodeDuplicateTransaction ErrorCode = "duplicate_transaction"
	CodeDatabaseError       ErrorCode = "database_error"
)

type AppError struct {
	Code    ErrorCode
	Message string
	Err     error
}
```

Implement `Error()`, `Unwrap()`, and `NewError(code ErrorCode, message string, err error) error`.

- [ ] **Step 3: Implement identity selector**

Create `internal/app/identity.go`:

```go
type IdentitySelector struct {
	AuthorID         *int64
	DiscordID        *string
	TelegramID       *string
	PhoneNumber      *string
	WhatsappID       *string
}
```

Implement:

```go
func (s IdentitySelector) ValidateExactlyOne() error
func (s IdentitySelector) Normalized() IdentitySelector
```

Add a short comment inside `Normalized()` before the Telegram `|username` handling. The comment should explain that Nanobot can fall back to sender IDs like `123456789|rafael`, while the database stores the stable numeric Telegram user ID.

- [ ] **Step 4: Verify**

Run:

```bash
go test ./internal/app -run 'TestIdentitySelector'
```

Expected: tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/app/errors.go internal/app/identity.go internal/app/identity_test.go
git commit -m "feat: add app identity validation"
```

## Task 4: Application Services

**Files:**
- Create: `internal/app/service.go`
- Create: `internal/app/users.go`
- Create: `internal/app/users_test.go`
- Create: `internal/app/households.go`
- Create: `internal/app/transactions.go`
- Create: `internal/app/transactions_test.go`
- Modify: `internal/transaction/transaction_service.go`
- Modify: `internal/transaction/types.go`

- [ ] **Step 1: Add repository interfaces**

Create `internal/app/service.go` with small interfaces implemented by sqlc:

```go
type Repository interface {
	UserRepository
	HouseholdRepository
	TransactionRepository
}

type Service struct {
	repo Repository
	transactions TransactionService
}
```

The `TransactionService` interface should expose existing create/update/get behavior plus new soft-delete/restore behavior, so tests can use fakes.

- [ ] **Step 2: Write user service tests**

Cover:

```go
func TestCreateUserRejectsEmptyName(t *testing.T) {}
func TestUpdateUserCanClearPhoneNumber(t *testing.T) {}
func TestResolveUserByTelegramID(t *testing.T) {}
```

Expected: no DB required; use a fake repository.

- [ ] **Step 3: Implement user service**

Implement:

```go
func (s *Service) CreateUser(ctx context.Context, req CreateUserRequest) (UserDTO, error)
func (s *Service) UpdateUser(ctx context.Context, req UpdateUserRequest) (UserDTO, error)
func (s *Service) GetUser(ctx context.Context, id int64) (UserDTO, error)
func (s *Service) ResolveUser(ctx context.Context, selector IdentitySelector) (UserDTO, error)
func (s *Service) ListUsers(ctx context.Context) ([]UserDTO, error)
```

Clears must be explicit booleans: `ClearDiscordID`, `ClearTelegramID`, `ClearPhoneNumber`, `ClearWhatsappID`.

- [ ] **Step 4: Implement household service**

Implement:

```go
func (s *Service) GetHousehold(ctx context.Context, req GetHouseholdRequest) (HouseholdDTO, error)
func (s *Service) ListHouseholds(ctx context.Context) ([]HouseholdDTO, error)
func (s *Service) GetHouseholdUsers(ctx context.Context, householdID int64) ([]UserDTO, error)
```

`GetHouseholdRequest` accepts exactly one of `ID`, `GuildID`, or `Name`.

- [ ] **Step 5: Write transaction service tests**

Cover:

```go
func TestCreateTransactionResolvesAuthorAndRequiresHousehold(t *testing.T) {}
func TestBulkCreateReturnsPartialFailure(t *testing.T) {}
func TestListTransactionsDefaultsToDateDesc(t *testing.T) {}
func TestDeleteTransactionsReturnsDeletedIDs(t *testing.T) {}
```

Expected: no DB required; fake identity/user lookups and transaction service responses.

- [ ] **Step 6: Implement transaction app service**

Implement:

```go
func (s *Service) CreateTransaction(ctx context.Context, req CreateTransactionRequest) WriteResult
func (s *Service) CreateTransactions(ctx context.Context, req BulkCreateTransactionsRequest) WriteResult
func (s *Service) UpdateTransaction(ctx context.Context, req UpdateTransactionRequest) WriteResult
func (s *Service) UpdateTransactions(ctx context.Context, req BulkUpdateTransactionsRequest) WriteResult
func (s *Service) GetTransactions(ctx context.Context, ids []int64, includeDeleted bool) ([]TransactionDTO, error)
func (s *Service) ListTransactions(ctx context.Context, req ListTransactionsRequest) ([]TransactionDTO, error)
func (s *Service) DeleteTransactions(ctx context.Context, req DeleteTransactionsRequest) WriteResult
func (s *Service) RestoreTransactions(ctx context.Context, req RestoreTransactionsRequest) WriteResult
```

Write results must use:

```go
type WriteResult struct {
	CreatedIDs  []int64      `json:"createdIds"`
	UpdatedIDs  []int64      `json:"updatedIds"`
	DeletedIDs  []int64      `json:"deletedIds"`
	RestoredIDs []int64      `json:"restoredIds"`
	Errors      []WriteError `json:"errors"`
}
```

- [ ] **Step 7: Add soft-delete methods to existing transaction service**

Add:

```go
func (ts *TransactionService) SoftDeleteTransactionsById(ctx context.Context, ids []int64, deletedByUserID int64, reason *string) TransactionResult
func (ts *TransactionService) RestoreTransactionsById(ctx context.Context, ids []int64, restoredByUserID int64) TransactionResult
```

Use generated sqlc queries; do not hard delete rows.

- [ ] **Step 8: Verify**

Run:

```bash
go test ./internal/app ./internal/transaction
```

Expected: tests pass.

- [ ] **Step 9: Commit**

```bash
git add internal/app internal/transaction
git commit -m "feat: add CLI application services"
```

## Task 5: CLI Rendering

**Files:**
- Create: `internal/cli/render.go`
- Create: `internal/cli/render_test.go`

- [ ] **Step 1: Write renderer tests**

Cover:

```go
func TestRenderSingleTransactionCompact(t *testing.T) {}
func TestRenderTransactionsCSVUsesStableColumns(t *testing.T) {}
func TestRenderWriteResultJSON(t *testing.T) {}
```

Expected CSV header:

```text
id,amount,transaction_date,author_id,author_name,household_id,household_name,description,notes,created_at,deleted_at
```

- [ ] **Step 2: Implement renderers**

Implement:

```go
func RenderJSON(w io.Writer, value any) error
func RenderTransactionCompact(w io.Writer, tx app.TransactionDTO) error
func RenderTransactionsCSV(w io.Writer, txs []app.TransactionDTO) error
```

Compact text must match the spec shape:

```text
Transaction #101
Amount: 42.50
Date: 2026-05-05 14:30
Author: Rafael
Household: Home
Description: Groceries
Notes: Costco
```

- [ ] **Step 3: Verify**

Run:

```bash
go test ./internal/cli -run 'TestRender'
```

Expected: tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/cli/render.go internal/cli/render_test.go
git commit -m "feat: add CLI renderers"
```

## Task 6: Kong CLI Commands and Entrypoint

**Files:**
- Create: `internal/cli/commands.go`
- Create: `internal/cli/commands_test.go`
- Create: `cmd/voltr-finance/main.go`
- Modify: `internal/database/conn.go`
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Add Kong dependency**

Run:

```bash
go get github.com/alecthomas/kong@latest
```

Expected: `go.mod` includes `github.com/alecthomas/kong`.

- [ ] **Step 2: Add DB connection helper**

Add to `internal/database/conn.go`:

```go
func NewPool(ctx context.Context, connString string) (*pgxpool.Pool, error)
```

Keep existing `Init` and `InitReadOnly` unchanged for the bot.

- [ ] **Step 3: Write Kong command tests**

Cover:

```go
func TestKongTransactionsCreate(t *testing.T) {}
func TestKongTransactionsListCSV(t *testing.T) {}
func TestKongUsersResolveTelegram(t *testing.T) {}
func TestKongHouseholdsGetByName(t *testing.T) {}
```

Expected parsed commands call the fake app service with the intended request DTOs and write output to the injected writer.

- [ ] **Step 4: Implement Kong command dispatch**

Implement:

```go
func Run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, svc AppService) int
```

Use a typed Kong command tree:

```go
type CLI struct {
	Transactions TransactionsCmd `cmd:"" help:"Manage transactions."`
	Users        UsersCmd        `cmd:"" help:"Manage users."`
	Households   HouseholdsCmd   `cmd:"" help:"Read households."`
}
```

Parse with `kong.Parse(&cli, kong.Name("voltr-finance"), kong.Exit(func(int) {}), kong.Writers(stdout, stderr))`, then call `ctx.Run(runContext)` where `runContext` carries `context.Context`, `stdin`, `stdout`, `stderr`, and the app service.

Return exit codes:

```text
0 full success
1 unexpected runtime error
2 validation, not found, duplicate, or partial bulk failure
```

- [ ] **Step 5: Implement transactions commands**

Commands:

```text
transactions create
transactions create-bulk
transactions update
transactions update-bulk
transactions get
transactions list
transactions delete
transactions restore
```

Rules:

```text
single create/update use flags
bulk create/update read JSON from --input or stdin
delete accepts --ids, --reason, --deleted-by-user-id
restore accepts --ids, --restored-by-user-id
list defaults --format json
```

- [ ] **Step 6: Implement users and households commands**

Commands:

```text
users create
users update
users get
users resolve
users list
households get
households list
households users
```

`households get` accepts exactly one of `--id`, `--guild-id`, or `--name`.

- [ ] **Step 7: Implement main entrypoint**

Create `cmd/voltr-finance/main.go` to:

```text
parse --config before command dispatch
load JSON config
open pgx pool with config DB connection string
create sqlc repository
create transaction service
create app service
call cli.Run
os.Exit(code)
```

- [ ] **Step 8: Verify**

Run:

```bash
go test ./internal/cli
go build -o /tmp/voltr-finance ./cmd/voltr-finance
```

Expected: tests pass and binary builds.

- [ ] **Step 9: Commit**

```bash
git add go.mod go.sum internal/cli/commands.go internal/cli/commands_test.go cmd/voltr-finance/main.go internal/database/conn.go
git commit -m "feat: add voltr-finance CLI"
```

## Task 7: Manual Database Verification

**Files:**
- No file changes expected unless verification finds defects.

- [ ] **Step 1: Prepare database and CLI config**

With a running Postgres database, apply migrations, create a temporary JSON config pointing at that database, and build `/tmp/voltr-finance`.

- [ ] **Step 2: Run CLI checks**

```bash
/tmp/voltr-finance --config /tmp/voltr-finance-test.json users create --name "CLI Tester" --telegram-id "123456789"
/tmp/voltr-finance --config /tmp/voltr-finance-test.json users resolve --telegram-id "123456789"
/tmp/voltr-finance --config /tmp/voltr-finance-test.json households get --name "Home"
/tmp/voltr-finance --config /tmp/voltr-finance-test.json transactions create --amount 42.50 --transaction-date 2026-05-05T14:30:00-04:00 --description "Manual CLI test" --author-telegram-id "123456789" --household-id 1
/tmp/voltr-finance --config /tmp/voltr-finance-test.json transactions list --search "Manual CLI test" --sort transaction_date --order desc
/tmp/voltr-finance --config /tmp/voltr-finance-test.json transactions list --format csv --search "Manual CLI test"
/tmp/voltr-finance --config /tmp/voltr-finance-test.json transactions update --id <created-id> --amount 43.00 --description "Manual CLI test updated" --author-telegram-id "123456789" --household-id 1
/tmp/voltr-finance --config /tmp/voltr-finance-test.json transactions delete --ids <created-id> --reason "manual verification" --deleted-by-user-id <created-user-id>
/tmp/voltr-finance --config /tmp/voltr-finance-test.json transactions list --search "Manual CLI test updated"
/tmp/voltr-finance --config /tmp/voltr-finance-test.json transactions list --only-deleted --search "Manual CLI test updated"
/tmp/voltr-finance --config /tmp/voltr-finance-test.json transactions restore --ids <created-id> --restored-by-user-id <created-user-id>
```

Expected:

```text
user create/resolve returns the same generated user ID
household lookup by name returns the expected household ID
transaction create returns a created ID
JSON list and CSV list include the created transaction before delete
update changes the transaction amount/description
default list excludes the soft-deleted transaction
only-deleted list includes the soft-deleted transaction with deleted_at
restore makes the transaction visible in default list again
```

- [ ] **Step 3: Run direct SQL checks**

```sql
SELECT id, name, telegram_id FROM transactions.users WHERE telegram_id = '123456789';
SELECT id, amount, description, deleted_at, deleted_by_user_id, delete_reason FROM transactions.transaction WHERE description LIKE 'Manual CLI test%';
SELECT indexname, indexdef FROM pg_indexes WHERE schemaname = 'transactions' AND tablename = 'users' AND indexdef ILIKE '%telegram_id%';
```

Expected: user identity is stored once, transaction soft-delete fields match CLI actions, and the Telegram unique index exists.

- [ ] **Step 4: Commit fixes if verification found defects**

If verification required fixes:

```bash
git add .
git commit -m "fix: address manual CLI verification issues"
```

## Task 8: Build and Documentation

**Files:**
- Modify: `Dockerfile`
- Create: `docs/cli.md`

- [ ] **Step 1: Add Docker CLI target**

Modify `Dockerfile` so the existing bot final target remains compatible and add a target that builds the CLI binary:

```dockerfile
FROM base AS cli-build
COPY cmd ./cmd
COPY internal ./internal
RUN go build -o voltr-finance ./cmd/voltr-finance

FROM alpine:3.22 AS cli
COPY --from=cli-build /app/voltr-finance /usr/local/bin/voltr-finance
ENTRYPOINT ["voltr-finance"]
```

- [ ] **Step 2: Add CLI docs**

Create `docs/cli.md` with:

```text
config path lookup
sample config.json
install/build command
transaction create/list/delete examples
user resolve example
household lookup by name example
Nanobot mapping examples for Discord, Telegram, WhatsApp
```

Sample config:

```json
{
  "database": {
    "host": "localhost",
    "port": "5432",
    "name": "voltr_finance",
    "user": "voltr_cli_rw",
    "password": "change-me",
    "poolSize": 5
  }
}
```

- [ ] **Step 3: Verify builds**

Run:

```bash
go test ./...
go build ./cmd/main.go
go build -o /tmp/voltr-finance ./cmd/voltr-finance
docker build --target cli -t voltr-finance-cli .
```

Expected: Go tests/builds pass; Docker CLI image builds.

- [ ] **Step 4: Commit**

```bash
git add Dockerfile docs/cli.md
git commit -m "docs: add CLI build and usage docs"
```

## Task 9: Final Verification

**Files:**
- Inspect all modified files.

- [ ] **Step 1: Run sqlc**

```bash
sqlc generate
```

Expected: no generated diff after running.

- [ ] **Step 2: Run tests**

```bash
go test ./...
```

Expected: all tests pass.

- [ ] **Step 3: Build both binaries**

```bash
go build ./cmd/main.go
go build -o /tmp/voltr-finance ./cmd/voltr-finance
```

Expected: both builds pass.

- [ ] **Step 4: Review diff**

```bash
git diff --stat
git diff --check
```

Expected: no whitespace errors; diff is scoped to CLI repurpose work.

- [ ] **Step 5: Final commit if needed**

If verification required fixes:

```bash
git add .
git commit -m "chore: finalize CLI repurpose"
```

## Assumptions and Defaults

- Skip automated DB integration tests for v1, but perform manual CLI verification against a running database before completion.
- Use Kong for command parsing; do not add Cobra.
- Use JSON config at `$HOME/.config/voltr-finance/config.json`.
- No HTTP API, MCP server, LLM-backed query, arbitrary SQL command, user delete, or household mutation in v1.
- The existing bot remains available through `cmd/main.go` and the existing Docker final target.
