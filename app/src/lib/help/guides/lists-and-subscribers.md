# Lists, subscribers and importing

A list is a group of people who asked to hear from you, such as "Newsletter" or "Product updates". This guide covers creating lists, adding and importing subscribers, double opt-in, custom fields and the subscribe form.

## Create a list

1. Open **Lists** and press **New List**.
2. **Name**: such as "Newsletter". The **Slug** fills itself in; your app and forms use it.
3. **Description (optional)**.
4. **Require double opt-in** is on by default. Keep it on: new subscribers get an email asking them to confirm, which keeps fake and mistyped addresses off your list.
5. Press **Create List**.

## Add one subscriber

1. Open the list, then the **Subscribers** tab.
2. Press **Add subscriber**, type the **Email** and **Name (optional)**, and add them.

They join right away, without a confirmation email, and the list's **On signup** autoresponders start for them. Add only people who asked to hear from you.

## Import a CSV file

A CSV file is a spreadsheet saved as plain text. Any spreadsheet app can save one ("Save as" or "Download as" CSV).

1. Make sure the first row has the column names, with one column called `email`. A `name` column is used too. Other columns are ignored.
2. Open the list's **Subscribers** tab and press **Import CSV**.
3. Pick the file (up to 50 MB).
4. Watch the progress bar: "Importing (file): x / y". When it's done you see "Imported X of Y rows (Z refused or invalid)".

**What the import does**

- Imported people join as confirmed straight away, without a confirmation email. Import only people who agreed to hear from you.
- Someone already on the list, including someone who unsubscribed, is left as they are. Unsubscribed people are never re-added.
- Addresses at blocked domains are refused.
- Before you pick the file, **Import CSV** asks about **Start sign-up emails for these subscribers**. It is on by default: everyone imported gets the list's **On signup** autoresponders from the first email. Turn it off when importing an old list whose people already had them. See [Sequences and autoresponders](/help/sequences-and-autoresponders).

Thinking of importing an old or bought list? Check it with **Validation** first. Bad addresses bounce, and bounces hurt your sending reputation. See [Clean lists, validation and the blocklist](/help/clean-lists).

## Subscriber statuses

- **Confirmed**: gets your mail.
- **Pending**: signed up but hasn't clicked the confirmation link yet (double opt-in).
- **Unsubscribed**: left the list and gets nothing from it.

Click a subscriber for their details: what they were sent, opened and clicked, their custom fields and sequences. **Remove from List** takes them off this list.

## Custom fields

Custom fields hold extra details, such as company or birthday, and can be used in emails.

1. Open the list's **Custom Fields** tab and press **Add Field**.
2. **Field Name**: the label shown on the subscribe form.
3. **Merge Tag Key**: the short name used in emails as `{{key}}`.
4. **Field Type**: **Text**, **Number**, **Date** or **Dropdown** (with **Options**, separated by commas).
5. Tick **Required field** if people must fill it in to subscribe.

## Get subscribers: the subscribe form

Open the list's **Subscribe Form** tab.

- **Hosted Subscribe Form**: a ready page at your Outlet address. Press **Copy URL** and link to it from your website or social profiles.
- **Embeddable HTML Form**: press **Copy** and paste the code into your website.
- **API Subscription**: your app can subscribe people with `POST /sdk/v1/lists/(list slug)/subscribe`, sending the brand's API key in the `X-API-Key` header and `email`, `name` and `custom_fields` in the body. (The example on that tab shows `/api/sdk/v1` and an `Authorization: Bearer` header; use the path and header given here.)

## List settings

The list's **Settings** tab holds:

- **Require double opt-in**, and the **Confirmation Email** it sends (it must include `{{confirm_url}}`).
- **Subscribe Settings**: where people land after subscribing, after confirming, or when already subscribed. Empty uses a simple page from Outlet.
- **Thank You Email**: an optional email after subscribing.
- **Unsubscribe Settings**: single or double opt-out, and whether unsubscribing leaves **This list only** or **All lists**.
- **Goodbye Email**: an optional email after unsubscribing.
- **Delete List**, in the **Danger Zone**: deletes the list, its subscribers and its autoresponders, for good.

## Common problems

- **"The CSV needs an email column (and may have a name column)."**: name the first row's email column `email`.
- **"The file isn't a CSV with a header row."**: save the spreadsheet as CSV, with column names in the first row.
- **"This address's domain is on the brand's blocklist."**: see **Blocklist**.
- **Subscribers stay Pending**: they haven't confirmed. Check the confirmation email arrives (try it yourself), and that **Marketing** is carried by a pathway in **Settings**, **Sending**.
- **Exporting a list**: on the list's **Subscribers** tab, press **Export CSV**. Your browser downloads every subscriber of the list as a CSV file.
