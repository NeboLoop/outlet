# Connect a mailbox service

A mailbox service sells ready-made mailboxes for outreach and runs their domains' mail records for you. Connect one to Outlet with its API key, and Outlet can buy domains, connect domains you own, create mailboxes and bring them in, without you typing a single DNS record or password.

This is the most automatic way to set up outreach. Outlet supports **Maildoso**, **CheapInboxes** and **Infraforge**. For mailboxes you already have, such as Google Workspace or Microsoft 365, see [Add Google Workspace or Microsoft 365 mailboxes by hand](/help/google-workspace-and-microsoft-365).

**What needs a person, and why**

- **Signing up at the service and adding a card** needs you, because the service bills you directly. Outlet never holds your card.
- **Copying the API key** needs you, because only you can sign in to the service.
- **Setting nameservers at your registrar** needs you when you connect a domain you already own, because only you can sign in to the company you bought the domain from.
- **Saying yes to every purchase** needs you. Outlet shows a price quote first and spends nothing until you confirm.

## How it fits together

- Your outreach mailboxes form a **pool**: the pathway in **Settings**, **Sending** that carries **Outreach**.
- The pool has a **{{label.connectionsHeading}}** list. Each connection is one service account with its own API key. A pool can hold any number, even two accounts of the same service. See [Run several mailbox services together](/help/several-mailbox-services).
- Each outreach domain belongs to one connection, and its mailboxes come from that connection.
- The pool sends from every mailbox of every connection, under the pool's limits and warm-up. See [Warm-up and limits](/help/warm-up-and-limits).

## Add a connection

1. Sign up at the service and add a payment method there.
2. In the service's dashboard, find its API settings and create an API key. Copy it.
3. In Outlet, open {{label.connectionsScreen}} and find the pool that carries **Outreach**.
4. Under **{{label.connectionsHeading}}**, press **{{label.addConnection}}**.
5. **{{label.connectionService}}**: pick Maildoso, CheapInboxes or Infraforge.
6. **{{label.connectionName}}**: a name you'll recognize, such as "Maildoso main". Left empty, it is named after the service.
7. **{{label.connectionApiKey}}**: paste the key.
8. Press **{{label.addConnectionSave}}**. Outlet checks the key with the service before it saves, and stores it encrypted. It is never shown again.

The connection's row shows its name, service, whether its key is set, its domains, its mailbox count and its daily limit. Press **{{label.manage}}** on it for everything below.

If the key is wrong you see the service's own refusal. Copy the key again, with nothing before or after it.

Outlet needs its `ENCRYPTION_KEY` set on the server to store keys. Without it you see "ENCRYPTION_KEY is not set on this server, so it can't store credentials". Ask whoever installed Outlet to set it.

## Connect a domain you own

Use this when you already bought an outreach domain somewhere else, such as Namecheap or GoDaddy.

1. Press **{{label.manage}}** on the connection.
2. Under **{{label.connectDomainSection}}**, type the domain and press **{{label.connectDomain}}**.
3. Outlet shows the service's note and **Nameservers**.
4. Sign in at your domain registrar and replace the domain's nameservers with the ones shown. (Nameservers decide which company answers for the domain's DNS. See [Your sending domain and DNS explained](/help/sending-domain-and-dns).)
5. Wait. The service takes over the domain's DNS and writes its MX, SPF, DKIM and DMARC records itself.

On Infraforge, connecting a domain is a purchase: use **Buy** below with "{{label.buyKindConnect}}".

The domain appears on **Outbound**, **Setup**, under **Domains**, by itself. Outlet refuses your brand's own sending domain and any domain another brand uses.

## Buy a domain or mailboxes

Every purchase is two steps: a price quote, then your yes.

1. Press **{{label.manage}}** on the connection, and go to **{{label.buySection}}** (with the service's name).
2. Pick what to buy: "{{label.buyKindMailboxes}}", "{{label.buyKindDomain}}" or "{{label.buyKindConnect}}".
3. Type the domain.
4. For mailboxes, list the people one a line ({{label.buyPeople}}). For example `Sam Rivera`, or `Pat Rivera, pat` to choose the part before the @. Without it, the address is first.last@yourdomain. Up to 100 mailboxes an order.
5. Press **{{label.getQuote}}**. Outlet shows the price, or "Price unknown" when the service doesn't say, and when the quote expires. A quote belongs to that one connection, lasts 30 minutes and is used once.
6. Read it, then press **{{label.buy}}**. Outlet asks once more, with the amount, before anything is spent.

After the buy, Outlet tests each new mailbox's login and adds it to your pool. Mailboxes the service hasn't finished yet are listed as waiting: "Sync and import once the service has finished them."

## Sync

Use **{{label.syncSection}}** (with the service's name) when you made mailboxes in the service's own dashboard, or a purchase was still finishing.

- **{{label.syncInventory}}** lists the domains and mailboxes you have at the service.
- **{{label.syncCompare}}** shows what the service has that this connection doesn't, and the reverse.
- **{{label.syncImport}}** brings in every finished mailbox Outlet is missing, after testing its login.

## Change or remove a connection

Under **{{label.manage}}**:

- **{{label.connectionLimit}}**: the daily limit for this connection's mailboxes. Empty uses the pool's.
- Change the **Name** or paste a new **API key**, then press **{{label.saveConnection}}**.
- Each domain has **{{label.moveTo}}** and **{{label.move}}**: the domain moves with its mailboxes to another connection of the same pool.
- **{{label.removeConnection}}**: first move its domains to another connection. Outlet then forgets the connection and its key. Nothing changes at the service, so cancel there too if you're done with it.

## Delete a mailbox

Under **{{label.deleteSection}}** (with the service's name), press **{{label.deleteMailbox}}** next to the mailbox and confirm. The service deletes it (CheapInboxes cancels it at the end of its billing period instead). Maildoso and Infraforge delete its messages too. In Outlet the mailbox stays in the pool, paused for good, so its threads and history are kept.

## Maildoso

- **API key**: a personal access token from Maildoso's dashboard.
- **Automatic**: connects domains you own for free, buys new domains, creates mailboxes, and runs DNS for its domains.
- **Connecting a domain**: Maildoso replaces the domain's MX, SPF, DKIM and DMARC records. Set the nameservers it gives within 24 hours. It marks the domain active, usually within 2 hours.
- **Costs**: Maildoso's API gives no prices. Domains and mailboxes are billed to the payment method on your Maildoso account and plan, so the quote shows "Price unknown". Check prices in Maildoso before you buy.
- **Its limits**: Maildoso's terms cap cold email at **15 a day per mailbox**, after **14 days of warm-up**, to **business addresses only**. A Maildoso connection starts at 15 a day per mailbox, and its mailboxes warm up under the pool's warm-up (from 5 a day by default). Change it in the connection's **{{label.connectionLimit}}**, but going above Maildoso's terms risks your account.

## CheapInboxes

- **API key**: from CheapInboxes' dashboard. It starts with `ci_live_`.
- **Automatic**: sells Google Workspace mailboxes, buys new domains with their mailboxes, and runs DNS for its domains.
- **Connecting a domain**: free. The domain is staged first, and the move finishes in the order that buys its first mailboxes. CheapInboxes then gives the nameservers to set at your registrar.
- **Costs**: quotes show real prices in US dollars: what is charged today to the card on file, then a monthly price for the mailboxes.
- **Deleting**: cancels the mailbox at the end of its billing period. It keeps working until then.
- **Its limits**: follow CheapInboxes' own terms for sending volume. The connection uses the pool's limits: 30 a day by default, warming up from 5, adding 2 a day.

## Infraforge

- **API key**: from Infraforge's dashboard.
- **Automatic**: creates mailboxes on your domains, and runs DNS for domains connected to it.
- **Connecting a domain**: Infraforge bills for it, so it is a purchase. Use **{{label.buySection}}** with "{{label.buyKindConnect}}", get a quote and buy. Then set the nameservers it gives at your registrar.
- **Buying a domain**: not possible from Outlet. Registering a domain at Infraforge needs the owner's contact details, so buy it in Infraforge's own app, then sync.
- **Costs**: Infraforge invoices mailboxes and connections, and its API gives no price before the invoice. Check prices in Infraforge first.
- **Its limits**: follow Infraforge's own terms. The connection uses the pool's limits: 30 a day by default, warming up from 5, adding 2 a day.

## AI assistants

An AI assistant connected over MCP can do all of this with the `setup_sending` tool: list, add, change and remove connections, move domains, and quote, buy, sync and delete on a connection. It must show you a quote and get your yes before it buys.

## Common problems

- **"That was refused"** or the service's own message: the service said no. The message says why, such as a low balance or a domain that isn't free.
- **A domain "is already in use"**: each domain belongs to one connection and one brand.
- **Remove is refused**: move the connection's domains to another connection first.
- **Mailboxes stay waiting after a buy**: the service hasn't finished them. Wait, then press **{{label.syncImport}}**.
- **Mailboxes paused for DNS**: the nameservers aren't set yet, or the service hasn't finished the records. Set the nameservers and wait. Outlet checks daily and resumes the mailboxes by itself. Press **Check** on **Outbound**, **Setup** to check sooner.
