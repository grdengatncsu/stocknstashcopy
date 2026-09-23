# Pi API JSON Fixtures

These files are canonical examples for the revised target contract in
[`pi-api-contract.md`](../../pi-api-contract.md). Frontend contributors may
import them into mocks and tests. Backend contributors should update fixtures,
contract text, and automated tests together when an agreed format changes.

## Fixtures

| File | Purpose |
|---|---|
| `status-response.json` | Pi health and synchronization status |
| `scan-submit-request.json` | Edge-to-Pi scan report |
| `scan-submit-accepted-response.json` | Newly processed scan acknowledgment |
| `scan-submit-duplicate-response.json` | Idempotent retry acknowledgment |
| `inventory-list-response.json` | Wrapped active-inventory list |
| `inventory-create-request.json` | Manual inventory addition |
| `inventory-create-response.json` | Created inventory record |
| `inventory-update-request.json` | Versioned partial update with explicit nullable clearing |
| `inventory-update-response.json` | Updated inventory record |
| `pending-list-response.json` | Wrapped pending-result list |
| `pending-resolve-request.json` | Versioned pending correction/confirmation |
| `pending-resolve-response.json` | Inventory record produced by resolution |
| `error-invalid-request.json` | Validation failure |
| `error-not-found.json` | Missing record |
| `error-conflict.json` | Stale-version conflict |

`DELETE /api/inventory/{id}` has no JSON success body. The request supplies the
last observed version with an HTTP header such as `If-Match: "4"`; success is
`204 No Content`.

The current local Go backend does not yet match every target fixture. See the
contract's Implementation Gaps table before using these as implementation
claims.
