var e=`# Backups and updates

Everything in Outlet lives in one database file on your server. A backup is a copy of it. This guide covers making backups, keeping them safe on Amazon S3, scheduling them, restoring one, and updating Outlet.

Only a super admin can manage backups and updates. Both are in Outlet's own **Settings**: on the **Brands** screen, click your initial at the top right and choose **Settings**.

## Make a backup now

1. Open the **Backup** tab.
2. Under **Create Backup**, tick **Also upload to S3** if S3 is set up (below).
3. Press **Create Backup Now**.

The backup appears under **Backup History** with its size and where it's stored: **Local** (on the server, in a \`backups\` folder next to Outlet) or **S3**. Press **Download** to keep a copy on your computer.

Each backup is a zip holding the database twice: as a database file (\`.db\`) and as a readable SQL file (\`.sql\`).

## Keep backups off the server

A backup on the same server is lost if the server is. Store them on Amazon S3 (Amazon's file storage) too.

**This needs you**, because only you can sign in to AWS and create the storage.

1. In AWS, create an S3 bucket, such as "acme-outlet-backups".
2. Give an AWS user the permissions shown under **Required IAM Policy** on this tab, with your bucket's name in place of \`YOUR-BUCKET-NAME\`. You can add them to the user you made for Amazon SES.
3. In Outlet, under **S3 Storage Settings**, switch on **Enable S3 backup storage** and fill in **S3 Bucket**, **S3 Region**, **S3 Key Prefix (optional)**, **Access Key ID** and **Secret Access Key**.
4. Press **Save Settings**.

## Schedule backups

1. Under **Scheduled Backups**, switch on **Enable scheduled backups**.
2. **Schedule (Cron)**: when to run, in cron format (a standard way to write times). The default \`0 3 * * *\` means 3 AM every day.
3. **Retention (days)**: backups older than this are deleted.
4. Press **Save Settings**.

## Restore a backup

There's no restore button in the app: restoring replaces everything, so it's done on the server.

**This needs the person who runs your Outlet server**, because it means stopping Outlet and running a command.

1. Stop Outlet.
2. Run \`outlet restore\` with the backup's database file, such as \`outlet restore outlet-backup.db\`. Outlet saves a copy of the current database first, then asks to confirm.
3. Start Outlet again.

Unzip the backup first; the \`.db\` file inside is what to restore.

## Update Outlet

1. Open the **Updates** tab. **Current Version** shows what you run.
2. Press **Check for Updates**. You see "You're up to date!" or **Update Available** with **What's New**.

Installing an update is done on the server, by the person who runs it:

- **Command line (recommended)**: run \`outlet update\`.
- **Docker**: run \`docker pull ghcr.io/outlet-sh/outlet:latest\` and restart the container.
- **By hand**: download the new version from GitHub Releases, stop Outlet, replace the program file, and start it again.

Make a backup before every update.

## Common problems

- **Backup shows Failed**: the server may be out of disk space. Delete old local backups, or set a shorter **Retention (days)**.
- **S3 upload fails**: check the bucket name and region, and that the AWS user has the four S3 permissions on that bucket.
`,t=`# Campaigns and newsletters

A campaign is one email sent to one or more of your lists, such as a newsletter, an announcement or a sale. This guide covers writing it, choosing who gets it, testing, sending or scheduling, and reading the results.

Campaigns are **marketing** mail. They go only to people who joined your lists, never to outreach prospects, and they go out through the pathway that carries **Marketing** in **Settings**, **Sending**. See [How sending works](/help/how-sending-works).

## Create a campaign

1. Open **Campaigns** and press **New Campaign**.
2. Type a name at the top, where it says "Campaign Name". Only you see it.
3. **Subject Line**: what people see first in their inbox.
4. **Preview Text**: the short line shown after the subject in most inboxes.
5. Write the email in the editor. Merge tags put each person's details in, such as \`{{name}}\`, or \`{{name,fallback=Friend}}\` for people without a name. \`{{unsubscribe_url}}\` and \`{{web_version_url}}\` are also available.
6. **Recipients**: tick one or more lists. Outlet shows "Total: N recipients".
7. **Exclude Lists** (optional): tick lists whose members should not get this email, such as existing customers for a sign-up offer.
8. **Sender (optional)**: **From Name**, **From Email** and **Reply-To Email**. Leave them blank to use the brand's defaults.
9. **Template**: "None" sends your content as it is. Pick a design to wrap your content in it. See [Designs](/help/designs).
10. **Tracking**: **Track opens** and **Track clicks** are on by default.
11. Press **Save as Draft**.

The name, subject, body and at least one list are required. Without them you see "Please fill in all required fields".

## Send yourself a test

1. Under **Test Email**, type your address.
2. Press **Send Test**. The test arrives with "[Test]" before the subject.
3. Check it on your phone and computer: images, links, the unsubscribe link, and how merge tags came out.

## Send or schedule

Open the campaign from the **Campaigns** list.

- **Send now**: press **Send now**. Outlet asks "Send this campaign now?" and says how many people it goes to. It starts within a minute and can't be undone.
- **Schedule**: under **Schedule**, pick a date and time and press **Schedule**. The time is in your computer's time zone, shown as "Your local time". Pick a time in the future.

The **Audience** card says how many subscribers it goes to. Unsubscribed, bounced and suppressed addresses are skipped, and someone on two of the chosen lists gets only one copy.

To change a draft or scheduled campaign, press **Edit**. To throw away a draft, press **Delete draft**.

## Read the results

After sending, the campaign's page shows **Recipients**, **Sent**, **Delivered**, **Opened**, **Clicked**, **Bounced**, **Complaints** and **Unsubscribed**, plus **Links clicked** and a preview of the email.

Opens are rough: Apple Mail opens every email automatically, so open rates run high. Clicks are the reliable signal.

The **Campaigns** list has tabs for **All**, **Drafts**, **Scheduled**, **Sending** and **Sent**.

## Common problems

- **The campaign shows "paused"**: Outlet pauses a campaign after 100 errors in a row, usually a sending setup problem. Check **Settings**, **Sending**, and see [Troubleshooting](/help/troubleshooting).
- **"Only a draft or scheduled campaign can be sent."**: the campaign already went out or is sending.
- **"brand X has no sending pathway for marketing mail"**: tick **Marketing** on a pathway in **Settings**, **Sending**.
- **Fewer recipients than subscribers**: unsubscribed, bounced, blocked and suppressed people are skipped, and pending double opt-in subscribers haven't confirmed yet.
- **To reuse a campaign**, create a new one and paste in the content. The **Duplicate** menu item currently opens an empty campaign.
`,n=`# Clean lists, validation and the blocklist

Sending to bad or uninterested addresses hurts your reputation with mail providers, and then even your good mail lands in spam. Three screens keep your lists clean: **Validation** checks addresses, **Blocklist** stops mail to people and domains, and **Housekeeping** removes people who no longer read your mail.

## Validation

Validation checks whether an address can receive mail, without sending anything.

**Check one address**

1. Open **Validation**.
2. Type the address under **Check an email** and press **Validate**.
3. Read the result, and any "Did you mean ...?" suggestion for typos such as "gmial.com".

**Check a whole list or file**

1. Under **Bulk jobs**, press **New Job**.
2. Pick **From a list**, or **Upload CSV** and choose the **Email column**.
3. **Job name (optional)**, then press **Start Validation**.
4. Open the job when it's done. **Export CSV** downloads the results.
5. Press **Suppress** to stop mail to the bad addresses. **Invalid**, **Spamtrap**, **Abuse** and **Do not mail** are ticked by default. Suppressed addresses get no email at all.

**What the results mean**

- **Valid**: the address exists.
- **Invalid**: it doesn't exist. Don't send.
- **Catch-all**: the domain accepts every address, so the check can't tell. Send with care.
- **Unknown**: the server didn't answer. Try again later.
- **Spamtrap**: an address kept to catch spammers. Never send.
- **Abuse**: belongs to someone known to report mail as spam. Don't send.
- **Do not mail**: shared role addresses such as info@, throwaway addresses, forwarding services and likely traps. Don't send.

## Blocklist

**Suppressed Emails**: addresses that get no mail from this brand, whatever list they're on. Outlet adds outreach opt-outs here itself. To add one by hand, use **Add Email to Suppression List**. Separately, Outlet never sends to an address that hard-bounced or complained, even if it isn't listed here.

**Blocked Domains**: whole domains, such as a competitor's. Outlet refuses new subscribers from a blocked domain and won't send to it. Use **Block a Domain**.

## Housekeeping

Housekeeping deletes contacts for good. Always press **Preview** first and read the number.

**Inactive Contacts** removes confirmed, still-subscribed people who haven't opened anything in **No opens in (days)** (90 by default) and haven't clicked anything in **No clicks in (days)** (180 by default). Every kind of email counts: campaigns, sequences, outreach and transactional. People newer than the windows are kept. Removing them improves your open rates and deliverability.

**Unconfirmed Subscriptions** removes contacts Outlet hasn't marked as verified, older than **Older than (days)** (30 by default). Be careful: today Outlet marks very few contacts as verified. People you imported or added by hand, people confirmed through double opt-in, and outreach prospects can all count as unverified. If the **Preview** number is bigger than the people you expect never confirmed, don't delete.

After **Preview**, press **Delete N Contact(s)** or **Delete N Subscription(s)**. Deleted contacts can't be recovered, so take a backup first (see [Backups and updates](/help/backups-and-updates)).

## Common problems

- **High bounces after an import**: validate the list, then suppress the bad addresses.
- **Someone says they never get your mail**: search for them in **Blocklist**, **Suppressed Emails**. Also ask whether their address ever bounced or they reported your mail as spam: Outlet then stops sending to them.
`,r=`# Connect a mailbox service

A mailbox service sells ready-made mailboxes for outreach and runs their domains' mail records for you. Connect one to Outlet with its API key, and Outlet can buy domains, connect domains you own, create mailboxes and bring them in, without you typing a single DNS record or password.

This is the most automatic way to set up outreach. Outlet supports **Maildoso**, **CheapInboxes** and **Infraforge**. For mailboxes you already have, such as Google Workspace or Microsoft 365, see [Add Google Workspace or Microsoft 365 mailboxes by hand](/help/google-workspace-and-microsoft-365).

**What needs a person, and why**

- **Signing up at the service and adding a card** needs you, because the service bills you directly. Outlet never holds your card.
- **Copying the API key** needs you, because only you can sign in to the service.
- **Setting nameservers at your registrar** needs you when you connect a domain you already own, because only you can sign in to the company you bought the domain from.
- **Saying yes to every purchase** needs you. Outlet shows a price quote first and spends nothing until you confirm.

## How it fits together

- A brand can hold several mailbox service connections. Each is one service with its own API key. See [Run several mailbox services together](/help/several-mailbox-services).
- Each outreach domain belongs to one connection.
- Your outreach pool sends from every mailbox of every connection, under the pool's daily limit and warm-up. See [Warm-up and limits](/help/warm-up-and-limits).

## Add a connection

1. Sign up at the service and add a payment method there.
2. In the service's dashboard, find its API settings and create an API key. Copy it.
3. In Outlet, open {{label.connectionsScreen}} and find **{{label.connectionsHeading}}**.
4. Press **{{label.addConnection}}**.
5. **{{label.connectionService}}**: pick Maildoso, CheapInboxes or Infraforge.
6. **{{label.connectionApiKey}}**: paste the key.
7. Press **{{label.saveConnection}}**. Outlet tests the key with the service before it saves, and stores it encrypted.

If the key is wrong you see the service's own refusal. Copy the key again, with nothing before or after it.

Outlet needs its \`ENCRYPTION_KEY\` set on the server to store keys. Without it you see "ENCRYPTION_KEY is not set on this server, so it can't store credentials". Ask whoever installed Outlet to set it.

## Connect a domain you own

Use this when you already bought an outreach domain somewhere else, such as Namecheap or GoDaddy.

1. Under **{{label.connectDomainSection}}**, type the domain and press **{{label.connectDomain}}**.
2. Outlet shows the service's note and **Nameservers**.
3. Sign in at your domain registrar and replace the domain's nameservers with the ones shown. (Nameservers decide which company answers for the domain's DNS. See [Your sending domain and DNS explained](/help/sending-domain-and-dns).)
4. Wait. The service takes over the domain's DNS and writes its MX, SPF, DKIM and DMARC records itself.

The domain appears on **Outbound**, **Setup**, under **Domains**, by itself. Outlet refuses your brand's own sending domain and any domain another brand uses.

## Buy a domain or mailboxes

Every purchase is two steps: a price quote, then your yes.

1. Under **{{label.buySection}}**, pick what to buy: "{{label.buyKindMailboxes}}", "{{label.buyKindDomain}}" or "{{label.buyKindConnect}}".
2. Type the **Domain**.
3. For mailboxes, list the people in the box marked "{{label.buyPeople}}", one a line: \`First Last\`, or \`First Last, local\` to choose the part before the @. Without it, the address is first.last@yourdomain. Up to 100 mailboxes an order.
4. Press **{{label.getQuote}}**. Outlet shows the price, or "Price unknown" when the service doesn't say, and when the quote expires. A quote lasts 30 minutes and is used once.
5. Read it, then press **{{label.buy}}**. Outlet asks once more, with the amount, before anything is spent.

After the buy, Outlet tests each new mailbox's login and adds it to your pool. Mailboxes the service hasn't finished yet are listed as waiting. Bring them in later with sync, below.

## Sync

Use **{{label.syncSection}}** (with the service's name) when you made mailboxes in the service's own dashboard, or a purchase was still finishing.

- **{{label.syncInventory}}** lists the domains and mailboxes you have at the service.
- **{{label.syncCompare}}** shows what the service has that Outlet doesn't, and the reverse.
- **{{label.syncImport}}** brings in every finished mailbox Outlet is missing, after testing its login.

## Delete a mailbox

Under **{{label.deleteSection}}** (with the service's name), press **{{label.deleteMailbox}}** next to the mailbox and confirm. The service deletes it (CheapInboxes cancels it at the end of its billing period instead). Maildoso and Infraforge delete its messages too. In Outlet the mailbox stays, paused for good, so its threads and history are kept.

## Maildoso

- **API key**: a personal access token from Maildoso's dashboard.
- **Automatic**: connects domains you own for free, buys new domains, creates mailboxes, and runs DNS for its domains.
- **Connecting a domain**: Maildoso replaces the domain's MX, SPF, DKIM and DMARC records. Set the nameservers it gives within 24 hours. It marks the domain active, usually within 2 hours.
- **Costs**: Maildoso's API gives no prices. Domains and mailboxes are billed to the payment method on your Maildoso account and plan, so the quote shows "Price unknown". Check prices in Maildoso before you buy.
- **Its limits**: Maildoso's terms cap cold email at **15 a day per mailbox**, after **14 days of warm-up**, to **business addresses only**. Choosing Maildoso sets the pool to 15 a day, warming up from 5 and adding 1 a day. You can change the limits later in **Settings**, **Sending**, but going above Maildoso's terms risks your account.

## CheapInboxes

- **API key**: from CheapInboxes' dashboard. It starts with \`ci_live_\`.
- **Automatic**: sells Google Workspace mailboxes, buys new domains with their mailboxes, and runs DNS for its domains.
- **Connecting a domain**: free. The domain is staged first, and the move finishes in the order that buys its first mailboxes. CheapInboxes then gives the nameservers to set at your registrar.
- **Costs**: quotes show real prices in US dollars: what is charged today to the card on file, then a monthly price for the mailboxes.
- **Deleting**: cancels the mailbox at the end of its billing period. It keeps working until then.
- **Its limits**: follow CheapInboxes' own terms for sending volume. Outlet uses its normal outreach limits: 30 a day, warming up from 5, adding 2 a day.

## Infraforge

- **API key**: from Infraforge's dashboard.
- **Automatic**: creates mailboxes on your domains, and runs DNS for domains connected to it.
- **Connecting a domain**: Infraforge bills for it, so it is a purchase. Use **{{label.buySection}}** with "{{label.buyKindConnect}}", get a quote and buy. Then set the nameservers it gives at your registrar.
- **Buying a domain**: not possible from Outlet. Registering a domain at Infraforge needs the owner's contact details, so buy it in Infraforge's own app, then sync.
- **Costs**: Infraforge invoices mailboxes and connections, and its API gives no price before the invoice. Check prices in Infraforge first.
- **Its limits**: follow Infraforge's own terms. Outlet uses its normal outreach limits: 30 a day, warming up from 5, adding 2 a day.

## Common problems

- **"The mailbox service refused"** or the service's own message: the service said no. The message says why, such as a low balance or a domain that isn't free.
- **A domain "belongs to another mailbox pool"** or "is already in use": each domain can belong to only one connection and one brand.
- **Mailboxes stay waiting after a buy**: the service hasn't finished them. Wait, then use **{{label.syncImport}}**.
- **Mailboxes paused for DNS**: the nameservers aren't set yet, or the service hasn't finished the records. Set the nameservers and wait. Outlet checks daily and resumes the mailboxes by itself. Press **Check** on **Outbound**, **Setup** to check sooner.
`,i=`# Designs

A design is a reusable email layout, such as your logo at the top and your address and social links at the bottom. Your campaign's content is placed inside it, so every newsletter looks the same without copying the layout each time.

In the menu it's called **Designs**; the page itself is titled **Templates**.

## Create a design

1. Open **Designs** and press **New Template**.
2. In **Create New Template**, fill in **Name**, **Slug** (a short name, which can't be changed later), **Category** ("General", "Newsletter", "Promotional" or "Transactional") and **Description (optional)**.
3. Press **Create Template**.
4. In the editor, build the layout. Put \`{{content}}\` where each email's own content goes. It is listed as "Main content area" among the merge tags.
5. Keep **Active** on. Press **Preview** to see it, then **Save & Close**.

If a design has no \`{{content}}\`, Outlet places the content at the end of the email instead.

## Use a design

In the campaign editor, under **Template**, pick the design. "None" sends your content as it is. Only **Active** designs are offered. See [Campaigns and newsletters](/help/campaigns-and-newsletters).

Transactional emails have their own templates on the **Transactional** page. See [Transactional email from your app](/help/transactional-email).

## Tips

- Keep designs simple: one column, a logo, your content, and a footer with your address and the unsubscribe link (\`{{unsubscribe_url}}\`).
- Send a test campaign to yourself on a phone and a computer before using a new design.
- To make a variation, create a new design and paste in the old one's code. The **Duplicate** menu item doesn't work yet.

## Common problems

- **A design doesn't appear in the campaign editor**: switch it to **Active**.
- **Content shows at the bottom instead of the middle**: add \`{{content}}\` to the design where the content belongs.
`,a=`# Getting started

This guide takes you from a fresh Outlet install to your first email. It covers the setup wizard, your first brand, and the settings that decide how your mail goes out.

**Words used in this guide**

- **Brand**: one business or product you send email for. Each brand has its own lists, designs, settings and API key. One Outlet can run many brands.
- **Amazon SES**: Amazon's email sending service. It is low cost and handles large volumes. Outlet can set up most of it for you through Amazon's own API.
- **SMTP relay**: another company's email sending server, such as SendGrid, Mailgun or Postmark.

## 1. Run the setup wizard

The first time you open Outlet, it shows the setup wizard. It has three steps: **Account**, **Email** and **Complete**.

**Create Admin Account**

1. Fill in **Company**, **Name**, **Email Address**, **Password** (at least 8 characters) and **Confirm Password**.
2. Check **Timezone**. Outlet fills it in from your computer. It is used to schedule campaigns and show report times.
3. Press **Continue**. You are signed in automatically.

This first account is a super admin: it can do everything, including adding other users.

**Connect Amazon SES**

1. Pick the **AWS Region** closest to your audience. The default is "US East (N. Virginia)".
2. Paste the **Access Key ID** and **Secret Access Key** of an AWS user that has Outlet's permissions. [Set up Amazon SES](/help/set-up-amazon-ses) shows how to make one, step by step.
3. Press **Complete Setup**.

Prefer another sending company, or don't have AWS yet? Press **Skip for now**. You can connect Amazon SES or an SMTP relay later in **Settings**, **Integrations**. See [Use your own SMTP server](/help/use-your-own-smtp-server).

**Complete**

You see "You're all set!". Press **Go to Dashboard**.

## 2. Create your first brand

1. On the **Brands** screen, press **New Brand** (or **Create brand** when you have none yet).
2. **Name**: your business or product name.
3. Under **Email Settings**:
   - **From Name**: the name people see in their inbox, such as "Acme" or "Sam at Acme".
   - **From Email**: the address mail comes from, such as \`hello@acme.com\`. Use a domain you own, not a gmail.com address.
   - **Reply To (optional)**: where replies go. Left empty, replies go to the From Email.
4. Press **Create**, then click the new brand to open it.

You can also add a brand from the **Brands** menu at the top, with **Add brand**. That asks for a **Brand Name** and **URL Slug** (the short name in the brand's web address, such as \`/acme\`, which can't be changed later), then opens a welcome screen asking for the same **From Name**, **From Email** and **Reply-To Email (optional)**. Press **Save & Continue** there.

## 3. What happens automatically

When a brand gets a From Email, Outlet asks Amazon SES to verify its domain, in the background. It then lists the DNS records the domain needs, in the brand's **Settings**, **Email**, under **Domain Verification**. See [Set up Amazon SES](/help/set-up-amazon-ses).

You can change the sender later in the brand's **Settings**, **Email**, under **Sender Identity**.

## 4. Add the DNS records

**This step needs you**, because only you can sign in to the company that runs your domain's DNS (often where you bought the domain).

1. Open the brand's **Settings**, then **Email**.
2. Under **Domain Verification**, copy each record into your DNS host.
3. Press **Refresh** to check. Outlet also checks every minute by itself, and the page updates when the domain is **Verified**.

[Your sending domain and DNS explained](/help/sending-domain-and-dns) explains each record in plain words.

## 5. Check how your mail is sent

Open the brand's **Settings**, **Sending**. A new brand has one pathway, "Platform email", that sends **Transactional** and **Marketing** mail through the server's email settings. For most people that is right and there is nothing to change. [How sending works](/help/how-sending-works) explains pathways and when to add another.

## 6. Send your first email

- **A newsletter**: create a list in **Lists**, add subscribers, then create a campaign in **Campaigns**. See [Lists, subscribers and importing](/help/lists-and-subscribers) and [Campaigns and newsletters](/help/campaigns-and-newsletters).
- **Email from your app**, such as receipts or password resets: see [Transactional email from your app](/help/transactional-email).
- **One-to-one outreach** to people who haven't signed up: see [Outreach overview](/help/outreach-overview).

Before your first real send, use **Send Test** in the campaign editor to send yourself a copy.

## Common problems

- **New AWS accounts start in "sandbox mode".** In sandbox mode, Amazon SES only sends to addresses you have verified in AWS. Request production access in your AWS console before sending to customers. See [Set up Amazon SES](/help/set-up-amazon-ses).
- **The welcome screen keeps coming back.** A brand without a From Email sends you to it. Fill it in, or press **Skip for now**.
- **"Failed to save AWS credentials"**: check that both keys were pasted in full, with no spaces.
`,o=`# Add Google Workspace or Microsoft 365 mailboxes by hand

You can use mailboxes you already pay for, at Google Workspace or Microsoft 365, for outreach. Outlet signs in to each one to send and to read replies. This guide covers app passwords, 2-step verification, Microsoft's sign-in rules and the terms to keep in mind.

**What needs a person, and why**

- **Creating the mailbox and its outreach domain** at Google or Microsoft, because they bill you and only you can sign in to their admin console.
- **Turning on 2-step verification and making an app password**, because Google only lets the account owner do it.
- **DNS records** for the outreach domain, unless its DNS is on DigitalOcean (then Outlet writes them). See [Set up outreach end to end](/help/set-up-outreach).

## Before you start

1. Use a separate **outreach domain**, never your main business domain. Outlet refuses the brand's own sending domain.
2. Add the outreach domain to Google Workspace or Microsoft 365 as a domain of its own (Google calls it a secondary domain).
3. Create one mailbox per real person, with that person's name.
4. Add the domain in Outlet first: **Outbound**, **Setup**, **Domains**, **Add domain**. Pick **Mail provider** "Google Workspace" for Google. For Microsoft 365, pick "Other" and use the MX host and SPF include from Microsoft's domain setup page. A mailbox can only be added on one of these domains.

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
`,s=`# How sending works

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
- **"ENCRYPTION_KEY is not set on this server, so it can't store credentials"**: whoever installed Outlet needs to set \`ENCRYPTION_KEY\` on the server.
- **"No sending pathways. This brand sends nothing until one carries its mail."**: add a pathway.
`,c=`# Lists, subscribers and importing

A list is a group of people who asked to hear from you, such as "Newsletter" or "Product updates". This guide covers creating lists, adding and importing subscribers, double opt-in, custom fields and the subscribe form.

## Create a list

1. Open **Lists** and press **New List**.
2. **Name**: such as "Newsletter". The **Slug** fills itself in; your app and forms use it.
3. **Description (optional)**.
4. **Require double opt-in** is on by default. Keep it on: new subscribers get an email asking them to confirm, which keeps fake and mistyped addresses off your list.
5. Press **Create List**.

## Add one subscriber

1. Open the list, then the **Subscribers** tab.
2. Press **Add subscriber**, type the **Email** and **Name (optional)**, and add them.

They join right away, without a confirmation email. Add only people who asked to hear from you.

## Import a CSV file

A CSV file is a spreadsheet saved as plain text. Any spreadsheet app can save one ("Save as" or "Download as" CSV).

1. Make sure the first row has the column names, with one column called \`email\`. A \`name\` column is used too. Other columns are ignored.
2. Open the list's **Subscribers** tab and press **Import CSV**.
3. Pick the file (up to 50 MB).
4. Watch the progress bar: "Importing (file): x / y". When it's done you see "Imported X of Y rows (Z refused or invalid)".

**What the import does**

- Imported people join as confirmed straight away, without a confirmation email. Import only people who agreed to hear from you.
- Someone already on the list, including someone who unsubscribed, is left as they are. Unsubscribed people are never re-added.
- Addresses at blocked domains are refused.
- Imports don't start the list's **On signup** autoresponder. See [Sequences and autoresponders](/help/sequences-and-autoresponders).

Thinking of importing an old or bought list? Check it with **Validation** first. Bad addresses bounce, and bounces hurt your sending reputation. See [Clean lists, validation and the blocklist](/help/clean-lists).

## Subscriber statuses

- **Confirmed**: gets your mail.
- **Pending**: signed up but hasn't clicked the confirmation link yet (double opt-in).
- **Unsubscribed**: left the list and gets nothing from it.

Click a subscriber for their details: what they were sent, opened and clicked, their custom fields and sequences. **Remove from List** takes them off this list.

## Custom fields

Custom fields hold extra details, such as company or birthday, and can be used in emails.

1. Open the list's **Custom Fields** tab and press **Add Field**.
2. **Field Name**: the label shown on the subscribe form.
3. **Merge Tag Key**: the short name used in emails as \`{{key}}\`.
4. **Field Type**: **Text**, **Number**, **Date** or **Dropdown** (with **Options**, separated by commas).
5. Tick **Required field** if people must fill it in to subscribe.

## Get subscribers: the subscribe form

Open the list's **Subscribe Form** tab.

- **Hosted Subscribe Form**: a ready page at your Outlet address. Press **Copy URL** and link to it from your website or social profiles.
- **Embeddable HTML Form**: press **Copy** and paste the code into your website.
- **API Subscription**: your app can subscribe people with \`POST /sdk/v1/lists/(list slug)/subscribe\`, sending the brand's API key in the \`X-API-Key\` header and \`email\`, \`name\` and \`custom_fields\` in the body. (The example on that tab shows \`/api/sdk/v1\` and an \`Authorization: Bearer\` header; use the path and header given here.)

## List settings

The list's **Settings** tab holds:

- **Require double opt-in**, and the **Confirmation Email** it sends (it must include \`{{confirm_url}}\`).
- **Subscribe Settings**: where people land after subscribing, after confirming, or when already subscribed. Empty uses a simple page from Outlet.
- **Thank You Email**: an optional email after subscribing.
- **Unsubscribe Settings**: single or double opt-out, and whether unsubscribing leaves **This list only** or **All lists**.
- **Goodbye Email**: an optional email after unsubscribing.
- **Delete List**, in the **Danger Zone**: deletes the list, its subscribers and its autoresponders, for good.

## Common problems

- **"The CSV needs an email column (and may have a name column)."**: name the first row's email column \`email\`.
- **"The file isn't a CSV with a header row."**: save the spreadsheet as CSV, with column names in the first row.
- **"This address's domain is on the brand's blocklist."**: see **Blocklist**.
- **Subscribers stay Pending**: they haven't confirmed. Check the confirmation email arrives (try it yourself), and that **Marketing** is carried by a pathway in **Settings**, **Sending**.
- **Exporting a list**: there is no export button on lists yet. A **Validation** job of the list has **Export CSV**.
`,l=`# Connect an AI assistant over MCP, and API keys

Outlet can be run by an AI assistant such as Claude or ChatGPT, through MCP (Model Context Protocol, a standard way for AI assistants to use other software). The assistant can create lists, write campaigns, find prospects, draft outreach and explain problems, all by asking. This guide also explains Outlet's two kinds of keys.

## Two kinds of keys

| Key | What it's for | Where |
| --- | --- | --- |
| **Brand API key** | Your apps sending email and subscribing people | The brand's **Settings**, **General**, **API Key** |
| **MCP key** | AI assistants managing Outlet | Outlet's **Settings**, **API**, **MCP keys** |

They don't mix: an MCP key can't send email, and a brand API key can't manage Outlet.

## Connect an AI assistant (the easy way)

1. Open Outlet's own **Settings**: on the **Brands** screen, click your initial at the top right and choose **Settings**. The **MCP** tab opens.
2. Copy **Your MCP Server URL**. It ends in \`/mcp\`.
3. Under **Setup Instructions**, pick your assistant: **Claude Desktop**, **ChatGPT**, **Claude Code** or **Cursor / IDE**, and follow its steps.
4. When the assistant connects, it opens a browser window. Sign in to Outlet there and allow access.

No key is needed this way: the assistant signs in as you (this is called OAuth), and can do what your account can do.

**Available Tools** on the same page lists everything the assistant can do, by group.

## Connect with an MCP key

Some tools can't open a sign-in window. Give them an MCP key instead.

1. In Outlet's **Settings**, open the **API** tab.
2. Under **MCP keys**, press **Create Key**, type a **Key Name** such as "Claude at work", and press **Create Key**.
3. Copy the key now. Outlet shows it once: "This is the only time you'll see this key."
4. In the tool, use it as a bearer token: the header \`Authorization: Bearer\` followed by the key.

To stop a key working, press the red button on its row and then **Revoke**. Revoking can't be undone.

## What an assistant can't do

On purpose, an assistant can't turn on outreach. Only the brand's admin can, in the app. An assistant also can't spend money at a mailbox service without your yes: it must show you the quote first.

## The brand API key

Your apps use the brand's API key to send transactional email and subscribe people. See [Transactional email from your app](/help/transactional-email).

1. Open the brand's **Settings**, **General**.
2. Under **API Key**, press the copy button.
3. **Regenerate Key** makes a new one. The old key stops working at once.

## Common problems

- **The assistant says it isn't allowed**: your account lacks the brand role for that action. See [Users and roles](/help/users-and-roles).
- **"Invalid API key" from your app**: you used an MCP key, or the brand key was regenerated.
- **The sign-in window doesn't open**: use an MCP key instead.
`,u=`# Outreach overview

Outreach (the **Outbound** section) sends short, one-to-one emails to people who have not signed up for your mail, from real mailboxes with a real person's name. This guide explains what it is for, what it is not, and the rules that keep your mail out of spam.

## What outreach is

- **Personal first emails** to people whose business address is published, such as a creator's channel page or a company's contact page.
- **A few follow-ups** in the same email thread, sent only if the person has not answered.
- **Sent from your own mailboxes** on separate outreach domains, a few a day from each mailbox, spaced minutes apart, inside working hours, on weekdays.
- **Watched for you.** Replies are read every 2 minutes and sorted. Anyone who replies stops getting follow-ups.

## What outreach is not

- **Not a newsletter.** Prospects are contacts who never signed up, so your campaigns and newsletters never reach them. Newsletters are in [Campaigns and newsletters](/help/campaigns-and-newsletters).
- **Not bulk mail.** A campaign holds at most 100 people, and each mailbox sends about 30 a day at most.
- **Not sent through Amazon SES or an email platform.** Amazon SES and most email platforms forbid cold email in their terms and can close your account. Outreach goes through your connected mailboxes only.
- **Not sent from your main domain.** Outlet refuses your brand's own sending domain for outreach, so a bad week of outreach never hurts your newsletters or receipts.

## The rules Outlet enforces

These are checked by the software. You can't turn them off, because breaking them is what gets mail blocked.

1. **Outreach is off until the brand's admin turns it on** in **Outbound**, **Setup**. An AI assistant connected over MCP can do almost everything else, but it can never turn outreach on.
2. **A mailing address is required.** Every outreach email ends with your opt-out line and your business mailing address. Laws like CAN-SPAM in the United States require a real postal address. You can't create a campaign without one.
3. **Every prospect needs a source**: where you found the address, such as "YouTube About page". Outlet checks each address, and holds risky ones so they are never sent.
4. **One mailbox per person.** Each prospect is given one mailbox, and every email in their sequence comes from it, in the same thread.
5. **Replies stop the sequence.** Any human reply stops that person's follow-ups at once. Only "Out of office" replies let the sequence carry on.
6. **Opt-outs are permanent.** "Unsubscribe", "remove me", "no thanks" and similar replies put the address on your suppression list. Outlet never writes to it again, and also won't let you answer it.
7. **The first email may have at most one link.** Cold email with several links lands in spam.
8. **Nothing sends without approval at first.** A new campaign asks you to approve every email. "Send without approval" is allowed only when every mailbox is at least 30 days old.

## Deliverability basics

Deliverability means your email reaches the inbox rather than spam. Four habits matter most.

- **About 30 a day per mailbox, no more.** Mailbox providers like Google and Microsoft watch how much a new mailbox sends. Outlet's default is 30 a day per mailbox, warming up from 5 on the first day (see [Warm-up and limits](/help/warm-up-and-limits)). To reach more people, add mailboxes, not volume.
- **No warm-up pools.** Some services offer to "warm up" a mailbox by swapping fake emails with thousands of other inboxes. Google and Microsoft spot these patterns. Outlet warms up the honest way: small real numbers that grow a little each day. Also use a new mailbox normally for about two weeks before outreach.
- **One person, one mailbox.** Don't send to the same person from several mailboxes. Outlet does this for you.
- **Answer replies, and let replies stop the sequence.** Reply rate is the number that counts. Opens are unreliable because Apple Mail opens every email automatically, which is why open and click tracking start off.

## Where things are

| Tab | What it is for |
| --- | --- |
| **Overview** | Status, next steps, today's sends, each mailbox's health, campaign results |
| **Campaigns** | Your outreach campaigns, and **New campaign** |
| **Drafts** | Emails waiting for your approval |
| **Replies** | Replies that need you, sorted by kind |
| **Prospects** | The people you plan to reach |
| **Setup** | On and off, mailing address, outreach domains, mailboxes |

The limits for your mailboxes live in **Settings**, **Sending**, on the mailbox pool (the pathway that carries outreach). See [How sending works](/help/how-sending-works).

## Next

Follow [Set up outreach end to end](/help/set-up-outreach).
`,d=`# Privacy and deleting data

People have the right to see the data you hold about them, and to have it deleted. Privacy laws such as the GDPR in Europe require it. This guide covers finding someone's data, their consent, exporting it, and deleting it.

## Find a person

1. Open the brand's **Settings**, **Privacy**.
2. Under **GDPR Compliance Tools**, type their email address and press **Search Contact**.

You see their **Contact Information**: email, contact ID and when they were added.

## Record consent

Under **Consent Preferences**, switch **GDPR Consent** and **Marketing Consent** to **Granted** or **Not Granted**, then press **Save Consent Changes**. Use this when someone tells you, by email or phone, what they agree to.

## Export someone's data

Under **Export Personal Data**, press **Export Data**. Outlet gathers their profile, consent, tags and activity into one file, which is kept for 24 hours.

## Delete someone's data

1. Under **Delete All Data**, press **Delete Data**.
2. Confirm. This can't be undone.

Only the brand's admin can delete.

**What is deleted**: the person, their list memberships, tags, custom field values, sequence progress, waiting emails, clicks and campaign records.

**What is kept**: records of transactional emails sent to them keep the email address, unlinked from the person, as a record that the mail was sent.

**Important**: deleting someone does not stop them being added again. If they sign up again, or are in a file you import later, they come back. If they asked never to be emailed again, also add their address in **Blocklist**, **Suppressed Emails** after deleting.

## Unsubscribes

People unsubscribe with the link in every marketing email. You choose in each list's **Settings** whether that leaves **This list only** or **All lists**. Unsubscribed people are never added back by an import.

## Common problems

- **The search finds nobody**: check the spelling, and that you're in the right brand.
- **The export download doesn't start**: the export is made, but the **Download Export** link doesn't work yet. Ask whoever runs your Outlet server for the file.
`,f=`# Your sending domain and DNS explained

Before mail services trust your email, your domain has to vouch for it. It does that through DNS records: small public notes attached to your domain. This guide explains each record in plain words, the difference between nameservers and DNS records, and what Outlet writes for you.

## The words

- **Domain**: your name on the internet, such as \`acme.com\`.
- **Registrar**: the company you bought the domain from, such as Namecheap, GoDaddy or Cloudflare.
- **DNS host**: the company that answers for your domain's records. Often it's the registrar, but it doesn't have to be.
- **DNS record**: one public note about your domain, such as "mail for acme.com goes to Google".

## Nameservers vs DNS records

These are often confused.

- **Nameservers** say **which company** holds your domain's DNS records. They are set at your **registrar**. A domain has one set of nameservers.
- **DNS records** are the notes themselves. They are added at your **DNS host**, the company the nameservers point to.

Changing nameservers moves all of your DNS to another company. It is how a mailbox service or DigitalOcean takes over a domain's records. Do this only on a domain used just for outreach, or after copying every existing record, or your website and other mail may stop working.

## The records for email

**SPF** (Sender Policy Framework): a TXT record listing the servers allowed to send mail for your domain. A domain must have only **one** SPF record. To allow two services, combine them in one record, such as \`v=spf1 include:_spf.google.com include:amazonses.com ~all\`.

**DKIM** (DomainKeys Identified Mail): a signature on every email, checked against a public key in your DNS. It proves the email really came from you and wasn't changed on the way. Amazon SES uses three CNAME records ending in \`._domainkey\`. Google uses one TXT record at \`google._domainkey\`.

**DMARC**: a TXT record at \`_dmarc.yourdomain\` that tells receivers what to do with mail that fails SPF and DKIM, and where to send reports. Outlet suggests starting with \`v=DMARC1; p=none\`, which only watches. Google, Yahoo and Microsoft expect every sender to have one.

**MX** (Mail Exchanger): says which server receives mail for a domain. Outreach mailboxes need MX records so replies arrive. Amazon SES uses an MX record on your MAIL FROM subdomain to receive bounces.

**Verification TXT**: a one-off record that proves to a service you own the domain, such as \`_amazonses.yourdomain\` for Amazon SES.

**Tracking CNAME**: for outreach with click tracking on, a \`go.\` subdomain pointing at your Outlet server, so tracked links use your own domain.

## What Outlet does for you

| Domain | What is automatic | What you do |
| --- | --- | --- |
| Brand sending domain on Amazon SES | Outlet registers it with Amazon, creates the DKIM keys and MAIL FROM, lists every record, and checks every minute | Add the records at your DNS host |
| Outreach domain on DigitalOcean DNS | Outlet writes MX, SPF and DMARC itself, and DKIM too when you paste its value from your mail provider; only the missing ones | Save a DigitalOcean token once |
| Outreach domain at a mailbox service | The service writes every record | Set the service's nameservers at your registrar, once |
| Outreach domain anywhere else | Outlet lists every record and checks them daily | Add the records at your DNS host |

**DigitalOcean automation, step by step**

1. Your domain's nameservers must point to DigitalOcean, and the domain must be added in DigitalOcean's **Networking**, **Domains**. If not, Outlet says "DigitalOcean has no zone for this domain. Point the domain's nameservers at DigitalOcean and add it there first."
2. In DigitalOcean, create a personal access token with read and write access to domains.
3. In Outlet, open **Outbound**, **Setup**, and under **DigitalOcean DNS token** paste it and press **Save token**.
4. Add the outreach domain with **DNS host** set to "DigitalOcean".

Outlet never adds a second SPF, DKIM or DMARC record, and never changes your other records. If DigitalOcean refuses some records, Outlet shows them to add by hand.

## How to add a record by hand

1. Sign in at your DNS host and open the DNS settings for the domain.
2. Add a record with the **Type**, **Name** (also called Host) and **Value** Outlet shows. For MX records, also the **Priority**.
3. Many DNS hosts add your domain to the name by themselves. If Outlet shows \`_dmarc.acme.com\`, type just \`_dmarc\`.
4. Save, then press **Refresh** (brand email) or **Check** (outreach domain) in Outlet.

Changes usually show up within minutes to an hour, and can take up to 72 hours.

## Common problems

- **Two SPF records**: receivers then ignore both. Merge them into one.
- **The domain appears twice in a name**, such as \`_dmarc.acme.com.acme.com\`: type only the part before your domain.
- **The record is there, but Outlet says it's missing**: wait an hour; DNS takes time to spread. Check you added it to the right domain.
- **Your website stopped working after changing nameservers**: the new DNS host doesn't have your website's records. Copy them over from the old host.

See also [Set up Amazon SES](/help/set-up-amazon-ses) and [Set up outreach end to end](/help/set-up-outreach).
`,p=`# Sequences and autoresponders

An autoresponder is a series of emails sent automatically, one after another, to each person who joins a list. A welcome series is the classic example: a welcome email right away, a tip the next day, an offer a few days later. Outlet calls these series sequences; the ones tied to a list are autoresponders.

Autoresponders are **marketing** mail and go out through the pathway that carries **Marketing**. Outreach follow-ups are a different thing, covered in [Set up outreach end to end](/help/set-up-outreach).

## Create an autoresponder

1. Open **Lists** and click the list.
2. Open the **Autoresponders** tab and press **New Autoresponder**.
3. In **Create Autoresponder**, give it a **Name** and a **Slug** (the short name your app uses to enroll people), pick the **Trigger**, and press **Create**.

**Triggers**

- **On signup**: starts when someone joins the list through your subscribe form or your app's API. With double opt-in, it starts when they confirm.
- **Manual enrollment**: starts only when you enroll someone yourself, or your app does.

Each list runs one **On signup** autoresponder. Put the whole welcome series in one, not several.

## Add the emails

1. On the autoresponder's page, under **Emails**, press **Add Email**.
2. Write the **Subject Line** and the email.
3. Set **Send Timing**: the first email counts **hours after signup** (0 means immediately); later emails count **hours after previous email** (24 by default).
4. Add as many emails as you need.

Each wait counts from when the previous email was actually sent.

## Settings

- **List**: move the autoresponder to another list.
- **On Completion**: when someone finishes this autoresponder, enroll them in another sequence. "None (do nothing)" is the default.
- **Save Changes** saves; **Delete** removes the autoresponder and all of its emails.

On the **Autoresponders** tab, each autoresponder's menu has **Pause** and **Activate**. A new autoresponder starts active straight away, so write its emails before people join the list.

## Enroll someone by hand

1. Open the list's **Subscribers** tab and click the person.
2. Press **Enroll in Sequence** and pick the sequence.

Someone already in a sequence is never enrolled in it twice.

**From your app**: \`POST /sdk/v1/sequences/enroll\` with your brand's API key in the \`X-API-Key\` header, and \`sequence_slug\` and \`email\` in the body. See [Transactional email from your app](/help/transactional-email) for API keys.

## Who gets an On signup autoresponder

- **Yes**: people who join through the hosted subscribe form, the embedded form, or your app's subscribe API, once they're active (after confirming, with double opt-in).
- **No**: people you add with **Add subscriber** or **Import CSV**. Enroll them by hand if they should get it.

## Common problems

- **Nobody gets the welcome email**: check the trigger is **On signup**, the autoresponder is **Active**, and **Marketing** is carried by a pathway in **Settings**, **Sending**. Check the person joined through a form or the API, not an import.
- **People wait for confirmation**: with double opt-in on, the autoresponder starts only once they click the confirmation link.
`,m=`# Set up Amazon SES

Amazon SES (Simple Email Service) is Amazon's email sending service. It is low cost, handles large volumes, and Outlet sets up most of it for you through Amazon's own API: domain verification, DKIM signing, the bounce domain and bounce tracking.

**What Outlet does automatically**

- Registers your sending domain with Amazon SES as soon as a brand has a From Email.
- Turns on DKIM signing (a signature that proves the mail is really from you) and creates its keys.
- Sets up a MAIL FROM subdomain, such as \`mail.acme.com\`, so bounces come back to your own domain.
- Sets up bounce and complaint tracking, so bounced and complaining addresses stop getting mail.
- Checks every minute until the domain is verified, and updates the page by itself.

**What needs a person, and why**

- **An AWS account with a card on file**, because Amazon bills you directly.
- **An AWS user and its keys**, because only you can sign in to AWS.
- **Adding the DNS records** at your DNS host, because only you can sign in there. Outlet shows each record to copy. (If your domain's DNS is on DigitalOcean, Outlet writes records only for outreach domains, not for Amazon SES domains.)
- **Asking Amazon for production access**, because new AWS accounts start in sandbox mode and only Amazon can lift it.

## 1. Create the AWS user and keys

1. Sign in to the AWS console and open **IAM** (Identity and Access Management).
2. Create a new user, for example "outlet". It doesn't need console access.
3. Attach a policy with Outlet's permissions. Outlet shows the policy under **Required IAM Policy** in the setup wizard and in **Settings**, **Integrations**. Press **Copy Policy**, paste it into a new policy in IAM, and attach it.
4. Also allow these actions, which Outlet uses for the bounce domain and bounce tracking:

   \`\`\`
   "ses:SetIdentityMailFromDomain",
   "ses:GetIdentityMailFromDomainAttributes",
   "ses:GetIdentityNotificationAttributes",
   "sns:CreateTopic",
   "sns:Subscribe"
   \`\`\`

   Without them, sending still works, but the MAIL FROM records are left off the list and bounces aren't tracked.
5. If you don't use cloud backups, you can remove the S3 part of the policy. If you do, replace \`YOUR-BACKUP-BUCKET\` with your bucket's name.
6. Create an access key for the user and copy the **Access Key ID** and **Secret Access Key**. AWS shows the secret only once.

## 2. Give the keys to Outlet

**For the whole Outlet** (every brand that sends through "This server's email settings"):

1. Open Outlet's own **Settings**: on the **Brands** screen, click your initial at the top right and choose **Settings**. Then open the **Integrations** tab.
2. Choose the **Amazon SES** tab.
3. Pick the **AWS Region**, closest to your audience, and paste the **Access Key ID** and **Secret Access Key**.
4. Press **Save SES Settings**. You see "AWS Credentials Configured".

The setup wizard's **Connect Amazon SES** step saves the same thing.

**For one brand only**, with its own AWS account: open the brand's **Settings**, **Sending**, press **Add pathway**, pick "Your own Amazon SES" under **Sends through**, fill in **AWS region**, **Access key** and **Secret key**, tick what it **Carries**, and press **Add pathway**. See [How sending works](/help/how-sending-works).

## 3. Verify your domain

This starts by itself when a brand gets a From Email. To start it by hand, or to pick the bounce subdomain:

1. Open the brand's **Settings**, **Email**.
2. Under **Domain Verification**, press **Verify Domain**.
3. **MAIL FROM Subdomain**: leave "mail" unless your domain already uses \`mail.\` for something else; then try "bounce", "email" or "ses".
4. Press **Verify Domain**.

Outlet lists the records under **Add these DNS records to your domain**, each with its **Type**, **Name/Host**, **Value** and **Purpose**, and a copy button:

| Purpose | Type | What it does |
| --- | --- | --- |
| verification | TXT at \`_amazonses.yourdomain\` | Proves to Amazon that you own the domain |
| dkim | 3 CNAME records ending in \`._domainkey\` | DKIM signing keys, so receivers trust your mail |
| mail_from | MX and TXT at your MAIL FROM subdomain | Where bounces go, and the SPF record that allows Amazon SES to send |
| dmarc | TXT at \`_dmarc\` | Your DMARC policy, starting gently with \`p=none\` |

## 4. Add the records and wait

1. Sign in at your DNS host (often where you bought the domain).
2. Add each record exactly as shown. Use the copy buttons. Some DNS hosts add your domain to the end of the name by themselves; if yours does, type only the part before your domain.
3. Back in Outlet, press **Refresh**, or just wait: Outlet checks every minute.

The badges next to **Verification:** and **DKIM:** read **Pending** while Amazon waits for the records and **Verified** when it finds them. DNS changes usually show up within an hour, but can take up to 72 hours.

## 5. Leave the sandbox

New AWS accounts start in **sandbox mode**. In sandbox mode, Amazon SES only sends to addresses you have verified in AWS, and only a few a day.

1. In the AWS console, open Amazon SES in the same region you chose in Outlet.
2. Find "Request production access" on the account dashboard.
3. Describe what you send: who signs up, how they opt in, and how you handle bounces and unsubscribes. Outlet handles the last two for you.

Amazon usually answers within a day. Until then, test with your own verified addresses.

## Common problems

- **Verification stays Pending for hours**: a record is missing or mistyped. Compare each record at your DNS host with Outlet's. Watch for the domain added twice, such as \`_amazonses.acme.com.acme.com\`.
- **Failed**: Amazon gave up looking. Fix the records, then press **Refresh**, or delete the domain with the trash icon and press **Verify Domain** again.
- **No MAIL FROM records listed**: the AWS user lacks the extra actions in step 1. Add them, delete the domain and verify it again.
- **"Another brand already uses (domain)"**: a domain belongs to one brand only.
- **"... outreach domains never send through SES"**: outreach domains are kept away from Amazon SES on purpose. See [Outreach overview](/help/outreach-overview).
- **The brand sends through an SMTP relay**: there is nothing to verify at Amazon; Outlet marks the domain verified straight away. Set up the relay's own DNS records as its documentation says.
`,h=`# Set up outreach end to end

This guide takes you from nothing to your first outreach emails and replies. Read [Outreach overview](/help/outreach-overview) first if you haven't: it explains the rules this setup follows.

**What needs a person, and why**

- **Turning outreach on** needs the brand's admin. This is on purpose: no AI assistant or API key can turn it on.
- **Nameservers at your domain registrar** (the company you bought the domain from) need you, because only you can sign in there.
- **A mailbox service account, its API key and a card on file** need you, because the service bills you directly.

Everything else can be done for you, by Outlet or by an AI assistant connected over MCP.

## 1. Turn outreach on

1. Open **Outbound**, then the **Setup** tab.
2. Turn on the switch at the top. It reads **Outbound is on**.

Only the brand's admin can do this. If you can't, ask the admin, or see [Users and roles](/help/users-and-roles).

## 2. Set your mailing address

Still on **Setup**, under **Settings**:

1. **Mailing address**: your business postal address, such as "123 Main St, Denver, CO 80202". Every outreach email carries it, and no campaign can be created without it.
2. **Opt-out line**: the sentence at the end of each email telling people how to stop. The default is "If this isn't for you, reply "no thanks" and I won't email again."
3. **Time zone**: a name like "America/Denver". Sending hours follow this time zone.
4. Press **Save settings**. You see "Settings saved".

## 3. Check the pathway that carries outreach

Outreach must go through your mailboxes, never through Amazon SES.

1. Open **Settings**, then **Sending**.
2. Find the pathway that **Sends through** "Your connected mailboxes" and has **Outreach** ticked under **Carries**.
3. If there is none, press **Add pathway**, give it a **Name** such as "Mailboxes", pick "Your connected mailboxes" under **Sends through**, tick **Outreach**, and press **Add pathway**.

Outlet warns "Amazon SES and most email platforms forbid cold email in their terms and can suspend the account" if outreach is ticked on any other kind of pathway. Take that warning seriously.

## 4. Get outreach domains

An outreach domain is a domain used only for outreach, such as "getacme.com" when your brand is "acme.com". Outlet refuses your brand's own sending domain, so its reputation is never at risk. Plan on one domain for every two or three mailboxes.

Pick the first way that fits you. They are in order of how much is done for you.

**A. Let a mailbox service do it (most automatic).** Maildoso, CheapInboxes and Infraforge can buy a domain for you, or take over one you own, and they write all of its mail records. You set their nameservers at your registrar once. The domain then shows up on **Setup** under **Domains** by itself. See [Connect a mailbox service](/help/connect-a-mailbox-service).

**B. Your domain's DNS is on DigitalOcean (automatic records).** Outlet writes the mail records for you.

1. In DigitalOcean, create a personal access token with read and write access to domains. It starts with "dop_v1_".
2. On **Setup**, under **DigitalOcean DNS token**, paste it and press **Save token**. A **Set** badge appears.
3. Under **Domains**, press **Add domain**, type the **Domain**, and pick "DigitalOcean" under **DNS host**.
4. Fill in **Mail provider** and, if you have it, **DKIM value** (see C below), then press **Add domain**.

Outlet adds only the records that are missing and never adds a second SPF, DKIM or DMARC record. If DigitalOcean refuses, you see "DigitalOcean didn't take the records" with the records to add by hand.

**C. You add the records yourself.**

1. Under **Domains**, press **Add domain**.
2. **Domain**: the outreach domain.
3. **DNS host**: "I add records myself".
4. **Mail provider**: "Google Workspace", or "Other" for anything else. For Other, also fill in **MX hosts** and **SPF include** from your provider's setup page.
5. **DKIM selector** and **DKIM value**: from your mail provider's admin console. For Google Workspace the selector is usually "google".
6. **DMARC reports to (optional)**: an address to receive DMARC reports, or leave it empty.
7. **Click tracking**: leave it off unless you need it. It adds a "go." subdomain for tracked links.
8. Press **Add domain**. Outlet lists the records under **Add these records at your DNS host**. Click a value to copy it, and add each record at your DNS host.

What these records are is explained in [Your sending domain and DNS explained](/help/sending-domain-and-dns).

**Check the domain.** Press **Check** next to the domain. It shows **Ready** when MX, SPF, DKIM and DMARC are all found, or **Missing records** with what is missing. DNS changes can take from minutes to a day to show up. Outlet also checks every domain daily. If records go missing later, it pauses that domain's mailboxes and resumes them once the records are back.

## 5. Add mailboxes

Each mailbox is a real inbox with a real person's name, on one of your outreach domains.

**From a mailbox service.** Create the mailboxes in {{label.connectionsScreen}} (see [Connect a mailbox service](/help/connect-a-mailbox-service)). Outlet tests each login and adds it for you.

**By hand**, for Google Workspace, Microsoft 365 or any mailbox with SMTP and IMAP:

1. On **Setup**, under **Mailboxes**, press **Add mailbox**.
2. **Email**: it must be on one of your outreach domains.
3. **From name**: the real person who stands behind this mailbox.
4. **Signature (optional)**.
5. **Provider**: "Google Workspace", or "Custom (SMTP and IMAP)" for others.
6. **App password** (Google) or **Password** (custom).
7. **Own daily limit (optional)**: leave it empty to use the pool's limit.
8. Press **Add mailbox**. The button reads "Checking the login..." while Outlet signs in to send and to read replies. It saves only when both work.

Details for Google and Microsoft are in [Add Google Workspace or Microsoft 365 mailboxes by hand](/help/google-workspace-and-microsoft-365).

A new mailbox warms up. It sends 5 on its first day and a little more each day. Use it normally for about two weeks before outreach if you can. See [Warm-up and limits](/help/warm-up-and-limits).

## 6. Add prospects

1. Open the **Prospects** tab and press **Add prospects**.
2. Type one person a line, as \`email, name, source\`. For example \`jane@example.com, Jane Doe, YouTube About page\`. The source says where you found the address, and it is required.
3. **List (optional)**: leave it empty to use the "Outreach" list.
4. Press **Add**.

Each line comes back as added, held, duplicate, suppressed or refused, with the reason. Outlet checks every address. Risky addresses are held and never sent. Addresses that bounced, complained or opted out before are suppressed.

## 7. Create your first campaign

1. Open **Campaigns** and press **New campaign**.
2. **Name**: for your own reference.
3. **Outreach list**: normally "Outreach".
4. Under **Emails**, write **Step 1**: a **Subject** and a **Body**. Use merge tags such as \`{{first_name}}\`, add a fallback with \`{{first_name,fallback=there}}\`, and vary wording with \`{Hi|Hey}\`. The first email may have at most one link.
5. Press **Add follow-up** for steps 2 and 3. Leave a follow-up's **Subject** empty to reply in the same thread. Set **Days after previous** (at least 1, default 3). Up to 5 steps; 3 over about 12 days works well.
6. **Approval**: keep "Approve every email" at first.
7. **Send from (hour)** and **Send until (hour)**: default 8 to 17, in your time zone. Weekends are skipped.
8. **Track opens** and **Track clicks**: leave off. Reply rate is the measure that counts, and tracking hurts cold delivery.
9. Under **Prospects**, tick up to 100 people. Only people free to enroll are listed: in no other campaign and not suppressed.
10. Press **Preview** to see the first emails without saving anything. Fix anything that looks wrong.
11. Press **Create**.

If a prospect lacks a value a merge tag needs, Outlet skips that person rather than sending "Hi ,".

## 8. Approve drafts

1. Open **Drafts**. Each email shows who it goes to, from which mailbox, and when it sends.
2. Press **Edit** to change a **Subject** or **Body**, then **Save**.
3. Press **Approve**, or **Skip** to drop it. To act on many at once, tick them and press **Approve selected** or **Skip selected**.

Approved emails go out inside your sending hours, a few minutes apart, within each mailbox's daily limit. Nothing asks "are you sure", so read before you approve.

## 9. Handle replies

1. Open **Replies**. **Needs attention** shows replies waiting for you.
2. Outlet sorts most replies itself: bounces, spam blocks, out of office, unsubscribes, complaints and "not interested". The rest show **Not sorted**.
3. Use **Sort as** to record what a reply means: "Interested", "Not interested", "Question", "Out of office", "Wrong person", "Unsubscribe" or "Complaint".
4. Press **Answer** to write back. It sends in the same thread from the mailbox that got the reply.
5. Press **Mark as partner** when someone says yes.

Any reply stops that person's follow-ups. Opt-outs go on your suppression list for good.

## 10. Watch the Overview

The **Overview** tab shows whether outreach is **On**, **Paused** or **Off**, a **Next steps** list, today's sends against today's limit, and each mailbox's bounces, spam blocks and complaints. Press **Explain** next to a mailbox for a plain answer to why it is or isn't sending. **Pause all** stops everything for the brand at once; **Resume all** starts it again.

If a mailbox is paused, see [Warm-up and limits](/help/warm-up-and-limits) and [Troubleshooting](/help/troubleshooting).
`,g=`# Run several mailbox services together

A brand can connect more than one mailbox service, and add mailboxes by hand alongside them. Outlet sends from all of them as one pool. This guide explains when that helps and how it works.

## Why use more than one

- **Spread the risk.** If one service has an outage, changes its terms or closes your account, the other mailboxes keep sending.
- **Mix mailbox types.** Recipients on Google and on Microsoft treat mail from different mailbox types differently. A mix delivers more evenly.
- **Use mailboxes you already have.** Add your own Google Workspace or Microsoft 365 mailboxes next to bought ones.

## How it works

- **Each connection is one service with its own API key.** You can connect Maildoso, CheapInboxes and Infraforge together, and use **{{label.manualService}}** for mailboxes you add by hand.
- **Each outreach domain belongs to one connection.** A domain bought or connected at Maildoso stays with your Maildoso connection. Outlet refuses to put the same domain under two connections.
- **One pool sends from every mailbox.** The pathway that carries **Outreach** sends from all mailboxes of all connections. Each prospect gets one mailbox for their whole sequence, picked from the least-used active ones.
- **One set of limits, with exceptions.** The pool's daily limit, warm-up, spacing and health pauses apply to every mailbox. A single mailbox can have its own daily limit.

## Set it up

1. Open {{label.connectionsScreen}}, find **{{label.connectionsHeading}}** and press **{{label.addConnection}}** once for each service. See [Connect a mailbox service](/help/connect-a-mailbox-service).
2. In each connection, connect or buy domains and create mailboxes.
3. Add any hand-made mailboxes on **Outbound**, **Setup**, under **Mailboxes**. See [Add Google Workspace or Microsoft 365 mailboxes by hand](/help/google-workspace-and-microsoft-365).
4. Open **Outbound**, **Overview** to see every mailbox in one table, with today's sends and its health.

## Respect each service's own limit

The pool's limit applies to every mailbox, but services set different limits in their terms. Maildoso's terms cap cold email at 15 a day per mailbox, while Outlet's normal outreach default is 30.

When you mix services, set each mailbox to the strictest limit its service allows:

1. Open **Outbound**, **Setup**, and find the mailbox under **Mailboxes**.
2. Type a number under **Own daily limit** and press **Save**. You see "Daily limit saved".
3. Clear the box and save to go back to the pool's limit.

Or set the pool's **Emails a day** to the lowest of your services' limits in **Settings**, **Sending**. See [Warm-up and limits](/help/warm-up-and-limits).

## Remove a service

1. Delete its mailboxes with **{{label.deleteMailbox}}** under **{{label.deleteSection}}** (with the service's name), or leave them and cancel at the service. Deleted mailboxes stay in Outlet, paused for good, so their threads are kept.
2. Prospects already assigned to those mailboxes don't move to another mailbox mid-sequence. Their follow-ups wait while the mailbox is paused.

## Common problems

- **"(domain) belongs to another mailbox pool of this brand."**: a domain lives under one connection only.
- **One service's mailboxes are all paused**: usually DNS at that service. Press **Explain** on one of them in **Outbound**, **Overview** for the reason.
`,_=`# Transactional email from your app

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
3. **Slug**: the short name your app uses to pick this email, such as \`password-reset\`.
4. **Subject**: it can hold variables, such as \`Reset your password, {{name}}\`.
5. Press **Create and edit**.
6. Write the **HTML body**, and the **Plain-text version** (shown by mail apps that don't display HTML; it helps delivery).
7. Use \`{{variable}}\` wherever your app supplies a value, such as \`{{reset_link}}\`. Fill in **Sample values** to see them in the **Preview**.
8. Press **Save changes**.
9. Under **Send a test**, press **Send test**. It sends the saved version.

Empty sender fields use the brand's settings. A template switched off shows "Inactive: sends are refused".

## Send through the API

Your developer sends a \`POST\` to \`https://your-outlet-address/sdk/v1/emails\` with the header \`X-API-Key: your brand API key\` and a JSON body:

\`\`\`json
{
  "to": "jane@example.com",
  "template_slug": "password-reset",
  "variables": { "name": "Jane", "reset_link": "https://..." }
}
\`\`\`

Without a template, send \`subject\` and \`body\` (HTML) instead. Other fields: \`text_body\`, \`from_name\`, \`from_email\`, \`reply_to\`, \`tags\`, \`meta\` and \`attachments\`.

The answer has a \`message_id\` and a \`status\`: \`queued\`, \`sent\`, \`failed\`, \`suppressed\` (the person bounced, complained or is on the suppression list, so it wasn't sent) or \`limited\` (over the pathway's limit; try again later).

Outlet's **Settings** (on the **Brands** screen, under your initial at the top right), **API** tab, shows the base address and code examples in cURL, Node.js, Python, Go, PHP and SMTP.

## Send over SMTP

Many apps and plugins can already send email over SMTP. Outlet can receive it and send it on.

**This needs the person who runs your Outlet server**: SMTP is off until \`SMTP_ENABLED=true\` is set on the server.

Then, in your app's email settings:

| Setting | Value |
| --- | --- |
| Host | your Outlet address |
| Port | 587 (or the server's \`SMTP_PORT\`) |
| Encryption | STARTTLS |
| Username | the brand's slug, such as \`acme\` |
| Password | the brand's API key |

The brand's **Settings**, **SMTP** tab shows these for your brand. Messages can be up to 25 MB with up to 100 recipients. The brand's From address is used, whatever From your app sets. Mail sent over SMTP is treated as transactional.

## Common problems

- **"Missing API key"** or **"Invalid API key"**: the \`X-API-Key\` header is missing or the key was regenerated.
- **"Template '(slug)' not found"** or **"is inactive"**: check the slug on the **Transactional** page, and that the template is active.
- **"Subject is required when not using a template"**: send \`template_slug\`, or both \`subject\` and \`body\`.
- **Status \`suppressed\`**: the person bounced, complained or is on your suppression list. Check **Blocklist**.
- **SMTP "invalid credentials"**: the username must be the slug of the same brand the API key belongs to.
- **Nothing arrives**: see [Troubleshooting](/help/troubleshooting).
`,v=`# Troubleshooting

When mail doesn't go out, lands in spam, or a mailbox stops, start here. Each section says where to look first and what to do.

## Mail is not sending

Work down this list. Most problems are found in the first three steps.

1. **Is there a pathway for this kind of mail?** Open the brand's **Settings**, **Sending**. Each kind of mail (**Transactional**, **Marketing**, **Outreach**) must be ticked on one pathway. A kind no pathway carries isn't sent at all. See [How sending works](/help/how-sending-works).
2. **Is the sending service set up?** For "This server's email settings", check Outlet's **Settings**, **Integrations** shows "AWS Credentials Configured" or a saved SMTP relay. For your own Amazon SES or relay, check the keys on the pathway.
3. **Is the domain verified?** Open the brand's **Settings**, **Email**. **Verification:** and **DKIM:** should read **Verified**. See [Set up Amazon SES](/help/set-up-amazon-ses).
4. **Is Amazon SES still in sandbox mode?** In sandbox mode it only sends to addresses verified in AWS. Request production access in the AWS console.
5. **Is the person blocked?** Outlet never sends to an address that hard-bounced or complained, or that is in **Blocklist**, **Suppressed Emails**, or at a blocked domain. An API send to them comes back as \`suppressed\`.
6. **Is a limit holding it back?** Mail over a pathway's daily limit waits; it isn't lost. An API send comes back as \`limited\`. See [Warm-up and limits](/help/warm-up-and-limits).
7. **Campaign paused?** Outlet pauses a campaign after 100 errors in a row. Fix the cause above, then send again.

## Outreach is not sending: use Explain

Open **Outbound**, **Overview** and read the status at the top first:

- **Off**: outreach isn't turned on. The brand's admin turns it on in **Outbound**, **Setup**.
- **Paused**: all outreach is paused. Press **Resume all**.
- **On**: look at **Next steps** and at the mailboxes.

Then press **Explain** next to a mailbox. Outlet answers in plain words, for example:

- "... is active and warming up: 7 allowed today, climbing each day to 30 a day; it sent 7." The mailbox is fine; it reached today's warm-up limit.
- "... is paused because (domain) is missing ... Fix those records, then resume the mailbox."
- "... is paused to protect its reputation: ..." See "A mailbox is paused" below.

Other common reasons outreach waits:

- **Drafts waiting**: emails need approval. Open **Drafts**.
- **Outside sending hours**: each campaign sends only between its **Send from (hour)** and **Send until (hour)**, on weekdays, in the brand's time zone.
- **Spacing**: each mailbox waits 4 to 9 minutes between emails.

An AI assistant connected over MCP can also explain one email ("why didn't this send?") or one prospect ("why can't they be enrolled?").

## DNS is failing

- **Brand domain stays Pending**: compare each record at your DNS host with Outlet's, character by character. Watch for the domain added twice, such as \`_dmarc.acme.com.acme.com\`. Give it an hour, then press **Refresh**.
- **Outreach domain shows Missing records**: press **Records** next to the domain on **Outbound**, **Setup** to see each record, then **Check**. "Missing:" lists what's not found.
- **Two SPF records**: receivers ignore both. Merge them into one.
- **"DigitalOcean has no zone for this domain"**: the domain's nameservers don't point to DigitalOcean yet, or it isn't added in DigitalOcean. Fix that, then add the domain again.
- **"DigitalOcean refused the token"**: make a new token with read and write access to domains and save it on **Setup**.
- **Domain at a mailbox service stays Waiting**: the nameservers at your registrar aren't set to the service's yet. Changes can take up to a day.

[Your sending domain and DNS explained](/help/sending-domain-and-dns) explains each record.

## A mailbox is paused

The reason is shown under **Paused** on **Outbound**, **Overview**.

- **"DNS: ..."**: fix the records. The mailbox resumes by itself once they pass.
- **"Health: bounce rate ..."**: too many addresses didn't exist. Validate your prospects before adding them, and drop sources that give bad addresses.
- **"Health: ... spam blocks"**: receiving servers refused the mail as spam. Write shorter, plainer, more personal emails, with no links in the first one, and fewer a day.
- **"Health: ... complaints"**: people pressed "Report spam". Target more carefully and keep the opt-out line clear.
- **"Deleted: at (service) on (date)"**: the mailbox was deleted at its mailbox service. It stays paused for good.

A health pause lifts only once the mailbox's last 7 days are back under the limits. See [Warm-up and limits](/help/warm-up-and-limits).

## Bounces and spam blocks

- **A bounce** is an email that couldn't be delivered. After a hard bounce (the address doesn't exist), Outlet never sends to that address again.
- **A spam block** is a receiving server refusing your mail because it looks like spam, often with a code like 5.7.1.
- **A complaint** is someone pressing "Report spam". Outlet never sends to that address again.

To bring them down:

1. Validate lists before sending, and suppress bad addresses. See [Clean lists, validation and the blocklist](/help/clean-lists).
2. Use double opt-in on your lists.
3. Check SPF, DKIM and DMARC all pass.
4. Remove people who never open or click, with **Housekeeping**.
5. For outreach, send fewer a day per mailbox and add mailboxes instead.

## Still stuck

- Check the top of the screen for a red message: Outlet's error messages say what to do next.
- Ask an AI assistant connected over MCP: it can read your setup and explain what's wrong. See [Connect an AI assistant over MCP, and API keys](/help/mcp-and-api-keys).
- Whoever runs your Outlet server can read its logs for the full error.
`,y=`# Use your own SMTP server

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
4. Press **Save SMTP Settings**.

**Common providers**

| Provider | SMTP Host | Port | Username |
| --- | --- | --- | --- |
| SendGrid | smtp.sendgrid.net | 587 | \`apikey\` (the word itself), with your API key as the password |
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

- **Mail doesn't go out, or authentication fails**: check the username and password at the relay. For SendGrid the username is literally \`apikey\`.
- **Connection refused or times out**: check the host and port. Some hosting companies block port 25; use 587.
- **"ENCRYPTION_KEY is not set on this server, so it can't store credentials"**: whoever installed Outlet needs to set \`ENCRYPTION_KEY\` on the server.
- **Mail lands in spam**: the relay's SPF and DKIM records are probably missing. Add them at your DNS host.
`,b=`# Users and roles

Outlet is invite-only. Each person signs in with their own account, and what they can do depends on their role. This guide explains the roles and how to add people.

## The roles

**Platform roles**, for the whole Outlet:

- **Super admin**: runs every brand and the platform itself. Only a super admin manages users, platform settings (such as Amazon SES and SMTP relay), backups, updates and new brands. The account made in the setup wizard is a super admin.
- **User**: reaches only the brands they're invited to, with the role they have there.

**Brand roles**, for one brand:

- **Member**: runs the brand's everyday mail: contacts, lists, sequences, designs, campaigns and sends.
- **Admin**: everything a member can do, plus the brand itself: its settings, sending setup, keys, webhooks, people, turning outreach on, and anything that deletes or clears its data.

## Invite someone to a brand

Brand admins do this.

1. Open the brand's **Settings**, **People**.
2. Under **Invite someone**, type the **Email address** and pick the **Role**: **Member** or **Admin**.
3. Press **Invite**.

Outlet emails an invitation. It works once and expires; the page shows the date. If you see "Invitation email did not go out", the brand's sending isn't working yet: check **Settings**, **Sending**, then invite again.

## Add a user to Outlet

Super admins do this.

1. Open Outlet's own **Settings**: on the **Brands** screen, click your initial at the top right and choose **Settings**.
2. Open the **Users** tab and press **Add User**.
3. Fill in **Name**, **Email**, **Password** and **Role** ("User" or "Super admin"), and press **Add User**.

**Edit** changes a user, including switching them **Active** or **Inactive**. **Delete** removes them for good.

## Common problems

- **"You don't have access to this brand."**: ask the brand's admin to invite you.
- **"This needs the brand's admin role."**: ask an admin, or to be made an admin.
- **"Only a super admin can do this."**: platform settings, users, backups and new brands need a super admin.
- **"This account is not active."**: a super admin switched the account off. Ask them to switch it back on.
`,x=`# Warm-up and limits

Outlet protects each sender's reputation with a daily limit, a warm-up that starts slow, spacing between emails, and automatic pauses when something looks wrong. This guide covers the defaults, how to change them, and what to do when a mailbox is paused.

## Why limits matter

Google, Microsoft and other mail providers judge every sender. A new mailbox that suddenly sends a hundred emails looks like a spammer, and its mail lands in spam for weeks. A mailbox that starts with a few emails a day and grows slowly builds trust.

## The defaults for outreach mailboxes

| Setting | Default |
| --- | --- |
| Emails a day, per mailbox | 30 |
| Warm-up first day | 5 |
| Warm-up added each day | 2 |
| Time between sends | 4 to 9 minutes, at random |
| Sending hours | Each campaign's hours, 8 to 17 by default, weekdays only |

So a new mailbox sends at most 5 on its first day, 7 on the second, 9 on the third, and reaches 30 a day after about two weeks. The warm-up counts from the day the mailbox was connected.

**Maildoso mailboxes** follow Maildoso's terms instead: 15 a day, warming up from 5, adding 1 a day. Choosing Maildoso sets those limits. See [Connect a mailbox service](/help/connect-a-mailbox-service).

**Amazon SES, SMTP relays and the platform pathway** have no daily limit, warm-up or spacing by default; those services limit sending on their side.

## Change the pool's limits

1. Open **Settings**, **Sending**.
2. On the pathway that carries **Outreach**, press **Edit**.
3. Under **Limits per sender**, change **Emails a day**, **Warm-up first day**, **Warm-up added each day**, **Seconds between sends, at least** or **Seconds between sends, at most**. A 0 turns a limit off.
4. Press **Save**.

Only the brand's admin can change limits. Changing a pathway's mailbox service resets its limits to that service's defaults.

## Give one mailbox its own daily limit

1. Open **Outbound**, **Setup**.
2. Under **Mailboxes**, find the mailbox and type a number under **Own daily limit**.
3. Press **Save**. You see "Daily limit saved".

Clear the number and press **Save** to go back to the pool's limit ("The mailbox uses its pool's daily limit"). The warm-up still applies: a mailbox's own limit is the top it climbs to.

The **Overview** tab shows each mailbox's limit in words, such as "Warming up: 9 today, climbing to 30 a day (the pool's)".

## Health pauses

Outlet watches each mailbox over the last 7 days and pauses it by itself when it crosses a line:

- **Bounce rate above 3%**, once it has at least 2 bounces. A bounce is an email that couldn't be delivered, usually because the address doesn't exist.
- **2 spam blocks.** A spam block is a receiving server refusing the mail as spam.
- **2 complaints.** A complaint is someone pressing "Report spam".

Change these in **Settings**, **Sending**, under **Pause a sender over 7 days at**.

Only that one mailbox pauses. The others keep sending. Outlet also pauses every mailbox on a domain whose DNS records go missing, and resumes them by itself once the records are back.

## What to do when a mailbox is paused

1. Open **Outbound**, **Overview**. Paused mailboxes show **Paused** with the reason, and the row is red.
2. Press **Explain** next to the mailbox. Outlet says in plain words why it is paused and how to fix it.
3. Fix the cause:
   - **"Health: bounce rate ..."**: your prospect list has bad addresses. Check where they came from, and run them through **Validation** before using them again (see [Clean lists, validation and the blocklist](/help/clean-lists)).
   - **"Health: ... spam blocks"**: receivers think the mail is spam. Shorten the emails, remove links from the first email, write more personally, and slow down.
   - **"Health: ... complaints"**: people are reporting the mail. Target more carefully and make the opt-out line clear.
   - **"DNS: (domain) is missing ..."**: add the missing records, then press **Check** next to the domain on **Setup**. The mailbox resumes by itself.
4. Press **Resume** once the problem is fixed.

A mailbox paused for health can only resume once its last 7 days are back under the limits. Until then Outlet refuses with "... is still over a health limit (...); it can resume once the 7-day window clears". This is on purpose: sending more from a damaged mailbox damages it further.

**Pause all** at the top of **Overview** stops all outreach for the brand. **Resume all** starts it again. Campaigns can be paused one at a time on their own page.

## Common problems

- **Sent today is far below the limit**: the mailbox is still warming up, it is outside the campaign's sending hours or a weekend, or there are no approved emails waiting. Press **Explain**.
- **Nothing sends at all**: check the status at the top of **Overview**: **Off** means outreach isn't turned on, **Paused** means everything is paused.
`,S=`# Webhooks

A webhook lets Outlet tell another system the moment something happens, such as an email bouncing or someone unsubscribing. Outlet sends a short message to a web address you choose. Use it to keep your CRM, your app's database or an automation tool like Zapier or Make in step.

## Add a webhook

1. Open the brand's **Settings**, **Webhooks**.
2. Press **Add Webhook**.
3. **Endpoint URL**: the web address that receives the messages. Your developer or automation tool gives you this.
4. **Events to Subscribe**: tick what to send.
   - **Contacts**: **Contact Created**, **Contact Unsubscribed**.
   - **Email**: **Email Sent**, **Email Delivered**, **Email Bounced**, **Marked as Spam**, **Email Opened**, **Email Clicked**.
5. Keep **Active** ticked and press **Create Webhook**.
6. Copy the **Signing Secret** now. Outlet shows it once: "Save this secret now. You won't be able to see it again."

Only the brand's admin can manage webhooks.

## What Outlet sends

Each event is one \`POST\` to your address with a JSON body holding \`event\`, \`timestamp\` and \`data\`, and these headers:

- \`X-Webhook-Signature\`: a signature of the body, made with your signing secret (HMAC-SHA256, in hex). Your system recomputes it to be sure the message came from Outlet.
- \`X-Webhook-Event\`: the event name, such as \`email.bounced\`.
- \`X-Webhook-ID\`: the webhook's id.

The event names are \`contact.created\`, \`contact.unsubscribed\`, \`email.sent\`, \`email.delivered\`, \`email.bounced\`, \`email.complained\` (Marked as Spam), \`email.opened\` and \`email.clicked\`.

Any answer in the 200s counts as delivered. Outlet waits up to 30 seconds and sends each event once; it doesn't retry. If your system was down, the event is missed, so have it catch up from the API when it comes back.

## Test and watch

Each webhook's row has:

- **Test Webhook**: sends a test event now and shows whether it worked.
- **View Logs**: every delivery with its **Event**, **Status**, **Duration** and **Time**.
- **Disable** and **Enable**: pause it without deleting it.
- **Delete**: remove it.

The **Deliveries** column counts successes and failures.

## Common problems

- **Every delivery fails**: open **View Logs**. Check the address is public (not \`localhost\`), uses \`https\`, and answers within 30 seconds.
- **The signature doesn't match**: sign the raw body exactly as received, before your system parses it, with the secret from when you created the webhook.
- **Lost the secret**: delete the webhook and create it again to get a new one.
`,C={connectionsScreen:`Settings, Sending`,connectionsHeading:`Mailbox services`,addConnection:`Add mailbox service`,connectionService:`Service`,connectionApiKey:`API key`,saveConnection:`Save`,manualService:`By hand (SMTP and IMAP logins)`,connectDomainSection:`Connect a domain you own`,connectDomain:`Connect`,buySection:`Buy`,buyKindMailboxes:`Mailboxes on a domain`,buyKindDomain:`A new domain`,buyKindConnect:`Connect a domain I own (Infraforge)`,buyPeople:`One mailbox a line: First Last, or First Last, local`,getQuote:`Get a quote`,buy:`Buy`,syncSection:`Sync with`,syncInventory:`What (service name) has`,syncCompare:`Compare`,syncImport:`Import missing mailboxes`,deleteSection:`Delete at`,deleteMailbox:`Delete`},w=/\{\{label\.([A-Za-z]+)\}\}/g;function T(e){return e.replace(w,(e,t)=>C[t]??e)}var E=[{section:`Start here`,slugs:[`getting-started`,`how-sending-works`,`set-up-amazon-ses`,`use-your-own-smtp-server`,`sending-domain-and-dns`]},{section:`Outreach`,slugs:[`outreach-overview`,`set-up-outreach`,`connect-a-mailbox-service`,`several-mailbox-services`,`google-workspace-and-microsoft-365`,`warm-up-and-limits`]},{section:`Email marketing`,slugs:[`campaigns-and-newsletters`,`sequences-and-autoresponders`,`transactional-email`,`lists-and-subscribers`,`clean-lists`,`designs`]},{section:`Connect and manage`,slugs:[`webhooks`,`mcp-and-api-keys`,`users-and-roles`,`privacy-and-deleting-data`,`backups-and-updates`]},{section:`When something goes wrong`,slugs:[`troubleshooting`]}],D=Object.assign({"./guides/backups-and-updates.md":e,"./guides/campaigns-and-newsletters.md":t,"./guides/clean-lists.md":n,"./guides/connect-a-mailbox-service.md":r,"./guides/designs.md":i,"./guides/getting-started.md":a,"./guides/google-workspace-and-microsoft-365.md":o,"./guides/how-sending-works.md":s,"./guides/lists-and-subscribers.md":c,"./guides/mcp-and-api-keys.md":l,"./guides/outreach-overview.md":u,"./guides/privacy-and-deleting-data.md":d,"./guides/sending-domain-and-dns.md":f,"./guides/sequences-and-autoresponders.md":p,"./guides/set-up-amazon-ses.md":m,"./guides/set-up-outreach.md":h,"./guides/several-mailbox-services.md":g,"./guides/transactional-email.md":_,"./guides/troubleshooting.md":v,"./guides/use-your-own-smtp-server.md":y,"./guides/users-and-roles.md":b,"./guides/warm-up-and-limits.md":x,"./guides/webhooks.md":S});Object.keys(D).map(e=>e.replace(`./guides/`,``).replace(/\.md$/,``));function O(e,t){let n=D[`./guides/${e}.md`];if(n===void 0)throw Error(`Help guide ${e} has no file`);let r=T(n),i=r.split(`
`),a=i.findIndex(e=>e.startsWith(`# `)),o=a>=0?i[a].slice(2).trim():``,s=i.slice(a+1),c=s.findIndex(e=>e.trim()!==``),l=s.findIndex((e,t)=>t>c&&e.trim()===``);return{slug:e,section:t,title:o,summary:s.slice(c,l<0?void 0:l).join(` `).replace(/\*\*|`|\[|\]\([^)]*\)/g,``).trim(),markdown:r,text:r.toLowerCase()}}var k=E.flatMap(({section:e,slugs:t})=>t.map(t=>O(t,e))),A=E.map(({section:e})=>e);function j(e){return k.find(t=>t.slug===e)}function M(e){let t=e.toLowerCase().split(/\s+/).filter(Boolean);if(t.length===0)return k;let n=k.filter(e=>t.every(t=>e.text.includes(t))),r=e=>t.every(t=>e.title.toLowerCase().includes(t));return[...n.filter(r),...n.filter(e=>!r(e))]}function N(e,t){let n=t.toLowerCase().split(/\s+/).find(Boolean),r=e.markdown.replace(/^#+ .*$/gm,``).replace(/^\s*(?:[-*]|\d+\.)\s+/gm,``).replace(/\|/g,` `).replace(/\*\*|`|\[|\]\([^)]*\)/g,``).replace(/\s+/g,` `).trim();if(!n)return e.summary;let i=r.toLowerCase().indexOf(n);if(i<0)return e.summary;let a=Math.max(0,i-60),o=Math.min(r.length,i+140);return`${a>0?`...`:``}${r.slice(a,o)}${o<r.length?`...`:``}`}export{N as a,A as i,k as n,M as r,j as t};