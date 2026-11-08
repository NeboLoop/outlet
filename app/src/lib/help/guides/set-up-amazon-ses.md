# Set up Amazon SES

Amazon SES (Simple Email Service) is Amazon's email sending service. It is low cost, handles large volumes, and Outlet sets up most of it for you through Amazon's own API: domain verification, DKIM signing, the bounce domain and bounce tracking.

**What Outlet does automatically**

- Registers your sending domain with Amazon SES as soon as a brand has a From Email.
- Turns on DKIM signing (a signature that proves the mail is really from you) and creates its keys.
- Sets up a MAIL FROM subdomain, such as `mail.acme.com`, so bounces come back to your own domain.
- Sets up bounce and complaint tracking, so bounced and complaining addresses stop getting mail.
- Checks every minute until the domain is verified, and updates the page by itself.

**What needs a person, and why**

- **An AWS account with a card on file**, because Amazon bills you directly.
- **An AWS user and its keys**, because only you can sign in to AWS.
- **Adding the DNS records** at your DNS host, because only you can sign in there. Outlet shows each record to copy. (If your domain's DNS is on DigitalOcean, Outlet writes records only for outreach domains, not for Amazon SES domains.)
- **Asking Amazon for production access**, because new AWS accounts start in sandbox mode and only Amazon can lift it.

## 1. Create the AWS user and keys

1. Sign in to the AWS console and open **IAM** (Identity and Access Management).
2. Create a new user, for example "outlet". It doesn't need console access.
3. Attach a policy with Outlet's permissions. Outlet shows the policy under **Required IAM Policy** in the setup wizard and in **Settings**, **Integrations**. Press **Copy Policy**, paste it into a new policy in IAM, and attach it.
4. Also allow these actions, which Outlet uses for the bounce domain and bounce tracking:

   ```
   "ses:SetIdentityMailFromDomain",
   "ses:GetIdentityMailFromDomainAttributes",
   "ses:GetIdentityNotificationAttributes",
   "sns:CreateTopic",
   "sns:Subscribe"
   ```

   Without them, sending still works, but the MAIL FROM records are left off the list and bounces aren't tracked.
5. If you don't use cloud backups, you can remove the S3 part of the policy. If you do, replace `YOUR-BACKUP-BUCKET` with your bucket's name.
6. Create an access key for the user and copy the **Access Key ID** and **Secret Access Key**. AWS shows the secret only once.

## 2. Give the keys to Outlet

**For the whole Outlet** (every brand that sends through "This server's email settings"):

1. Open Outlet's own **Settings**: on the **Brands** screen, click your initial at the top right and choose **Settings**. Then open the **Integrations** tab.
2. Choose the **Amazon SES** tab.
3. Pick the **AWS Region**, closest to your audience, and paste the **Access Key ID** and **Secret Access Key**.
4. Press **Save SES Settings**. You see "AWS Credentials Configured".

The setup wizard's **Connect Amazon SES** step saves the same thing.

**For one brand only**, with its own AWS account: open the brand's **Settings**, **Sending**, press **Add pathway**, pick "Your own Amazon SES" under **Sends through**, fill in **AWS region**, **Access key** and **Secret key**, tick what it **Carries**, and press **Add pathway**. See [How sending works](/help/how-sending-works).

## 3. Verify your domain

This starts by itself when a brand gets a From Email. To start it by hand, or to pick the bounce subdomain:

1. Open the brand's **Settings**, **Email**.
2. Under **Domain Verification**, press **Verify Domain**.
3. **MAIL FROM Subdomain**: leave "mail" unless your domain already uses `mail.` for something else; then try "bounce", "email" or "ses".
4. Press **Verify Domain**.

Outlet lists the records under **Add these DNS records to your domain**, each with its **Type**, **Name/Host**, **Value** and **Purpose**, and a copy button:

| Purpose | Type | What it does |
| --- | --- | --- |
| verification | TXT at `_amazonses.yourdomain` | Proves to Amazon that you own the domain |
| dkim | 3 CNAME records ending in `._domainkey` | DKIM signing keys, so receivers trust your mail |
| mail_from | MX and TXT at your MAIL FROM subdomain | Where bounces go, and the SPF record that allows Amazon SES to send |
| dmarc | TXT at `_dmarc` | Your DMARC policy, starting gently with `p=none` |

## 4. Add the records and wait

1. Sign in at your DNS host (often where you bought the domain).
2. Add each record exactly as shown. Use the copy buttons. Some DNS hosts add your domain to the end of the name by themselves; if yours does, type only the part before your domain.
3. Back in Outlet, press **Refresh**, or just wait: Outlet checks every minute.

The badges next to **Verification:** and **DKIM:** read **Pending** while Amazon waits for the records and **Verified** when it finds them. DNS changes usually show up within an hour, but can take up to 72 hours.

## 5. Leave the sandbox

New AWS accounts start in **sandbox mode**. In sandbox mode, Amazon SES only sends to addresses you have verified in AWS, and only a few a day.

1. In the AWS console, open Amazon SES in the same region you chose in Outlet.
2. Find "Request production access" on the account dashboard.
3. Describe what you send: who signs up, how they opt in, and how you handle bounces and unsubscribes. Outlet handles the last two for you.

Amazon usually answers within a day. Until then, test with your own verified addresses.

## Common problems

- **Verification stays Pending for hours**: a record is missing or mistyped. Compare each record at your DNS host with Outlet's. Watch for the domain added twice, such as `_amazonses.acme.com.acme.com`.
- **Failed**: Amazon gave up looking. Fix the records, then press **Refresh**, or delete the domain with the trash icon and press **Verify Domain** again.
- **No MAIL FROM records listed**: the AWS user lacks the extra actions in step 1. Add them, delete the domain and verify it again.
- **"Another brand already uses (domain)"**: a domain belongs to one brand only.
- **"... outreach domains never send through SES"**: outreach domains are kept away from Amazon SES on purpose. See [Outreach overview](/help/outreach-overview).
- **The brand sends through an SMTP relay**: there is nothing to verify at Amazon; Outlet marks the domain verified straight away. Set up the relay's own DNS records as its documentation says.
