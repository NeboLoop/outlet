# Users and roles

Outlet is invite-only. Each person signs in with their own account, and what they can do depends on their role. This guide explains the roles and how to add people.

## The roles

**Platform roles**, for the whole Outlet:

- **Super admin**: runs every brand and the platform itself. Only a super admin manages users, platform settings (such as Amazon SES and SMTP relay), backups, updates and new brands. The account made in the setup wizard is a super admin.
- **User**: reaches only the brands they're invited to, with the role they have there.

**Brand roles**, for one brand:

- **Member**: runs the brand's everyday mail: contacts, lists, sequences, designs, campaigns and sends.
- **Admin**: everything a member can do, plus the brand itself: its settings, sending setup, keys, webhooks, people, turning outreach on, and anything that deletes or clears its data.

## Invite someone to a brand

Brand admins do this.

1. Open the brand's **Settings**, **People**.
2. Under **Invite someone**, type the **Email address** and pick the **Role**: **Member** or **Admin**.
3. Press **Invite**.

Outlet emails an invitation. It works once and expires; the page shows the date. If you see "Invitation email did not go out", the brand's sending isn't working yet: check **Settings**, **Sending**, then invite again.

## See who's in a brand

The same **People** page lists the brand's **Members**, each with their role, and its **Pending invitations**: the ones nobody has accepted yet, with the date each expires. An expired one is marked **Expired**. **Resend** emails a fresh link to that person, with the same role, and the old link stops working.

## Add a user to Outlet

Super admins do this.

1. Open Outlet's own **Settings**: on the **Brands** screen, click your initial at the top right and choose **Settings**.
2. Open the **Users** tab and press **Add User**.
3. Fill in **Name**, **Email**, **Password** and **Role** ("User" or "Super admin"), and press **Add User**.

**Edit** changes a user, including switching them **Active** or **Inactive**. **Delete** removes them for good.

## Common problems

- **"You don't have access to this brand."**: ask the brand's admin to invite you.
- **"This needs the brand's admin role."**: ask an admin, or to be made an admin.
- **"Only a super admin can do this."**: platform settings, users, backups and new brands need a super admin.
- **"This account is not active."**: a super admin switched the account off. Ask them to switch it back on.
