# Sequences and autoresponders

An autoresponder is a series of emails sent automatically, one after another, to each person who joins a list. A welcome series is the classic example: a welcome email right away, a tip the next day, an offer a few days later. Outlet calls these series sequences; the ones tied to a list are autoresponders.

Autoresponders are **marketing** mail and go out through the pathway that carries **Marketing**. Outreach follow-ups are a different thing, covered in [Set up outreach end to end](/help/set-up-outreach).

## Create an autoresponder

1. Open **Lists** and click the list.
2. Open the **Autoresponders** tab and press **New Autoresponder**.
3. In **Create Autoresponder**, give it a **Name** and a **Slug** (the short name your app uses to enroll people), pick the **Trigger**, and press **Create**.

**Triggers**

- **On signup**: starts when someone joins the list through your subscribe form or your app's API. With double opt-in, it starts when they confirm.
- **Manual enrollment**: starts only when you enroll someone yourself, or your app does.

Each list runs one **On signup** autoresponder. Put the whole welcome series in one, not several.

## Add the emails

1. On the autoresponder's page, under **Emails**, press **Add Email**.
2. Write the **Subject Line** and the email.
3. Set **Send Timing**: the first email counts **hours after signup** (0 means immediately); later emails count **hours after previous email** (24 by default).
4. Add as many emails as you need.

Each wait counts from when the previous email was actually sent.

## Settings

- **List**: move the autoresponder to another list.
- **On Completion**: when someone finishes this autoresponder, enroll them in another sequence. "None (do nothing)" is the default.
- **Save Changes** saves; **Delete** removes the autoresponder and all of its emails.

On the **Autoresponders** tab, each autoresponder's menu has **Pause** and **Activate**. A new autoresponder starts active straight away, so write its emails before people join the list.

## Enroll someone by hand

1. Open the list's **Subscribers** tab and click the person.
2. Press **Enroll in Sequence** and pick the sequence.

Someone already in a sequence is never enrolled in it twice.

**From your app**: `POST /sdk/v1/sequences/enroll` with your brand's API key in the `X-API-Key` header, and `sequence_slug` and `email` in the body. See [Transactional email from your app](/help/transactional-email) for API keys.

## Who gets an On signup autoresponder

- **Yes**: people who join through the hosted subscribe form, the embedded form, or your app's subscribe API, once they're active (after confirming, with double opt-in).
- **No**: people you add with **Add subscriber** or **Import CSV**. Enroll them by hand if they should get it.

## Common problems

- **Nobody gets the welcome email**: check the trigger is **On signup**, the autoresponder is **Active**, and **Marketing** is carried by a pathway in **Settings**, **Sending**. Check the person joined through a form or the API, not an import.
- **People wait for confirmation**: with double opt-in on, the autoresponder starts only once they click the confirmation link.
