# Clean lists, validation and the blocklist

Sending to bad or uninterested addresses hurts your reputation with mail providers, and then even your good mail lands in spam. Three screens keep your lists clean: **Validation** checks addresses, **Blocklist** stops mail to people and domains, and **Housekeeping** removes people who no longer read your mail.

## Validation

Validation checks whether an address can receive mail, without sending anything.

**Check one address**

1. Open **Validation**.
2. Type the address under **Check an email** and press **Validate**.
3. Read the result, and any "Did you mean ...?" suggestion for typos such as "gmial.com".

**Check a whole list or file**

1. Under **Bulk jobs**, press **New Job**.
2. Pick **From a list**, or **Upload CSV** and choose the **Email column**.
3. **Job name (optional)**, then press **Start Validation**.
4. Open the job when it's done. **Export CSV** downloads the results.
5. Press **Suppress** to stop mail to the bad addresses. **Invalid**, **Spamtrap**, **Abuse** and **Do not mail** are ticked by default. Suppressed addresses get no email at all.

**What the results mean**

- **Valid**: the address exists.
- **Invalid**: it doesn't exist. Don't send.
- **Catch-all**: the domain accepts every address, so the check can't tell. Send with care.
- **Unknown**: the server didn't answer. Try again later.
- **Spamtrap**: an address kept to catch spammers. Never send.
- **Abuse**: belongs to someone known to report mail as spam. Don't send.
- **Do not mail**: shared role addresses such as info@, throwaway addresses, forwarding services and likely traps. Don't send.

## Blocklist

**Suppressed Emails**: addresses that get no mail from this brand, whatever list they're on. Outlet adds outreach opt-outs here itself. To add one by hand, use **Add Email to Suppression List**. Separately, Outlet never sends to an address that hard-bounced or complained, even if it isn't listed here.

**Blocked Domains**: whole domains, such as a competitor's. Outlet refuses new subscribers from a blocked domain and won't send to it. Use **Block a Domain**.

## Housekeeping

Housekeeping deletes contacts for good. Always press **Preview** first and read the number.

**Inactive Contacts** removes confirmed, still-subscribed people who haven't opened anything in **No opens in (days)** (90 by default) and haven't clicked anything in **No clicks in (days)** (180 by default). Every kind of email counts: campaigns, sequences, outreach and transactional. People newer than the windows are kept. Removing them improves your open rates and deliverability.

**Unconfirmed Subscriptions** removes contacts Outlet hasn't marked as verified, older than **Older than (days)** (30 by default). Be careful: today Outlet marks very few contacts as verified. People you imported or added by hand, people confirmed through double opt-in, and outreach prospects can all count as unverified. If the **Preview** number is bigger than the people you expect never confirmed, don't delete.

After **Preview**, press **Delete N Contact(s)** or **Delete N Subscription(s)**. Deleted contacts can't be recovered, so take a backup first (see [Backups and updates](/help/backups-and-updates)).

## Common problems

- **High bounces after an import**: validate the list, then suppress the bad addresses.
- **Someone says they never get your mail**: search for them in **Blocklist**, **Suppressed Emails**. Also ask whether their address ever bounced or they reported your mail as spam: Outlet then stops sending to them.
