# Outlet.sh

The modern, self-hosted email platform for indie hackers and creators. Marketing campaigns, transactional emails, and AI-powered automation—all in a single binary.

## Why Outlet.sh?

- **Single binary deployment** — No PHP, no external database. Just download and run.
- **SQLite with WAL mode** — Zero configuration, handles 100k+ subscribers.
- **Marketing + Transactional** — One platform for newsletters and application emails.
- **AI-native MCP integration** — Build email sequences by talking to Claude.
- **Built-in backup system** — One-click download or automated S3 backups.
- **Full source code included** — No obfuscation, modify anything you need.
- **SMTP ingress server** — Drop-in replacement for any SMTP setup. Point Postfix, your app, or any mail client at Outlet.

## Quick Start

### Option 1: Direct Binary

```bash
# Download the binary for your platform
wget https://outlet.sh/download/outlet-linux-amd64
chmod +x outlet-linux-amd64

# Set required environment variables
export JWT_SECRET="your-secret-key-here"
export JWT_REFRESH_SECRET="your-refresh-secret-here"
export ADMIN_EMAIL="admin@example.com"
export ADMIN_PASSWORD="your-admin-password"

# Run it
./outlet-linux-amd64
```

Open `http://localhost:9888` and log in with your admin credentials.

### Option 2: Docker Compose

Create a `.env` file:

```bash
# .env
JWT_SECRET=your-secret-key-here
JWT_REFRESH_SECRET=your-refresh-secret-here
ENCRYPTION_KEY=your-32-byte-hex-key
PRODUCTION_MODE=true
APP_DOMAIN=mail.yourdomain.com
APP_BASE_URL=https://mail.yourdomain.com
```

Create `compose.yaml`:

```yaml
services:
  outlet:
    image: ghcr.io/outlet-sh/outlet:latest
    ports:
      - "443:443"
      - "80:80"
      - "587:587"
    volumes:
      - ./data:/app/data
    env_file: .env
    restart: always
```

```bash
docker compose up -d
```

### Option 3: Systemd Service

Create `/etc/systemd/system/outlet.service`:

```ini
[Unit]
Description=Outlet.sh Email Platform
After=network.target

[Service]
Type=simple
User=outlet
WorkingDirectory=/opt/outlet
ExecStart=/opt/outlet/outlet
Restart=always
RestartSec=5
EnvironmentFile=/opt/outlet/.env

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable outlet
sudo systemctl start outlet
```

## Features

### Subscriber Management
- CSV import/export
- Custom fields and tags
- Segmentation and filtering
- Double opt-in flow
- Automatic bounce/complaint handling
- GDPR compliance tools

### Marketing Campaigns
- Rich text and HTML editor
- Template library
- Broadcast and scheduled campaigns
- RSS-to-email campaigns
- Open/click tracking
- Per-campaign analytics

### Transactional Email API
- REST API for application emails
- Template-based messages with merge fields
- Dedicated sending queue (separate from marketing)
- Per-template analytics
- Webhook callbacks for delivery events
- Auto-generated SDKs (TypeScript, Python, Go, PHP)

### SMTP Ingress Server

Send emails through Outlet using standard SMTP protocol — a **100% drop-in replacement** for any existing SMTP setup. Point your application, Postfix relay, or any mail client at Outlet and get all the platform features automatically.

**Authentication:**
- Username: Your brand slug (e.g., `my-company`)
- Password: Your brand's API key

The username **must** match the brand slug associated with your API key. This enforces brand isolation and prevents cross-brand sending.

**Custom Headers** for advanced control:

| Header | Purpose | Example |
|--------|---------|---------|
| `X-Outlet-List` | Associate with a list | `newsletter` |
| `X-Outlet-Tags` | Comma-separated tags | `welcome,onboarding` |
| `X-Outlet-Template` | Use a template | `order-confirmation` |
| `X-Outlet-Type` | `marketing` or `transactional` | `transactional` |
| `X-Outlet-Track` | `opens,clicks` or `none` | `opens,clicks` |
| `X-Outlet-Meta-*` | Custom metadata | `X-Outlet-Meta-OrderID: 12345` |

**Example with swaks:**

```bash
swaks --to user@example.com \
      --from hello@yourdomain.com \
      --server mail.yourdomain.com:587 \
      --auth PLAIN \
      --auth-user my-brand \
      --auth-password "your-api-key" \
      --tls \
      --header "X-Outlet-List: customers" \
      --header "X-Outlet-Tags: purchase,vip" \
      --body "Thanks for your order!"
```

**Example with Python:**

```python
import smtplib
from email.mime.text import MIMEText

msg = MIMEText("<h1>Welcome!</h1><p>Thanks for signing up.</p>", "html")
msg["Subject"] = "Welcome to our platform"
msg["From"] = "hello@yourdomain.com"
msg["To"] = "user@example.com"
msg["X-Outlet-List"] = "newsletter"
msg["X-Outlet-Tags"] = "welcome,new-user"

with smtplib.SMTP("mail.yourdomain.com", 587) as server:
    server.starttls()
    server.login("my-brand", "your-api-key")
    server.send_message(msg)
```

**Configuration via Environment Variables** (recommended for Docker):

```bash
SMTP_ENABLED=true              # Enable SMTP server
SMTP_PORT=587                  # Standard submission port (default: 587)
SMTP_ALLOW_INSECURE_AUTH=true  # Allow AUTH without TLS (for testing/internal use)
SMTP_TLS_CERT=/path/to/cert.pem   # TLS certificate (optional)
SMTP_TLS_KEY=/path/to/key.pem     # TLS private key (optional)
```

**Docker Compose example with SMTP:**

Create a `.env` file:

```bash
# .env
JWT_SECRET=your-secret-key-here
JWT_REFRESH_SECRET=your-refresh-secret-here
ENCRYPTION_KEY=your-32-byte-hex-key
PRODUCTION_MODE=true
APP_DOMAIN=mail.yourdomain.com
APP_BASE_URL=https://mail.yourdomain.com

# SMTP Configuration
SMTP_ENABLED=true
SMTP_PORT=587
SMTP_ALLOW_INSECURE_AUTH=false
SMTP_TLS_CERT=/app/certs/cert.pem
SMTP_TLS_KEY=/app/certs/key.pem
```

Create `compose.yaml`:

```yaml
services:
  outlet:
    image: ghcr.io/outlet-sh/outlet:latest
    ports:
      - "443:443"
      - "80:80"
      - "587:587"     # SMTP submission port
    volumes:
      - ./data:/app/data
      - ./certs:/app/certs
    env_file: .env
    restart: always
```

**Configuration via YAML** (for direct binary):

```yaml
# In etc/outlet.yaml
SMTP:
  enabled: "true"
  port: "587"
  tlscert: /path/to/cert.pem
  tlskey: /path/to/key.pem
  allowinsecureauth: "false"   # Set to "true" only without TLS
```

**Testing with curl:**

```bash
curl -v --url 'smtp://mail.yourdomain.com:587' \
  --mail-from 'hello@yourdomain.com' \
  --mail-rcpt 'recipient@example.com' \
  --user 'my-brand:your-api-key' \
  --upload-file - <<< "Subject: Test Email

This is a test email sent via Outlet SMTP."
```

**Note:** For production use, always configure TLS certificates. The `SMTP_ALLOW_INSECURE_AUTH=true` option should only be used for testing or when running behind a TLS-terminating proxy.

### Automation
- Autoresponder sequences
- Trigger rules (tag added, link clicked, date-based)
- Transactional-to-marketing handoff

### AI-Powered Email (MCP Integration)

Outlet.sh includes a built-in MCP server, enabling AI assistants like Claude to manage your email operations:

```
You: "Create a 7-day welcome sequence for my SaaS. Day 1 introduces
     the product, Day 3 covers the main feature, Day 5 shares a case
     study, Day 7 offers a discount."

Claude: [Creates sequence, adds emails with proper delays, writes
        copy, sets triggers — all via MCP tools]
```

MCP capabilities include:
- List and subscriber management
- Campaign creation and scheduling
- Sequence building with email content
- Analytics queries and insights
- Transactional email sending

### Backup System

- **Dashboard download** — One-click database backup
- **S3 automated backup** — Scheduled backups with retention policies
- **CLI backup** — `outlet backup` or `outlet backup --s3`
- **Safe live backups** — Uses SQLite backup API (no corruption)
- **Easy restore** — Replace file and restart, or use dashboard

## Configuration

Configuration uses environment variables. Create a `.env` file or set them directly:

### Required

| Variable | Description |
|----------|-------------|
| `JWT_SECRET` | Secret key for access tokens |
| `JWT_REFRESH_SECRET` | Secret key for refresh tokens |
| `ADMIN_EMAIL` | Super admin email (created on first startup) |
| `ADMIN_PASSWORD` | Super admin password |

### Production

| Variable | Description |
|----------|-------------|
| `PRODUCTION_MODE` | Set to `true` for HTTPS/Let's Encrypt |
| `BEHIND_PROXY` | Set to `true` when a load balancer or ingress terminates TLS: plain HTTP on port 8888 with the embedded UI, no Let's Encrypt |
| `APP_DOMAIN` | Your domain (e.g., `mail.yourdomain.com`) |
| `APP_BASE_URL` | Full URL (e.g., `https://mail.yourdomain.com`) |

### Optional

| Variable | Description |
|----------|-------------|
| `DATABASE_PATH` | SQLite path (default: `./data/outlet.db`) |
| `ENCRYPTION_KEY` | 32-byte hex key for credential encryption |
| `ANTHROPIC_API_KEY` | Claude AI integration |
| `STRIPE_SECRET_KEY` | Payment processing |
| `STRIPE_WEBHOOK_SECRET` | Stripe webhook verification |

### SMTP Ingress Server

| Variable | Description |
|----------|-------------|
| `SMTP_ENABLED` | Set to `true` to enable SMTP server |
| `SMTP_PORT` | SMTP port (default: `587`) |
| `SMTP_TLS_CERT` | Path to TLS certificate file |
| `SMTP_TLS_KEY` | Path to TLS private key file |
| `SMTP_ALLOW_INSECURE_AUTH` | Set to `true` to allow AUTH without TLS (testing only) |

## Amazon SES Setup

Outlet.sh works best with Amazon SES for cost-effective sending (~$0.10 per 1,000 emails).

1. **Create SES credentials** in AWS Console → SES → SMTP Settings
2. **Verify your domain** in SES → Verified Identities
3. **Configure in Outlet.sh** dashboard → Settings → Email Provider
4. **Request production access** if still in SES sandbox

## API & SDKs

Outlet.sh provides a REST API with auto-generated clients:

### Transactional Email Example (TypeScript)

```typescript
import { Outlet } from '@outlet/sdk';

const outlet = new Outlet('your-api-key', 'https://mail.yourdomain.com');

await outlet.emails.sendEmail({
  to: 'user@example.com',
  template_slug: 'welcome-email',
  variables: {
    name: 'John',
    activation_link: 'https://...'
  }
});
```

### Transactional Email Example (Go)

```go
import "github.com/outlet-sh/outlet/sdk/go"

client := outlet.NewClient("your-api-key", "https://mail.yourdomain.com")

err := client.Emails.SendEmail(ctx, outlet.SendEmailRequest{
    To:           "user@example.com",
    TemplateSlug: "welcome-email",
    Variables: map[string]string{
        "name":            "John",
        "activation_link": "https://...",
    },
})
```

## Architecture

```
┌───────────────────────────────────────────────────────────────────┐
│                         Single Binary                              │
├───────────────────────────────────────────────────────────────────┤
│  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────────┐   │
│  │ SvelteKit │  │  go-zero  │  │   SMTP    │  │  MCP Server   │   │
│  │ Frontend  │  │  REST API │  │  Ingress  │  │ (AI Integration)│  │
│  └───────────┘  └───────────┘  └───────────┘  └───────────────┘   │
├───────────────────────────────────────────────────────────────────┤
│  ┌───────────────┐  ┌───────────────┐  ┌───────────────────────┐  │
│  │ Email Workers │  │ Rules Engine  │  │  Webhook Dispatcher   │  │
│  └───────────────┘  └───────────────┘  └───────────────────────┘  │
├───────────────────────────────────────────────────────────────────┤
│                       SQLite (WAL Mode)                            │
│                       ./data/outlet.db                             │
└───────────────────────────────────────────────────────────────────┘
```

- **Frontend**: Embedded SvelteKit 5 static build
- **REST API**: go-zero framework with auto-generated handlers
- **SMTP Ingress**: Standard SMTP server (port 587) for drop-in integration
- **Database**: SQLite with WAL mode, auto-migrations on startup
- **Workers**: Background email sending with rate limiting
- **MCP**: Model Context Protocol server for AI assistants

## Development

### Prerequisites

- Go 1.21+
- Node.js 20+
- pnpm

### Setup

```bash
# Clone the repository
git clone https://github.com/outlet-sh/outlet
cd outlet

# Install dependencies
go mod download
cd app && pnpm install && cd ..

# Create .env file
cp .env.example .env
# Edit .env with your values

# Run backend (uses air for hot reload)
make run

# In another terminal, run frontend
cd app && pnpm dev
```

### Build

```bash
# Build Go binary
make build

# Build frontend
cd app && pnpm build

# The binary embeds the frontend automatically
```

### Code Generation

```bash
# After modifying outlet.api
make gen          # Regenerates handlers, types, and TS client

# After modifying SQL queries
make sqlc-gen     # Regenerates type-safe Go from SQL

# All code generation
make gen-all
```

### Testing

```bash
# Go tests
go test ./...
go test ./internal/services/emailval/   # Single package
go test -run TestName ./internal/...    # Single test

# Frontend tests
cd app && pnpm test
```

## License

Outlet is open source under the [AGPL-3.0 License](LICENSE).

- Self-host freely for personal or commercial use
- Modifications must be shared under AGPL if distributed or offered as a service
- Commercial licensing available for companies needing different terms

## Support

- Documentation: https://outlet.sh/docs
- Issues: https://github.com/outlet-sh/outlet/issues
- Email: support@outlet.sh

---

Built with care for indie hackers who value simplicity, ownership, and modern tooling.
