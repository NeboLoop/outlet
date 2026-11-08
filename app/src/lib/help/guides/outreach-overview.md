# Outreach overview

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
