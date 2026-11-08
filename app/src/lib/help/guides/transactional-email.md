# Transactional email from your app

Transactional email is mail one person needs because of something they did: a receipt, a sign-in link, a password reset. Your app, website or online store asks Outlet to send it, through Outlet's API or over SMTP. This guide covers templates, the API, SMTP and the brand's API key.

Transactional mail goes out through the pathway that carries **Transactional** in **Settings**, **Sending**. Keep it apart from marketing so a newsletter problem never delays a receipt. See [How sending works](/help/how-sending-works).

## Your brand's API key

Every request from your app carries the brand's API key.

1. Open the brand's **Settings**, **General**.
2. Under **API Key**, press the copy button.
3. Give it to whoever builds your app. Treat it like a password.

**Regenerate Key** makes a new key. The old one stops working at once, so update your apps the same day.

This key is only for sending and subscribing. AI assistants use MCP keys instead; see [Connect an AI assistant over MCP, and API keys](/help/mcp-and-api-keys).

## Create a template

Templates let you change an email's wording without touching your app.

1. Open **Transactional** and press **New Template**.
2. **Name**: for your own reference.
3. **Slug**: the short name your app uses to pick this email, such as `password-reset`.
4. **Subject**: it can hold variables, such as `Reset your password, {{name}}`.
5. Press **Create and edit**.
6. Write the **HTML body**, and the **Plain-text version** (shown by mail apps that don't display HTML; it helps delivery).
7. Use `{{variable}}` wherever your app supplies a value, such as `{{reset_link}}`. Fill in **Sample values** to see them in the **Preview**.
8. Press **Save changes**.
9. Under **Send a test**, press **Send test**. It sends the saved version.

Empty sender fields use the brand's settings. A template switched off shows "Inactive: sends are refused".

## Send through the API

Your developer sends a `POST` to `https://your-outlet-address/sdk/v1/emails` with the header `X-API-Key: your brand API key` and a JSON body:

```json
{
  "to": "jane@example.com",
  "template_slug": "password-reset",
  "variables": { "name": "Jane", "reset_link": "https://..." }
}
```

Without a template, send `subject` and `body` (HTML) instead. Other fields: `text_body`, `from_name`, `from_email`, `reply_to`, `tags`, `meta` and `attachments`.

The answer has a `message_id` and a `status`: `queued`, `sent`, `failed`, `suppressed` (the person bounced, complained or is on the suppression list, so it wasn't sent) or `limited` (over the pathway's limit; try again later).

Outlet's **Settings** (on the **Brands** screen, under your initial at the top right), **API** tab, shows the base address and code examples in cURL, Node.js, Python, Go, PHP and SMTP.

## Send over SMTP

Many apps and plugins can already send email over SMTP. Outlet can receive it and send it on.

**This needs the person who runs your Outlet server**: SMTP is off until `SMTP_ENABLED=true` is set on the server.

Then, in your app's email settings:

| Setting | Value |
| --- | --- |
| Host | your Outlet address |
| Port | 587 (or the server's `SMTP_PORT`) |
| Encryption | STARTTLS |
| Username | the brand's slug, such as `acme` |
| Password | the brand's API key |

The brand's **Settings**, **SMTP** tab shows these for your brand. Messages can be up to 25 MB with up to 100 recipients. The brand's From address is used, whatever From your app sets. Mail sent over SMTP is treated as transactional.

## Common problems

- **"Missing API key"** or **"Invalid API key"**: the `X-API-Key` header is missing or the key was regenerated.
- **"Template '(slug)' not found"** or **"is inactive"**: check the slug on the **Transactional** page, and that the template is active.
- **"Subject is required when not using a template"**: send `template_slug`, or both `subject` and `body`.
- **Status `suppressed`**: the person bounced, complained or is on your suppression list. Check **Blocklist**.
- **SMTP "invalid credentials"**: the username must be the slug of the same brand the API key belongs to.
- **Nothing arrives**: see [Troubleshooting](/help/troubleshooting).
