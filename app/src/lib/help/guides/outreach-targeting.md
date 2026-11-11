# Targeting, materials and who is never contacted

Who your outreach is for, what it can point to, and who it never writes to. This guide covers the **Targeting**, **Materials** and **Do not contact** tabs, the daily pace, hot leads and the leads export, and how drafts show who wrote them.

## Brand profile

The **Targeting** tab starts with your brand profile: your website, what you sell, who buys it, your proof (customers, results, numbers) and how your emails sound. Outreach is written from it. If an AI assistant is connected over MCP, it can read your website and fill the profile in; you read it here and correct it.

## Segments

A segment is a group of people a campaign is for, such as "Dental clinics in Utah". It has a name, a plain description of who it targets, and filters: countries, regions, industries, company sizes, title keywords and custom fields.

- **Prospects belong to segments.** When you or the AI add prospects, you name the segment. Prospects outside its filters are noted when they are added, never refused: the filters guide whoever finds the people.
- **A campaign runs on a segment.** In **New campaign**, pick the segment, then pick prospects, or turn on **Everyone in the segment** to enroll everyone free to enroll (up to 100 at a time).
- **Results break down by segment** on the **Overview** tab: prospects, campaigns, replies and interested people for each.

## Materials

The **Materials** tab holds what outreach can point to: case studies, proof points, offers, links and assets. Each has a short summary, and can be for one segment or for all of them. When a campaign is written, the AI gets the profile, the segment and its materials to write from. A campaign pins the materials it used, and its page lists them under **Written from**, so you can check its claims against them.

## Who is never contacted

Before any outreach is queued, and again as each follow-up goes, every person is checked. The first match is the reason, and anyone skipped says why: in the campaign's preview, when prospects are added, and on the campaign page under **Stopped early**.

1. **Suppression**: your suppression list, past hard bounces and spam complaints.
2. **Your own domains**: your sending and outreach domains, and the company domains the server protects.
3. **Your customers**: anyone on one of your lists (unsubscribed too), anyone sent a transactional email, and anyone who was your contact before outreach. Turn on **Skip everyone at a customer's company** to skip their colleagues too. Free mail domains like gmail.com never count.
4. **Do-not-contact lists**: lists you upload, of addresses or whole domains, such as "Customers" or "Open deals". Unlike the suppression list, they only stop outreach: receipts and newsletters still go.
5. **The outside check** (optional): an address in your CRM or another system that answers yes or no. Outlet sends it `email`, `domain`, `brand` and `timestamp` as JSON, signed in the `X-Outlet-Signature` header (HMAC-SHA256 of the body with your secret), and reads `{"do_not_contact": true or false, "reason": "..."}`. While it doesn't answer, nobody new is enrolled and emails already in a campaign wait: nothing is sent until it answers.

**Check an address** on the **Do not contact** tab to see which of these, if any, stops it.

## Daily pace

The pace, in **Setup**, is how many new people get their first email each day across your mailboxes: **Light** (25), **Typical** (50), **Growth** (100) or **Unlimited**, or a number of your own. Each choice shows how many emails a day it comes to once follow-ups catch up, and about how many mailboxes that needs. Past the day's number, first emails wait for the next day's sending window and follow-ups still go, so nothing is lost. The first time each day, webhooks get `outreach.paused` (see [Webhooks](/help/webhooks)), so you hear about it without checking.

Outlet has no price per email: mailbox services bill per mailbox or domain, priced when you get a quote.

## Hot leads and the leads export

A hot lead is a reply sorted as **Interested**. On the **Replies** tab, turn on **Hot leads only** to see just them. **Export leads** saves a spreadsheet with one row per person per campaign: email, name, company, title, segment, campaign, where they stand, why they stopped, how their reply was sorted and when it came.

## Who wrote each email

Every outreach email records who wrote it: **Written by AI**, **AI, edited by a person**, or **Written by a person**. Drafts and campaigns show it, so you know which emails need the closest read. Anything an AI writes over MCP is marked as AI; editing it yourself marks it as edited.

## Starting from your website

An AI assistant connected over MCP can go from your website to drafted outreach in a few steps: read the site, fill in the profile, propose segments for you to approve, add materials, add prospects to a segment, then draft a campaign for your approval. It asks Outlet for the steps with `explain` and the topic `start_from_website`. See [MCP and API keys](/help/mcp-and-api-keys).
