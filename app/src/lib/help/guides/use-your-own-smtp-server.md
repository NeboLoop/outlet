# Use your own SMTP server

Instead of Amazon SES, Outlet can send through an SMTP relay: another company's sending server, such as SendGrid, Mailgun or Postmark, or your own mail server. SMTP is the standard way mail servers hand email to each other.

This guide is about Outlet sending **out** through a relay. To have your own app send mail **into** Outlet over SMTP, see [Transactional email from your app](/help/transactional-email).

**What needs a person, and why**

- **An account at the relay company**, because it bills you directly.
- **The relay's host, username and password or API key**, because only you can sign in there.
- **The relay's DNS records** at your DNS host. Outlet sets up DNS automatically only for Amazon SES. Follow your relay's own instructions for its SPF and DKIM records.

## For the whole Outlet

Every brand that sends through "This server's email settings" uses this.

1. Open Outlet's own **Settings**: on the **Brands** screen, click your initial at the top right and choose **Settings**.
2. Open the **Integrations** tab, then the **SMTP Relay** tab.
3. Fill in:
   - **SMTP Host**: the relay's server name.
   - **Port**: usually 587.
   - **Username**: from the relay.
   - **Password / API Key**: from the relay.
4. Press **Save SMTP Settings**. Outlet connects to the relay and signs in before saving, so settings that can't send are never saved. A username and password are required: Outlet doesn't send through a relay that takes mail without a login.

**Common providers**

| Provider | SMTP Host | Port | Username |
| --- | --- | --- | --- |
| SendGrid | smtp.sendgrid.net | 587 | `apikey` (the word itself), with your API key as the password |
| Mailgun | smtp.mailgun.org | 587 | Your Mailgun SMTP login |
| Postmark | smtp.postmarkapp.com | 587 | Your server API token |

## For one brand only

Use this when one brand has its own relay account, for example a client's.

1. Open the brand's **Settings**, **Sending**.
2. Press **Add pathway**.
3. **Name**: for example "Client relay".
4. **Sends through**: "Your own SMTP relay".
5. Fill in **SMTP host**, **Port**, **Username** and **Password**. Outlet stores them encrypted.
6. Under **Carries**, tick **Transactional**, **Marketing** or both.
7. Press **Add pathway**.

Ticking a kind here takes it off the brand's other pathway. See [How sending works](/help/how-sending-works).

## Your sending domain

With an SMTP relay, Outlet has nothing to verify at Amazon, so it marks the brand's domain verified straight away. Your relay still needs its own DNS records (usually SPF and DKIM) to deliver well. Add the ones your relay's dashboard lists. [Your sending domain and DNS explained](/help/sending-domain-and-dns) explains what they are.

## Don't use a relay for outreach

Most relays, like Amazon SES, forbid cold email in their terms and can close your account. Outlet warns you if you tick **Outreach** on a relay. Outreach belongs on your own mailboxes. See [Outreach overview](/help/outreach-overview).

## Common problems

- **"Couldn't connect and log in to ..." when saving**: the relay turned the sign-in down or couldn't be reached. Check the username and password at the relay (for SendGrid the username is literally `apikey`), then the host and port. Some hosting companies block port 25; use 587. Nothing is saved until it works.
- **"Enter the SMTP username and password"**: fill in both. Outlet sends through a relay only with a login.
- **"ENCRYPTION_KEY is not set on this server, so it can't store credentials"**: whoever installed Outlet needs to set `ENCRYPTION_KEY` on the server.
- **Mail lands in spam**: the relay's SPF and DKIM records are probably missing. Add them at your DNS host.
