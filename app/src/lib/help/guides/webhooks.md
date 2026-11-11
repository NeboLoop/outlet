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

The event names are `contact.created`, `contact.unsubscribed`, `email.sent`, `email.delivered`, `email.bounced`, `email.complained` (Marked as Spam), `email.opened`, `email.clicked`, `email.replied` and `outreach.paused`.

`email.replied` fires when a person answers an outreach email. It carries the contact's `email`, the `mailbox` the reply reached, its `subject`, `message_id` and `received_at`, the `sequence_id` and the answered `email_id` when known, and its `classification` (for example `interested`, `out_of_office`, `unsubscribe` or `needs_review`). Bounces arrive as `email.bounced`, never as replies.

`outreach.paused` fires when outreach stops on its own, so you hear about it without checking. Its `scope` is `pace` when the day's new prospects (the pace dial) were reached: first emails wait for the next day's window (`resumes_at`) and follow-ups go on. It is `mailbox` when a mailbox paused for its health or its DNS (`mailbox`, `reason`). Nothing waiting is lost.

Any answer in the 200s counts as delivered. Outlet waits up to 30 seconds for it. When your system fails or doesn't answer, Outlet tries the event again, waiting longer each time: after 5 minutes, 1 hour, 6 hours and 18 hours, 5 tries over about a day. Every try carries the same body, signature and `X-Webhook-Delivery` ID, so your system can tell a repeat from a new event.

## Test and watch

Each webhook's row has:

- **Test Webhook**: sends a test event now and shows whether it worked.
- **Deliveries**: each event with its **Status** ("Delivered", "Retrying", "Sending" or "Failed"), **Tries** (such as "2 of 5"), **Last Response**, **Next Retry** and **Created**. **Send again** sends that event once more now, for example after you fixed your system and an event ran out of tries.
- **View Logs**: every try with its **Event**, **Status**, **Duration** and **Time**.
- **Disable** and **Enable**: pause it without deleting it.
- **Delete**: remove it.

The **Deliveries** column counts successes and failures, and how many events are waiting for another try. The **Retries** column shows them too.

## Common problems

- **Every delivery fails**: open **Deliveries** or **View Logs**. Check the address is public (not `localhost`), uses `https`, and answers within 30 seconds.
- **The signature doesn't match**: sign the raw body exactly as received, before your system parses it, with the secret from when you created the webhook.
- **Lost the secret**: delete the webhook and create it again to get a new one.
