# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Outlet is a self-hosted email platform (marketing + transactional) built with go-zero framework and SvelteKit 5. Single binary deployment with embedded frontend, SQLite database (WAL mode), and MCP (Model Context Protocol) integration for AI assistants.

**Module**: `github.com/outlet-sh/outlet`

## Commands

### Backend (Go)
```bash
air               # Run with hot reload (recommended for development)
make build        # Build main binary to bin/outlet
make run          # Build and run the binary
make test         # Run Go tests
make gen          # Regenerate API code from outlet.api (NEVER run goctl directly)
make sqlc-gen     # Generate type-safe Go code from SQL queries
make gen-sdk      # Generate SDK clients (TypeScript, Python, Go, PHP)
make proto-gen    # Generate gRPC code from proto files
make gen-all      # All code generation (API + sqlc + proto)
```

### Frontend (SvelteKit)
```bash
cd app
pnpm dev          # Start dev server (port 5173, proxies API to backend)
pnpm build        # Build static site to build/
pnpm check        # Type check
pnpm test         # Run tests with Vitest
```

### Running Tests
```bash
go test ./...                           # All Go tests
go test ./internal/services/emailval/   # Single package
go test -run TestSyntax ./internal/...  # Single test by name
cd app && pnpm test                     # Frontend tests
```

## Architecture

### Go Backend Structure
```
outlet.go              # Main entry point - embedded config, HTTP/HTTPS servers
outlet.api             # API definition file - defines all routes and types
internal/
├── config/            # Configuration structs (loaded from etc/outlet.yaml)
├── handler/           # Auto-generated request handlers (DO NOT EDIT)
├── logic/             # Business logic implementations
│   ├── admin/         # Admin dashboard endpoints
│   ├── auth/          # Authentication logic
│   ├── public/        # Public endpoints (signup, confirm)
│   └── sdk/           # SDK/API endpoints for external use
├── types/             # Auto-generated request/response types (DO NOT EDIT)
├── svc/               # ServiceContext - dependency injection container
├── db/                # Database layer
│   ├── migrations/    # Embedded SQL migrations (auto-run on startup)
│   ├── queries/       # SQL query files for sqlc
│   └── *.sql.go       # sqlc-generated query code
├── services/          # Domain services
│   ├── email/         # Email sending (SMTP, AWS SES)
│   ├── crypto/        # Credential encryption
│   ├── emailval/      # Email validation
│   ├── tracking/      # Email open/click tracking
│   └── webhook/       # Outbound webhook dispatcher
├── mcp/               # MCP server for AI integrations
│   ├── tools/         # MCP tool implementations (brand, campaigns, contacts, etc.)
│   ├── mcpauth/       # MCP authentication (API key + OAuth)
│   ├── mcpctx/        # MCP request context
│   └── oauth/         # OAuth 2.0 handler for MCP
├── middleware/        # Auth, rate limiting, API key validation
├── events/            # Event bus for internal pub/sub
├── workers/           # Background workers (email sending, retries)
├── errorx/            # Custom error types with JSON responses
└── smtp/              # SMTP ingress server
```

### Frontend Structure
See `app/CLAUDE.md` for detailed frontend guidance.

### Key Patterns

**API Code Generation**: The `outlet.api` file defines all routes and types. Running `make gen` generates:
- `internal/handler/` - HTTP handlers (DO NOT EDIT)
- `internal/types/` - Request/response structs (DO NOT EDIT)
- `app/src/lib/api/generate/` - TypeScript client

**Database Layer**: sqlc generates type-safe Go from SQL:
- Write queries in `internal/db/queries/*.sql`
- Run `make sqlc-gen` to regenerate `internal/db/*.sql.go`
- Migrations in `internal/db/migrations/` auto-run on startup (numbered SQL files)

**ServiceContext**: All services are initialized in `internal/svc/servicecontext.go` and passed to handlers via dependency injection.

**Error Handling**: All API errors return JSON via `internal/errorx/errorx.go`. Use `errorx.NewBadRequestError()`, `errorx.NewNotFoundError()`, etc.

**MCP Tools**: MCP tools are implemented in `internal/mcp/tools/`. Each tool file (brand.go, campaigns.go, contacts.go, etc.) registers handlers in `registry.go`.

**Production Mode**: When `ProductionMode: true` in config:
- Serves Let's Encrypt HTTPS on :443
- Embeds the SvelteKit static build
- Redirects www → non-www, HTTP → HTTPS

**Development Mode**: When `ProductionMode: false`:
- Go backend on port 8888 (use `air` for hot reload)
- Frontend dev server on port 5173 (run separately with `cd app && pnpm dev`)
- With Docker: backend mapped to port 20202 externally (Vite proxies to this)

**SMTP Ingress Server**: Optional SMTP server for receiving emails via standard SMTP protocol:
- Enabled via `SMTP_ENABLED=true` environment variable
- Authenticates with brand API keys (`brand-slug:api-key`)
- Configured in `internal/smtp/` (backend.go, server.go, processor.go)

## Critical Rules

1. **NEVER run goctl commands directly** - Always use `make gen` to regenerate API code
2. **Use pnpm** for all frontend package management
3. **Hot reload available** - Run `air` for Go hot reloading (no restart needed)
4. **Always build before pushing** - Run `make build` and `cd app && pnpm build`
5. **Idiomatic Go only** - One function with parameters, not multiple function variants
6. **Styles in app.css only** - No inline styles or `<style>` blocks in Svelte files
7. **Minimal changes** - Only modify code directly related to the task
8. **Never assume code is unused** - Code may be called from frontend, other services, or future features
9. **Always return JSON** - All API responses must be valid JSON, including errors
10. **Svelte 5 ONLY** - This is a Svelte 5 project. NEVER use Svelte 4 syntax.
11. **Fork pull requests** - Never approve a workflow run for a pull request from outside the organization (a fork). Only the owner approves those, after reading the change. Agents never approve them, even to get a pull request green.

## Svelte 5 Syntax (IMPORTANT)

This project uses **Svelte 5 with runes**. Do NOT use Svelte 4 patterns.

```svelte
<!-- CORRECT: Svelte 5 -->
<script lang="ts">
  let { data, onchange } = $props();     // Props via $props()
  let count = $state(0);                  // Reactive state via $state()
  let doubled = $derived(count * 2);      // Computed via $derived()

  $effect(() => {                         // Side effects via $effect()
    console.log(count);
  });
</script>

<!-- WRONG: Svelte 4 (DO NOT USE) -->
<script lang="ts">
  export let data;                        // NO: old prop syntax
  let count = 0;                          // NO: not reactive
  $: doubled = count * 2;                 // NO: reactive statements
</script>
```

**Key differences:**
- Props: `let { prop } = $props()` NOT `export let prop`
- State: `let x = $state(0)` NOT `let x = 0`
- Computed: `let y = $derived(x * 2)` NOT `$: y = x * 2`
- Effects: `$effect(() => {})` NOT `$: { }`
- Event handlers: Pass functions as props, not `createEventDispatcher`
- Slots: `{@render children?.()}` NOT `<slot />`

## Generated API Types (CRITICAL)

TypeScript client generated from `outlet.api` uses **snake_case** (matching Go/JSON):

```typescript
// WRONG - assumes camelCase
stats.withStripe;
customer.createdAt;
api.adminListCustomers({ pageSize: 10 });

// CORRECT - matches generated types
stats.with_stripe;
customer.created_at;
api.adminListCustomers({ page_size: 10 });
```

Always check `app/src/lib/api/generate/outletComponents.ts` for exact property names.

## Multi-Tenant Architecture

Outlet uses a **brand-based multi-tenancy** model:
- Each brand (organization) has its own API key, contacts, lists, campaigns
- Routes are scoped by brand: `/[brandSlug]/campaigns`, `/[brandSlug]/settings`
- API keys are per-brand and used for both REST API and SMTP authentication
- Brand context is set via URL slug, not global state

## Environment Configuration

Configuration loaded from `etc/outlet.yaml` or environment variables. Key settings:
- `DATABASE_PATH` / `Database.Path` - SQLite database path (default: `./data/outlet.db`)
- `JWT_SECRET` / `Auth.AccessSecret` - JWT access token secret
- `JWT_REFRESH_SECRET` / `Auth.RefreshSecret` - JWT refresh token secret
- `ENCRYPTION_KEY` / `Encryption.Key` - 32-byte hex key for credential encryption
- `PRODUCTION_MODE` / `App.ProductionMode` - Enable HTTPS/Let's Encrypt
- `APP_DOMAIN` / `App.Domain` - Domain for production
- `SMTP_ENABLED` - Enable SMTP ingress server
- `SMTP_PORT` - SMTP port (default: 587)
- `SMTP_ALLOW_INSECURE_AUTH` - Allow AUTH without TLS (testing only)

go-zero config uses struct tags:
- `json:",optional"` - field not required
- `json:",default=value"` - default value if missing

## First Run Setup

On first launch, the app redirects to `/setup` where you:
1. Create the admin account via the onboarding wizard
2. Configure SMTP settings for email sending

No admin credentials need to be set in config files - the setup wizard handles everything.
