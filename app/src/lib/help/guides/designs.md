# Designs

A design is a reusable email layout, such as your logo at the top and your address and social links at the bottom. Your campaign's content is placed inside it, so every newsletter looks the same without copying the layout each time.

In the menu it's called **Designs**; the page itself is titled **Templates**.

## Create a design

1. Open **Designs** and press **New Template**.
2. In **Create New Template**, fill in **Name**, **Slug** (a short name, which can't be changed later), **Category** ("General", "Newsletter", "Promotional" or "Transactional") and **Description (optional)**.
3. Press **Create Template**.
4. In the editor, build the layout. Put `{{content}}` where each email's own content goes. It is listed as "Main content area" among the merge tags.
5. Keep **Active** on. Press **Preview** to see it, then **Save & Close**.

If a design has no `{{content}}`, Outlet places the content at the end of the email instead.

## Use a design

In the campaign editor, under **Template**, pick the design. "None" sends your content as it is. Only **Active** designs are offered. See [Campaigns and newsletters](/help/campaigns-and-newsletters).

Transactional emails have their own templates on the **Transactional** page. See [Transactional email from your app](/help/transactional-email).

## Tips

- Keep designs simple: one column, a logo, your content, and a footer with your address and the unsubscribe link (`{{unsubscribe_url}}`).
- Send a test campaign to yourself on a phone and a computer before using a new design.
- To make a variation, create a new design and paste in the old one's code. The **Duplicate** menu item doesn't work yet.

## Common problems

- **A design doesn't appear in the campaign editor**: switch it to **Active**.
- **Content shows at the bottom instead of the middle**: add `{{content}}` to the design where the content belongs.
