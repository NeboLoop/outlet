# Getting started

This guide takes you from a fresh Outlet install to your first email. It covers the setup wizard, your first brand, and the settings that decide how your mail goes out.

**Words used in this guide**

- **Brand**: one business or product you send email for. Each brand has its own lists, designs, settings and API key. One Outlet can run many brands.
- **Amazon SES**: Amazon's email sending service. It is low cost and handles large volumes. Outlet can set up most of it for you through Amazon's own API.
- **SMTP relay**: another company's email sending server, such as SendGrid, Mailgun or Postmark.

## 1. Run the setup wizard

The first time you open Outlet, it shows the setup wizard. It has three steps: **Account**, **Email** and **Complete**.

**Create Admin Account**

1. Fill in **Company**, **Name**, **Email Address**, **Password** (at least 8 characters) and **Confirm Password**.
2. Check **Timezone**. Outlet fills it in from your computer. It is used to schedule campaigns and show report times.
3. Press **Continue**. You are signed in automatically.

This first account is a super admin: it can do everything, including adding other users.

**Connect Amazon SES**

1. Pick the **AWS Region** closest to your audience. The default is "US East (N. Virginia)".
2. Paste the **Access Key ID** and **Secret Access Key** of an AWS user that has Outlet's permissions. [Set up Amazon SES](/help/set-up-amazon-ses) shows how to make one, step by step.
3. Press **Complete Setup**.

Prefer another sending company, or don't have AWS yet? Press **Skip for now**. You can connect Amazon SES or an SMTP relay later in **Settings**, **Integrations**. See [Use your own SMTP server](/help/use-your-own-smtp-server).

**Complete**

You see "You're all set!". Press **Go to Dashboard**.

## 2. Create your first brand

1. On the **Brands** screen, press **New Brand** (or **Create brand** when you have none yet).
2. **Name**: your business or product name.
3. Under **Email Settings**:
   - **From Name**: the name people see in their inbox, such as "Acme" or "Sam at Acme".
   - **From Email**: the address mail comes from, such as `hello@acme.com`. Use a domain you own, not a gmail.com address.
   - **Reply To (optional)**: where replies go. Left empty, replies go to the From Email.
4. Press **Create**, then click the new brand to open it.

You can also add a brand from the **Brands** menu at the top, with **Add brand**. That asks for a **Brand Name** and **URL Slug** (the short name in the brand's web address, such as `/acme`, which can't be changed later), then opens a welcome screen asking for the same **From Name**, **From Email** and **Reply-To Email (optional)**. Press **Save & Continue** there.

## 3. What happens automatically

When a brand gets a From Email, Outlet asks Amazon SES to verify its domain, in the background. It then lists the DNS records the domain needs, in the brand's **Settings**, **Email**, under **Domain Verification**. See [Set up Amazon SES](/help/set-up-amazon-ses).

You can change the sender later in the brand's **Settings**, **Email**, under **Sender Identity**.

## 4. Add the DNS records

**This step needs you**, because only you can sign in to the company that runs your domain's DNS (often where you bought the domain).

1. Open the brand's **Settings**, then **Email**.
2. Under **Domain Verification**, copy each record into your DNS host.
3. Press **Refresh** to check. Outlet also checks every minute by itself, and the page updates when the domain is **Verified**.

[Your sending domain and DNS explained](/help/sending-domain-and-dns) explains each record in plain words.

## 5. Check how your mail is sent

Open the brand's **Settings**, **Sending**. A new brand has one pathway, "Platform email", that sends **Transactional** and **Marketing** mail through the server's email settings. For most people that is right and there is nothing to change. [How sending works](/help/how-sending-works) explains pathways and when to add another.

## 6. Send your first email

- **A newsletter**: create a list in **Lists**, add subscribers, then create a campaign in **Campaigns**. See [Lists, subscribers and importing](/help/lists-and-subscribers) and [Campaigns and newsletters](/help/campaigns-and-newsletters).
- **Email from your app**, such as receipts or password resets: see [Transactional email from your app](/help/transactional-email).
- **One-to-one outreach** to people who haven't signed up: see [Outreach overview](/help/outreach-overview).

Before your first real send, use **Send Test** in the campaign editor to send yourself a copy.

## Common problems

- **New AWS accounts start in "sandbox mode".** In sandbox mode, Amazon SES only sends to addresses you have verified in AWS. Request production access in your AWS console before sending to customers. See [Set up Amazon SES](/help/set-up-amazon-ses).
- **The welcome screen keeps coming back.** A brand without a From Email sends you to it. Fill it in, or press **Skip for now**.
- **"Failed to save AWS credentials"**: check that both keys were pasted in full, with no spaces.
