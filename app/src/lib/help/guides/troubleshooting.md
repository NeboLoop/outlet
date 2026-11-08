# Troubleshooting

When mail doesn't go out, lands in spam, or a mailbox stops, start here. Each section says where to look first and what to do.

## Mail is not sending

Work down this list. Most problems are found in the first three steps.

1. **Is there a pathway for this kind of mail?** Open the brand's **Settings**, **Sending**. Each kind of mail (**Transactional**, **Marketing**, **Outreach**) must be ticked on one pathway. A kind no pathway carries isn't sent at all. See [How sending works](/help/how-sending-works).
2. **Is the sending service set up?** For "This server's email settings", check Outlet's **Settings**, **Integrations** shows "AWS Credentials Configured" or a saved SMTP relay. For your own Amazon SES or relay, check the keys on the pathway.
3. **Is the domain verified?** Open the brand's **Settings**, **Email**. **Verification:** and **DKIM:** should read **Verified**. See [Set up Amazon SES](/help/set-up-amazon-ses).
4. **Is Amazon SES still in sandbox mode?** In sandbox mode it only sends to addresses verified in AWS. Request production access in the AWS console.
5. **Is the person blocked?** Outlet never sends to an address that hard-bounced or complained, or that is in **Blocklist**, **Suppressed Emails**, or at a blocked domain. An API send to them comes back as `suppressed`.
6. **Is a limit holding it back?** Mail over a pathway's daily limit waits; it isn't lost. An API send comes back as `limited`. See [Warm-up and limits](/help/warm-up-and-limits).
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

- **Brand domain stays Pending**: compare each record at your DNS host with Outlet's, character by character. Watch for the domain added twice, such as `_dmarc.acme.com.acme.com`. Give it an hour, then press **Refresh**.
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
