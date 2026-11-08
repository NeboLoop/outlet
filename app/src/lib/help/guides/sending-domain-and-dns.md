# Your sending domain and DNS explained

Before mail services trust your email, your domain has to vouch for it. It does that through DNS records: small public notes attached to your domain. This guide explains each record in plain words, the difference between nameservers and DNS records, and what Outlet writes for you.

## The words

- **Domain**: your name on the internet, such as `acme.com`.
- **Registrar**: the company you bought the domain from, such as Namecheap, GoDaddy or Cloudflare.
- **DNS host**: the company that answers for your domain's records. Often it's the registrar, but it doesn't have to be.
- **DNS record**: one public note about your domain, such as "mail for acme.com goes to Google".

## Nameservers vs DNS records

These are often confused.

- **Nameservers** say **which company** holds your domain's DNS records. They are set at your **registrar**. A domain has one set of nameservers.
- **DNS records** are the notes themselves. They are added at your **DNS host**, the company the nameservers point to.

Changing nameservers moves all of your DNS to another company. It is how a mailbox service or DigitalOcean takes over a domain's records. Do this only on a domain used just for outreach, or after copying every existing record, or your website and other mail may stop working.

## The records for email

**SPF** (Sender Policy Framework): a TXT record listing the servers allowed to send mail for your domain. A domain must have only **one** SPF record. To allow two services, combine them in one record, such as `v=spf1 include:_spf.google.com include:amazonses.com ~all`.

**DKIM** (DomainKeys Identified Mail): a signature on every email, checked against a public key in your DNS. It proves the email really came from you and wasn't changed on the way. Amazon SES uses three CNAME records ending in `._domainkey`. Google uses one TXT record at `google._domainkey`.

**DMARC**: a TXT record at `_dmarc.yourdomain` that tells receivers what to do with mail that fails SPF and DKIM, and where to send reports. Outlet suggests starting with `v=DMARC1; p=none`, which only watches. Google, Yahoo and Microsoft expect every sender to have one.

**MX** (Mail Exchanger): says which server receives mail for a domain. Outreach mailboxes need MX records so replies arrive. Amazon SES uses an MX record on your MAIL FROM subdomain to receive bounces.

**Verification TXT**: a one-off record that proves to a service you own the domain, such as `_amazonses.yourdomain` for Amazon SES.

**Tracking CNAME**: for outreach with click tracking on, a `go.` subdomain pointing at your Outlet server, so tracked links use your own domain.

## What Outlet does for you

| Domain | What is automatic | What you do |
| --- | --- | --- |
| Brand sending domain on Amazon SES | Outlet registers it with Amazon, creates the DKIM keys and MAIL FROM, lists every record, and checks every minute | Add the records at your DNS host |
| Outreach domain on DigitalOcean DNS | Outlet writes MX, SPF and DMARC itself, and DKIM too when you paste its value from your mail provider; only the missing ones | Save a DigitalOcean token once |
| Outreach domain at a mailbox service | The service writes every record | Set the service's nameservers at your registrar, once |
| Outreach domain anywhere else | Outlet lists every record and checks them daily | Add the records at your DNS host |

**DigitalOcean automation, step by step**

1. Your domain's nameservers must point to DigitalOcean, and the domain must be added in DigitalOcean's **Networking**, **Domains**. If not, Outlet says "DigitalOcean has no zone for this domain. Point the domain's nameservers at DigitalOcean and add it there first."
2. In DigitalOcean, create a personal access token with read and write access to domains.
3. In Outlet, open **Outbound**, **Setup**, and under **DigitalOcean DNS token** paste it and press **Save token**.
4. Add the outreach domain with **DNS host** set to "DigitalOcean".

Outlet never adds a second SPF, DKIM or DMARC record, and never changes your other records. If DigitalOcean refuses some records, Outlet shows them to add by hand.

## How to add a record by hand

1. Sign in at your DNS host and open the DNS settings for the domain.
2. Add a record with the **Type**, **Name** (also called Host) and **Value** Outlet shows. For MX records, also the **Priority**.
3. Many DNS hosts add your domain to the name by themselves. If Outlet shows `_dmarc.acme.com`, type just `_dmarc`.
4. Save, then press **Refresh** (brand email) or **Check** (outreach domain) in Outlet.

Changes usually show up within minutes to an hour, and can take up to 72 hours.

## Common problems

- **Two SPF records**: receivers then ignore both. Merge them into one.
- **The domain appears twice in a name**, such as `_dmarc.acme.com.acme.com`: type only the part before your domain.
- **The record is there, but Outlet says it's missing**: wait an hour; DNS takes time to spread. Check you added it to the right domain.
- **Your website stopped working after changing nameservers**: the new DNS host doesn't have your website's records. Copy them over from the old host.

See also [Set up Amazon SES](/help/set-up-amazon-ses) and [Set up outreach end to end](/help/set-up-outreach).
