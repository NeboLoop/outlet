# Run several mailbox services together

A mailbox pool can hold several connections side by side: different mailbox services, two accounts of the same service, and mailboxes you add by hand. Outlet sends from all of them as one pool. This guide explains when that helps and how it works.

## Why use more than one

- **Spread the risk.** If one service has an outage, changes its terms or closes your account, the other mailboxes keep sending.
- **Mix mailbox types.** Recipients on Google and on Microsoft treat mail from different mailbox types differently. A mix delivers more evenly.
- **Use mailboxes you already have.** Add your own Google Workspace or Microsoft 365 mailboxes next to bought ones.

## How it works

- **Each connection is one service account with its own API key.** Use Maildoso, CheapInboxes and Infraforge together, add a second account of one of them, and use **{{label.manualServiceShort}}** for mailboxes you add yourself.
- **Each outreach domain belongs to one connection.** A domain bought or connected at Maildoso stays with that Maildoso connection, and its mailboxes come from it.
- **One pool sends from every mailbox.** Sending draws on every connection's mailboxes. Each prospect keeps one mailbox for their whole sequence, picked from the least-used active ones.
- **One set of limits, with exceptions.** The pool's warm-up, spacing and health pauses apply to every mailbox. A connection can have its own daily limit, and a single mailbox can have its own on top.

## Run two services side by side

1. Open {{label.connectionsScreen}} and find the pool that carries **Outreach**.
2. Under **{{label.connectionsHeading}}**, press **{{label.addConnection}}**, choose the **{{label.connectionService}}**, add a **{{label.connectionName}}**, paste the **{{label.connectionApiKey}}** and press **{{label.addConnectionSave}}**. The key is checked before it is saved.
3. Do the same for the second service. It can also be a second account of the same service.
4. On each connection, press **{{label.manage}}**. Either **{{label.connectDomain}}** a domain you own and set the nameservers it shows at your registrar, or buy one: pick the kind, enter the domain, press **{{label.getQuote}}**, then **{{label.buy}}**. On Infraforge, connecting a domain is a purchase.
5. Get mailboxes: **{{label.buy}}** with kind "{{label.buyKindMailboxes}}" (one person a line, "First Last" or "First Last, local"), or **{{label.syncImport}}** for ones made at the service.
6. Open **Outbound**, **Overview** to see every mailbox in one table, with today's sends and its health.

Step by step details, and each service's terms, are in [Connect a mailbox service](/help/connect-a-mailbox-service).

## Add hand-made mailboxes alongside

1. Add a connection with **{{label.connectionService}}** "{{label.manualService}}". It needs no API key.
2. Add the domain and its mailboxes in **Outbound**, **Setup**. See [Add Google Workspace or Microsoft 365 mailboxes by hand](/help/google-workspace-and-microsoft-365).

Domains added there join the pool's first hand-added connection. To put a domain on another connection, press **{{label.manage}}** on the connection that holds it, pick the target under **{{label.moveTo}}** and press **{{label.move}}**.

## Respect each service's own limit

Services set different limits in their terms. Maildoso's terms cap cold email at 15 a day per mailbox, while Outlet's normal outreach default is 30.

- A **Maildoso connection** starts at 15 a day per mailbox. Change it in the connection's **{{label.manage}}**, under **{{label.connectionLimit}}**.
- Other connections use the pool's limit until you set **{{label.connectionLimit}}**.
- One mailbox can have its own limit: on **Outbound**, **Setup**, under **Mailboxes**, type a number under **Own daily limit** and press **Save**.

The most specific limit wins: a mailbox's own, then its connection's, then the pool's. See [Warm-up and limits](/help/warm-up-and-limits).

## Remove a connection

1. Press **{{label.manage}}** on it.
2. Move each of its domains to another connection of the same pool with **{{label.moveTo}}** and **{{label.move}}**.
3. Press **{{label.removeConnection}}** and confirm. Outlet forgets the connection and its key; nothing changes at the service, so cancel there too if you're done with it.

To stop using a service's mailboxes instead, delete them under **{{label.deleteSection}}** (with the service's name). Deleted mailboxes stay in the pool, paused for good, so their threads are kept. Prospects already on a mailbox don't move to another one mid-sequence; their follow-ups wait while it is paused.

## Common problems

- **Remove is refused**: the connection still holds domains. Move them first.
- **A domain "is already in use"**: a domain belongs to one connection and one brand.
- **One service's mailboxes are all paused**: usually DNS at that service. Press **Explain** on one of them in **Outbound**, **Overview** for the reason.
