# Purpose

Define bearer API-key authentication and credential-safety requirements for the finance API.

## Requirements
### Requirement: Finance API routes require a bearer API key
The system SHALL require every `/v1` request to include the configured API key as an `Authorization: Bearer` credential.

#### Scenario: Correct API key is supplied
- **WHEN** a request to a `/v1` route contains the configured bearer API key
- **THEN** authentication succeeds and request processing continues

#### Scenario: API key is missing
- **WHEN** a request to a `/v1` route has no bearer credential
- **THEN** the API returns HTTP 401 without executing the requested application operation

#### Scenario: API key is malformed or incorrect
- **WHEN** a request to a `/v1` route has a malformed authorization header or a bearer key that does not match the configured key
- **THEN** the API returns HTTP 401 with a generic authentication error

### Requirement: Server requires API-key configuration
The API server SHALL refuse to start when its API key configuration is empty.

#### Scenario: API key is not configured
- **WHEN** the API server starts without a non-empty API key
- **THEN** startup fails before the server accepts requests

### Requirement: API-key handling does not disclose credentials
The system MUST NOT return or log a supplied or configured API key, and key comparison SHALL avoid ordinary data-dependent string comparison.

#### Scenario: Authentication fails
- **WHEN** a client supplies an incorrect API key
- **THEN** logs and the HTTP response identify the authentication failure without containing either the supplied key or configured key

### Requirement: Liveness does not require API authentication
The server SHALL expose a liveness endpoint outside the finance API authentication boundary.

#### Scenario: Unauthenticated liveness request
- **WHEN** a client requests the liveness endpoint without an API key
- **THEN** the server returns its liveness result without granting access to any finance operation
