# How sending works

Every email Outlet sends goes out through a **sending pathway**: one way a brand delivers mail, such as Amazon SES, an SMTP relay or your own mailboxes. This guide explains the three kinds of mail, why each kind goes through exactly one pathway, and the limits that protect your reputation.

You manage pathways in the brand's **Settings**, **Sending**.

## The three kinds of mail

| Kind | What it is | Examples |
| --- | --- | --- |
| **Transactional** | Mail one person needs because of something they did | Receipts, sign-in links, password resets, anything your app sends through the API or SMTP |
| **Marketing** | Mail to people who signed up for it | Campaigns, newsletters, autoresponders and sequences |
| **Outreach** | One-to-one email to people who haven't signed up | The **Outbound** section |

## Why each kind goes through one pathway

Each kind of mail is carried by **at most one** pathway. Ticking a kind on one pathway takes it off the others. This keeps the kinds apart, so a problem with one never spreads to the others:

- **Your receipts keep arriving** even if a newsletter gets many complaints, when the two go through different pathways.
- **Outreach never touches Amazon SES.** Amazon SES and most email platforms forbid cold email in their terms and can close your account. Outreach goes through your own mailboxes, on domains of its own.
- **Nothing falls back silently.** If no pathway carries a kind, Outlet doesn't send that kind at all, and says so: "brand X has no sending pathway for marketing mail; add one in Settings, Sending". This is on purpose: quietly sending through the wrong service could break that service's terms.

## The four ways to send

When you add a pathway, **Sends through** offers:

- **This server's email settings**: the Amazon SES or SMTP relay set for the whole Outlet in **Settings**, **Integrations**. This is the default.
- **Your own Amazon SES**: this brand's own AWS keys and region, for example when a client pays for their own AWS. See [Set up Amazon SES](/help/set-up-amazon-ses).
- **Your own SMTP relay**: this brand's own SendGrid, Mailgun, Postmark or similar. See [Use your own SMTP server](/help/use-your-own-smtp-server).
- **Your connected mailboxes**: a pool of real mailboxes, for outreach. See [Set up outreach end to end](/help/set-up-outreach).

A new brand starts with one pathway, "Platform email", which sends through this server's email settings and carries **Transactional** and **Marketing**.

## Add or change a pathway

1. Open **Settings**, **Sending** and press **Add pathway**.
2. **Name**: anything you like, such as "Receipts" or "Mailboxes".
3. **Sends through**: one of the four above. It can't be changed later; add a new pathway instead.
4. Fill in the keys or login the pathway asks for. Outlet stores them encrypted.
5. Under **Carries**, tick the kinds of mail it should send.
6. Press **Add pathway**.

A new pathway starts with its provider's default limits. Press **Edit** on its card afterwards to change them.

To remove a pathway, first move its kinds of mail and its domains to another pathway. **Remove** stays greyed out while the pathway carries any mail. Each domain on a pathway's card has a **Move to...** list and a **Move** button for this.

## Limits and warm-up

Each pathway has limits **per sender**. A sender is each mailbox of a mailbox pool, or the pathway itself for the others. Press **Edit** on a pathway to see them.

**Limits per sender**

- **Emails a day**: the most one sender sends in a day.
- **Warm-up first day** and **Warm-up added each day**: a new sender starts low and climbs each day until it reaches **Emails a day**.
- **Seconds between sends, at least** and **at most**: Outlet waits a random time in this range between two emails from the same sender, so the sending looks like a person's.

**Pause a sender over 7 days at**

- **Bounce rate (%)** with **With bounces, at least**: pause when too many emails bounce.
- **Spam blocks**: pause when receiving servers block the mail as spam.
- **Complaints**: pause when people mark the mail as spam.

A 0 turns a limit off.

**Defaults**

| | Daily | Warm-up | Between sends | Pauses at |
| --- | --- | --- | --- | --- |
| Mailbox pool | 30 a day | 5, then +2 a day | 4 to 9 minutes | 3% bounces (2 or more), 2 spam blocks, 2 complaints |
| Amazon SES, SMTP relay, platform | No limit | None | None | 3% bounces (2 or more), 2 spam blocks, 2 complaints |

Amazon SES and SMTP relays have their own sending limits on their side, so Outlet doesn't add daily limits for them by default.

**What a limit does:** an email held back by a limit is not lost. It waits and goes out when the limit allows.

More on mailbox limits and what to do when a sender is paused: [Warm-up and limits](/help/warm-up-and-limits).

## Common problems

- **"brand X has no sending pathway for (kind) mail; add one in Settings, Sending"**: tick that kind on a pathway.
- **"This brand can't send (kind) mail from (address). Add (domain) to its (pathway) sending pathway first."**: the From address's domain belongs to a different pathway. Move the domain, or send from another address.
- **"(name) still carries (kinds) mail; tick those kinds on another pathway first"**: you tried to remove a pathway that still sends mail.
- **"ENCRYPTION_KEY is not set on this server, so it can't store credentials"**: whoever installed Outlet needs to set `ENCRYPTION_KEY` on the server.
- **"No sending pathways. This brand sends nothing until one carries its mail."**: add a pathway.
