# Backups and updates

Everything in Outlet lives in one database file on your server. A backup is a copy of it. This guide covers making backups, keeping them safe on Amazon S3, scheduling them, restoring one, and updating Outlet.

Only a super admin can manage backups and updates. Both are in Outlet's own **Settings**: on the **Brands** screen, click your initial at the top right and choose **Settings**.

## Make a backup now

1. Open the **Backup** tab.
2. Under **Create Backup**, tick **Also upload to S3** if S3 is set up (below).
3. Press **Create Backup Now**.

The backup appears under **Backup History** with its size and where it's stored: **Local** (on the server, in a `backups` folder next to Outlet) or **S3**. Press **Download** to keep a copy on your computer.

Each backup is a zip holding the database twice: as a database file (`.db`) and as a readable SQL file (`.sql`).

## Keep backups off the server

A backup on the same server is lost if the server is. Store them on Amazon S3 (Amazon's file storage) too.

**This needs you**, because only you can sign in to AWS and create the storage.

1. In AWS, create an S3 bucket, such as "acme-outlet-backups".
2. Give an AWS user the permissions shown under **Required IAM Policy** on this tab, with your bucket's name in place of `YOUR-BUCKET-NAME`. You can add them to the user you made for Amazon SES.
3. In Outlet, under **S3 Storage Settings**, switch on **Enable S3 backup storage** and fill in **S3 Bucket**, **S3 Region**, **S3 Key Prefix (optional)**, **Access Key ID** and **Secret Access Key**.
4. Press **Save Settings**.

## Schedule backups

1. Under **Scheduled Backups**, switch on **Enable scheduled backups**.
2. **Schedule (Cron)**: when to run, in cron format (a standard way to write times). The default `0 3 * * *` means 3 AM every day.
3. **Retention (days)**: backups older than this are deleted.
4. Press **Save Settings**.

## Restore a backup

There's no restore button in the app: restoring replaces everything, so it's done on the server.

**This needs the person who runs your Outlet server**, because it means stopping Outlet and running a command.

1. Stop Outlet.
2. Run `outlet restore` with the backup's database file, such as `outlet restore outlet-backup.db`. Outlet saves a copy of the current database first, then asks to confirm.
3. Start Outlet again.

Unzip the backup first; the `.db` file inside is what to restore.

## Update Outlet

1. Open the **Updates** tab. **Current Version** shows what you run.
2. Press **Check for Updates**. You see "You're up to date!" or **Update Available** with **What's New**.

Installing an update is done on the server, by the person who runs it:

- **Command line (recommended)**: run `outlet update`.
- **Docker**: run `docker pull ghcr.io/outlet-sh/outlet:latest` and restart the container.
- **By hand**: download the new version from GitHub Releases, stop Outlet, replace the program file, and start it again.

Make a backup before every update.

## Common problems

- **Backup shows Failed**: the server may be out of disk space. Delete old local backups, or set a shorter **Retention (days)**.
- **S3 upload fails**: check the bucket name and region, and that the AWS user has the four S3 permissions on that bucket.
