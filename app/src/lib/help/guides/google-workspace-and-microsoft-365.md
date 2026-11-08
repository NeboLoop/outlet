# Add Google Workspace or Microsoft 365 mailboxes by hand

You can use mailboxes you already pay for, at Google Workspace or Microsoft 365, for outreach. Outlet signs in to each one to send and to read replies. This guide covers app passwords, 2-step verification, Microsoft's sign-in rules and the terms to keep in mind.

**What needs a person, and why**

- **Creating the mailbox and its outreach domain** at Google or Microsoft, because they bill you and only you can sign in to their admin console.
- **Turning on 2-step verification and making an app password**, because Google only lets the account owner do it.
- **DNS records** for the outreach domain, unless its DNS is on DigitalOcean (then Outlet writes them). See [Set up outreach end to end](/help/set-up-outreach).

## Before you start

1. Use a separate **outreach domain**, never your main business domain. Outlet refuses the brand's own sending domain.
2. Add the outreach domain to Google Workspace or Microsoft 365 as a domain of its own (Google calls it a secondary domain).
3. Create one mailbox per real person, with that person's name.
4. In Outlet, open {{label.connectionsScreen}}, find the pool that carries **Outreach**, and under **{{label.connectionsHeading}}** press **{{label.addConnection}}**. Pick **{{label.connectionService}}** "{{label.manualService}}", give it a **{{label.connectionName}}** such as "Google Workspace", and press **{{label.addConnectionSave}}**. No API key is needed. Skip this if the pool already has a hand-added connection.
5. Add the domain in Outlet: **Outbound**, **Setup**, **Domains**, **Add domain**. Pick **Mail provider** "Google Workspace" for Google. For Microsoft 365, pick "Other" and use the MX host and SPF include from Microsoft's domain setup page. A mailbox can only be added on one of these domains. New domains join the pool's first hand-added connection; to put one on another connection, use **{{label.moveTo}}** and **{{label.move}}** under that connection's **{{label.manage}}**.

## Google Workspace

Google lets outside apps sign in with an **app password**: a 16-letter password made for one app, which works only when 2-step verification is on.

1. **Turn on 2-step verification.** Sign in as the mailbox user, open the Google Account, **Security**, and turn on **2-Step Verification**. If it isn't offered, the Workspace admin must allow it in the Admin console (**Security**, **Authentication**, **2-step verification**).
2. **Make an app password.** In the same Google Account, search for **App passwords**, give it a name such as "Outlet", and copy the 16 letters. Google shows them once.
3. **Allow IMAP.** Outlet reads replies over IMAP. In the Admin console, make sure IMAP access is allowed for the user.
4. **Add the mailbox in Outlet.** Open **Outbound**, **Setup**, under **Mailboxes** press **Add mailbox**, and fill in:
   - **Email**: the mailbox address.
   - **From name**: the real person's name.
   - **Provider**: "Google Workspace".
   - **App password**: the 16 letters, without spaces.
5. Press **Add mailbox**. Outlet fills in Google's servers (smtp.gmail.com and imap.gmail.com) and tests both logins before saving.

**DKIM for Google**: in the Admin console, open **Apps**, **Google Workspace**, **Gmail**, **Authenticate email**, pick the outreach domain and generate a key. Copy the value into **DKIM value** when you add the domain in Outlet (the selector is "google"), then press **Start authentication** in Google once the record is in DNS.

## Microsoft 365

Pick **Provider** "Custom (SMTP and IMAP)" and fill in:

| Field | Value |
| --- | --- |
| **SMTP host** | smtp.office365.com |
| **SMTP port** | 587 |
| **SMTP username** | the full mailbox address |
| **IMAP host** | outlook.office365.com |
| **IMAP port** | 993 |
| **IMAP username** | the full mailbox address |
| **Password** | the mailbox password, or an app password if your organization uses them |

**Microsoft's SMTP sign-in status.** Outlet signs in to the mailbox with a password. Microsoft has been switching off password sign-in for sending (it calls this "Basic authentication for SMTP AUTH") across Microsoft 365, and Authenticated SMTP is off for each mailbox by default.

- In the Microsoft 365 admin center, open the user, **Mail**, **Manage email apps**, and make sure **Authenticated SMTP** and **IMAP** are ticked.
- If your organization no longer allows password sign-in for SMTP, Outlet can't connect, and you see "SMTP login failed". Check Microsoft's current status for your tenant before buying Microsoft 365 mailboxes for outreach. Google Workspace or a mailbox service is the dependable choice today.

## The terms caveat

Google Workspace and Microsoft 365 are built for business mail, and both forbid spam and bulk unsolicited mail in their acceptable use policies. Personal, low-volume outreach to business addresses, with a working opt-out, is how most people stay within them, and it is how Outlet sends: about 30 a day per mailbox, spaced minutes apart, stopped by any reply. Going beyond that risks your mailbox, and even your whole Workspace or Microsoft 365 account, being suspended. Keep outreach on a separate domain so your main business mail is never at risk.

## Common problems

- **"SMTP login failed for ... For Google, turn on 2-step verification and use an app password."**: Google: use an app password, not the normal password, and check 2-step verification is on. Microsoft: check Authenticated SMTP is allowed (above).
- **"... is not on one of this brand's outreach domains. Add its domain in Outbound, Setup first."**: add the domain before the mailbox.
- **The mailbox is added but paused for DNS**: the outreach domain is missing records. Press **Check** next to the domain on **Setup** to see which.
- **Replies don't show up**: IMAP is blocked for the user. Allow it in the admin console. Outlet reads only mail that arrives after the mailbox was connected.
