# Connect a mailbox service

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

Outlet needs its `ENCRYPTION_KEY` set on the server to store keys. Without it you see "ENCRYPTION_KEY is not set on this server, so it can't store credentials". Ask whoever installed Outlet to set it.

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
3. For mailboxes, list the people in the box marked "{{label.buyPeople}}", one a line: `First Last`, or `First Last, local` to choose the part before the @. Without it, the address is first.last@yourdomain. Up to 100 mailboxes an order.
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

- **API key**: from CheapInboxes' dashboard. It starts with `ci_live_`.
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
