# Warm-up and limits

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
