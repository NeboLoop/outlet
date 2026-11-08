# Set up outreach end to end

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

**From a mailbox service.** In {{label.connectionsScreen}}, press **{{label.manage}}** on the service's connection and buy mailboxes, or press **{{label.syncImport}}** (see [Connect a mailbox service](/help/connect-a-mailbox-service)). Outlet tests each login and adds it for you.

**By hand**, for Google Workspace, Microsoft 365 or any mailbox with SMTP and IMAP. If the pool has no connection yet, first add one with **{{label.connectionService}}** "{{label.manualServiceShort}}" in {{label.connectionsScreen}} (see [Run several mailbox services together](/help/several-mailbox-services)). Then:

1. On **Setup**, under **Mailboxes**, press **Add mailbox**.
2. **Email**: it must be on one of your outreach domains.
3. **From name**: the real person who stands behind this mailbox.
4. **Signature (optional)**.
5. **Provider**: "Google Workspace", or "Custom (SMTP and IMAP)" for others.
6. **App password** (Google) or **Password** (custom).
7. **Own daily limit (optional)**: leave it empty to use its connection's or the pool's limit.
8. Press **Add mailbox**. The button reads "Checking the login..." while Outlet signs in to send and to read replies. It saves only when both work.

Details for Google and Microsoft are in [Add Google Workspace or Microsoft 365 mailboxes by hand](/help/google-workspace-and-microsoft-365).

A new mailbox warms up. It sends 5 on its first day and a little more each day. Use it normally for about two weeks before outreach if you can. See [Warm-up and limits](/help/warm-up-and-limits).

## 6. Add prospects

1. Open the **Prospects** tab and press **Add prospects**.
2. Type one person a line, as `email, name, source`. For example `jane@example.com, Jane Doe, YouTube About page`. The source says where you found the address, and it is required.
3. **List (optional)**: leave it empty to use the "Outreach" list.
4. Press **Add**.

Each line comes back as added, held, duplicate, suppressed or refused, with the reason. Outlet checks every address. Risky addresses are held and never sent. Addresses that bounced, complained or opted out before are suppressed.

## 7. Create your first campaign

1. Open **Campaigns** and press **New campaign**.
2. **Name**: for your own reference.
3. **Outreach list**: normally "Outreach".
4. Under **Emails**, write **Step 1**: a **Subject** and a **Body**. Use merge tags such as `{{first_name}}`, add a fallback with `{{first_name,fallback=there}}`, and vary wording with `{Hi|Hey}`. The first email may have at most one link.
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
