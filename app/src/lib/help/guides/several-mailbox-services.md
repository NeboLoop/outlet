# Run several mailbox services together

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
