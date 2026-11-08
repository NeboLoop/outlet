# Webhooks

A webhook lets Outlet tell another system the moment something happens, such as an email bouncing or someone unsubscribing. Outlet sends a short message to a web address you choose. Use it to keep your CRM, your app's database or an automation tool like Zapier or Make in step.

## Add a webhook

1. Open the brand's **Settings**, **Webhooks**.
2. Press **Add Webhook**.
3. **Endpoint URL**: the web address that receives the messages. Your developer or automation tool gives you this.
4. **Events to Subscribe**: tick what to send.
   - **Contacts**: **Contact Created**, **Contact Unsubscribed**.
   - **Email**: **Email Sent**, **Email Delivered**, **Email Bounced**, **Marked as Spam**, **Email Opened**, **Email Clicked**.
5. Keep **Active** ticked and press **Create Webhook**.
6. Copy the **Signing Secret** now. Outlet shows it once: "Save this secret now. You won't be able to see it again."

Only the brand's admin can manage webhooks.

## What Outlet sends

Each event is one `POST` to your address with a JSON body holding `event`, `timestamp` and `data`, and these headers:

- `X-Webhook-Signature`: a signature of the body, made with your signing secret (HMAC-SHA256, in hex). Your system recomputes it to be sure the message came from Outlet.
- `X-Webhook-Event`: the event name, such as `email.bounced`.
- `X-Webhook-ID`: the webhook's id.

The event names are `contact.created`, `contact.unsubscribed`, `email.sent`, `email.delivered`, `email.bounced`, `email.complained` (Marked as Spam), `email.opened` and `email.clicked`.

Any answer in the 200s counts as delivered. Outlet waits up to 30 seconds and sends each event once; it doesn't retry. If your system was down, the event is missed, so have it catch up from the API when it comes back.

## Test and watch

Each webhook's row has:

- **Test Webhook**: sends a test event now and shows whether it worked.
- **View Logs**: every delivery with its **Event**, **Status**, **Duration** and **Time**.
- **Disable** and **Enable**: pause it without deleting it.
- **Delete**: remove it.

The **Deliveries** column counts successes and failures.

## Common problems

- **Every delivery fails**: open **View Logs**. Check the address is public (not `localhost`), uses `https`, and answers within 30 seconds.
- **The signature doesn't match**: sign the raw body exactly as received, before your system parses it, with the secret from when you created the webhook.
- **Lost the secret**: delete the webhook and create it again to get a new one.
