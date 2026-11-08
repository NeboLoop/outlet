# Privacy and deleting data

People have the right to see the data you hold about them, and to have it deleted. Privacy laws such as the GDPR in Europe require it. This guide covers finding someone's data, their consent, exporting it, and deleting it.

## Find a person

1. Open the brand's **Settings**, **Privacy**.
2. Under **GDPR Compliance Tools**, type their email address and press **Search Contact**.

You see their **Contact Information**: email, contact ID and when they were added.

## Record consent

Under **Consent Preferences**, switch **GDPR Consent** and **Marketing Consent** to **Granted** or **Not Granted**, then press **Save Consent Changes**. Use this when someone tells you, by email or phone, what they agree to.

## Export someone's data

Under **Export Personal Data**, press **Export Data**. Outlet gathers their profile, consent, tags and activity (campaigns, sequence emails, transactional emails and clicks) into one JSON file, and your browser downloads it. Nothing is kept on the server. To save it again, press **Download Export**. The MCP gdpr tool's export gives the same data.

## Delete someone's data

1. Under **Delete All Data**, press **Delete Data**.
2. To also ask your email providers to delete what they keep about the address, tick **Also ask the brand's email providers to delete what they keep about this address**.
3. Confirm. This can't be undone.

Only the brand's admin can delete. Each deletion is recorded in the GDPR audit log, without the address itself, and the message after it names its audit record.

**What the providers box does**: Amazon SES takes the address off the SES account's suppression list, the only record about one person SES lets Outlet delete; SES keeps no sending log that can be deleted. SMTP relays and mailboxes have no way to be asked, so nothing is sent to them. After deleting, Outlet lists what each provider did ("What each email provider did"), so you can ask a relay's operator yourself. Outlet keeps its own bounce and suppression records for the address either way, so it isn't mailed again.

**What is deleted**: the person, their list memberships, tags, custom field values, sequence progress, waiting emails, clicks and campaign records.

**What is kept**: records of transactional emails sent to them keep the email address, unlinked from the person, as a record that the mail was sent.

**Important**: deleting someone does not stop them being added again. If they sign up again, or are in a file you import later, they come back. If they asked never to be emailed again, also add their address in **Blocklist**, **Suppressed Emails** after deleting.

## Unsubscribes

People unsubscribe with the link in every marketing email. You choose in each list's **Settings** whether that leaves **This list only** or **All lists**. Unsubscribed people are never added back by an import.

## Common problems

- **The search finds nobody**: check the spelling, and that you're in the right brand.
- **The export download doesn't start**: your browser may have blocked it. Press **Download Export** to save it again, or allow downloads from your Outlet address.
