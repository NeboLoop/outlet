# Campaigns and newsletters

A campaign is one email sent to one or more of your lists, such as a newsletter, an announcement or a sale. This guide covers writing it, choosing who gets it, testing, sending or scheduling, and reading the results.

Campaigns are **marketing** mail. They go only to people who joined your lists, never to outreach prospects, and they go out through the pathway that carries **Marketing** in **Settings**, **Sending**. See [How sending works](/help/how-sending-works).

## Create a campaign

1. Open **Campaigns** and press **New Campaign**.
2. Type a name at the top, where it says "Campaign Name". Only you see it.
3. **Subject Line**: what people see first in their inbox.
4. **Preview Text**: the short line shown after the subject in most inboxes.
5. Write the email in the editor. Merge tags put each person's details in, such as `{{name}}`, or `{{name,fallback=Friend}}` for people without a name. `{{unsubscribe_url}}` and `{{web_version_url}}` are also available.
6. **Recipients**: tick one or more lists. Outlet shows "Total: N recipients".
7. **Exclude Lists** (optional): tick lists whose members should not get this email, such as existing customers for a sign-up offer.
8. **Sender (optional)**: **From Name**, **From Email** and **Reply-To Email**. Leave them blank to use the brand's defaults.
9. **Design**: "None" sends your content as it is. Pick a design to wrap your content in it. See [Designs](/help/designs).
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
