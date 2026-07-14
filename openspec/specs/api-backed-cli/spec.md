# Purpose

Define the CLI as a thin authenticated REST API client with local rendering and consistent exit behavior.

## Requirements
### Requirement: CLI executes finance operations through the REST API
The CLI SHALL call the authenticated REST API for every finance operation and SHALL NOT connect directly to Postgres or initialize server application services.

#### Scenario: CLI command executes successfully
- **WHEN** a user runs a valid finance command with working API configuration
- **THEN** the CLI sends the corresponding authenticated HTTP request and renders the API response

#### Scenario: API is unavailable
- **WHEN** the CLI cannot connect to the configured API within its timeout
- **THEN** the CLI reports a transport failure without attempting a direct database fallback

### Requirement: CLI uses API configuration
The CLI SHALL load an API base URL and API key from its strict configuration, with `VOLTR_API_URL` and `VOLTR_API_KEY` overriding the corresponding file values. Existing explicit, environment, and default config-path resolution SHALL remain available.

#### Scenario: Environment overrides file API settings
- **WHEN** the config file contains API settings and the corresponding API environment variables are non-empty
- **THEN** the CLI uses the environment values for the request

#### Scenario: API configuration is incomplete
- **WHEN** a finance command runs without a valid base URL or API key after configuration resolution
- **THEN** the CLI exits with a configuration error before sending a request

#### Scenario: Help is requested without configuration
- **WHEN** a user requests CLI help without a readable config file
- **THEN** the CLI displays help successfully without attempting configuration loading or an API request

### Requirement: CLI preserves recognizable finance commands and rendering
The CLI SHALL retain recognizable transaction, user, household, category, and budget command workflows and SHALL render supported JSON, compact, and CSV formats locally from API responses.

#### Scenario: Transaction list is rendered as CSV
- **WHEN** a user requests CSV format for a transaction list
- **THEN** the CLI calls the transaction-list endpoint and converts the returned representation to CSV locally

#### Scenario: Default output is requested
- **WHEN** a command does not select another supported format
- **THEN** the CLI emits valid JSON for the operation result

### Requirement: Budget create flag selects the ensure endpoint
The existing `budgets get --create` CLI flag SHALL call the monthly-budget ensure endpoint directly, while the command without `--create` SHALL call the read-only endpoint.

#### Scenario: Create flag is supplied
- **WHEN** a user runs `budgets get` with `--create`
- **THEN** the CLI sends one request to the monthly-budget ensure endpoint without first issuing a read request

#### Scenario: Create flag is absent
- **WHEN** a user runs `budgets get` without `--create`
- **THEN** the CLI sends a read-only monthly-budget request and reports not-found without creating a budget

### Requirement: CLI exposes partial bulk outcomes
The CLI SHALL render the complete bulk response even when one or more items fail so that users can identify both committed successes and failures.

#### Scenario: Bulk command partially succeeds
- **WHEN** the API returns both successful and failed bulk items
- **THEN** the CLI renders both arrays and exits with the operation-failure exit code

### Requirement: CLI exit classes are consistent
The CLI SHALL use exit code `0` for complete operation success, `2` for usage, validation, or item-level operation failure, and `1` for configuration, authentication, transport, server, or unexpected failures.

#### Scenario: Bulk response contains one failed item
- **WHEN** a bulk API request is processed and at least one item fails
- **THEN** the CLI exits with code `2` after rendering the response

#### Scenario: API authentication fails
- **WHEN** the API rejects the configured key
- **THEN** the CLI reports an authentication failure and exits with code `1`
