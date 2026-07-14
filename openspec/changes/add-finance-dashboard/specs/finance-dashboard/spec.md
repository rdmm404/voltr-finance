## ADDED Requirements

### Requirement: Private monthly dashboard
The system SHALL serve a read-only monthly finance dashboard at the root of the human-facing hostname. The dashboard SHALL use a selected personal owner, household owner, and calendar month without invoking any finance mutation.

#### Scenario: Dashboard opens with configured defaults
- **WHEN** an authenticated UI visitor requests the dashboard without explicit owner selectors
- **THEN** the system renders the configured default personal and household scopes

#### Scenario: Dashboard never creates a missing budget
- **WHEN** the selected scope has no budget for the selected month
- **THEN** the system renders a scope-specific empty state and does not ensure or create a budget

### Requirement: Canonical month navigation
The dashboard SHALL represent the selected period as a `month=YYYY-MM` query parameter and SHALL provide previous-month and next-month navigation. The default current month SHALL be calculated in the process timezone.

#### Scenario: Missing month becomes canonical
- **WHEN** a visitor requests the dashboard without a month parameter
- **THEN** the system redirects to a URL containing the current month calculated from `TZ`

#### Scenario: Explicit month is bookmarkable
- **WHEN** a visitor requests a valid explicit month
- **THEN** the system renders that month and preserves it in owner-selection and month-navigation URLs

#### Scenario: Invalid month is rejected
- **WHEN** a visitor supplies a month that is not in `YYYY-MM` form or is not a valid calendar month
- **THEN** the system returns a safe validation page with HTTP status `400`

### Requirement: Configurable and selectable report owners
The dashboard SHALL use configured positive default user and household IDs and SHALL allow either owner to be overridden with bookmarkable query parameters selected from available users and households.

#### Scenario: Owner override is applied
- **WHEN** a visitor selects another existing user or household
- **THEN** the resulting GET request renders that owner's report and retains the selection in the URL

#### Scenario: Missing explicit owner is not replaced silently
- **WHEN** a visitor explicitly selects an owner ID that does not exist
- **THEN** the system returns a safe not-found page rather than falling back to the configured owner

### Requirement: Combined and scope-specific summaries
The dashboard SHALL display a combined monthly summary and separate personal and household summaries. Detailed personal and household reports SHALL remain distinct.

#### Scenario: Both scopes exist
- **WHEN** both selected monthly budgets exist
- **THEN** the dashboard shows their combined headline values, side-by-side scope summaries on wide layouts, and separate full-width detailed reports

#### Scenario: One scope is absent
- **WHEN** exactly one selected monthly budget exists
- **THEN** the combined summary uses only the available scope and the absent scope is identified clearly

#### Scenario: Both scopes are absent
- **WHEN** neither selected monthly budget exists
- **THEN** the dashboard renders a successful monthly empty state

### Requirement: Headline totals include unmapped spending
For each scope and the combined summary, the dashboard SHALL calculate total spending as mapped actual spending plus all unmapped spending and SHALL calculate effective remaining allocation by subtracting that total from allocation. Uncategorized spending SHALL NOT be added separately because it is a subset of unmapped spending.

#### Scenario: Unmapped spending affects remaining amount
- **WHEN** a report has allocated funds, mapped transactions, and unmapped transactions
- **THEN** headline spent and remaining values include both mapped and unmapped transaction amounts

#### Scenario: Unmapped amount is explained
- **WHEN** headline spending contains unmapped spending
- **THEN** the relevant scope summary and detailed report identify the unmapped amount and transactions explicitly

### Requirement: Eager transaction drill-down
The dashboard SHALL render each detailed budget line with its mapped transactions in the initial HTML response using a semantic disclosure control. It SHALL render unmapped transactions in a prominent separate disclosure section.

#### Scenario: Budget line is expanded without another request
- **WHEN** a visitor expands a budget line
- **THEN** its transaction ID, date, CAD amount, description, notes when present, category, and useful author identity are already available without calling `/v1` or making a partial-content request

#### Scenario: Unmapped transactions are distinguishable
- **WHEN** a detailed report contains transactions not mapped to a budget line
- **THEN** the dashboard presents them separately with a warning state and does not place them under a mapped line

### Requirement: CAD monetary presentation
The dashboard SHALL format all monetary values consistently as CAD using centralized presentation logic and SHALL not imply currency conversion or mixed-currency support.

#### Scenario: Amount is rendered
- **WHEN** the dashboard displays an allocation, actual, remaining, or transaction amount
- **THEN** it uses consistent `en-CA` CAD formatting, including negative values

### Requirement: Responsive and accessible presentation
The dashboard SHALL remain usable on current desktop and mobile browsers. Interactive controls SHALL be keyboard operable, visible states SHALL not rely on color alone, and no essential information SHALL require hover.

#### Scenario: Narrow viewport
- **WHEN** the dashboard is viewed on a phone-sized viewport
- **THEN** summaries and detailed sections stack without losing financial values or transaction drill-down controls

#### Scenario: Keyboard disclosure
- **WHEN** a keyboard user focuses and activates a budget-line disclosure
- **THEN** the transaction content opens using native semantic behavior with a visible focus state

### Requirement: Safe complete-page failure behavior
Expected missing budgets SHALL render empty states, while unexpected report failures SHALL prevent rendering potentially misleading partial totals.

#### Scenario: Unexpected scope failure
- **WHEN** either selected detailed report fails for an unexpected internal reason
- **THEN** the system returns a safe complete error page with HTTP status `500` and logs the internal cause server-side

### Requirement: Embedded same-origin assets
The dashboard SHALL serve its compiled styles and icons from embedded same-origin assets and SHALL remain functionally usable without JavaScript. Production rendering SHALL NOT depend on a CDN.

#### Scenario: JavaScript is unavailable
- **WHEN** a visitor loads the dashboard with JavaScript disabled
- **THEN** month navigation, owner selection, report reading, and transaction disclosure remain usable

#### Scenario: Asset is requested
- **WHEN** the browser requests a dashboard asset under `/assets/`
- **THEN** the Go server returns the embedded asset with the correct content type

### Requirement: Separate UI and API deployment boundaries
Production deployment SHALL route the full-access human UI through `finance.homelab.voltr.org` with Traefik BasicAuth and SHALL route only `/v1` through `finance-api.homelab.voltr.org`, where existing bearer authentication remains authoritative. BasicAuth credentials SHALL NOT be committed to source control.

#### Scenario: UI request lacks BasicAuth
- **WHEN** a client requests the human-facing hostname without valid Traefik BasicAuth credentials
- **THEN** Traefik rejects the request before it reaches the application

#### Scenario: API hostname receives a UI path
- **WHEN** a client requests a root-level UI path through the API hostname
- **THEN** the API Traefik router does not expose that path

#### Scenario: API request uses valid bearer authentication
- **WHEN** a client calls `/v1` through the API hostname with the configured bearer key
- **THEN** the existing JSON API behavior remains available without BasicAuth
