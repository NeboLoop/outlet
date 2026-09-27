<script lang="ts">
	import { Card, CodeBlock } from '$lib/components/ui';
	import { Copy, Check, ExternalLink, HelpCircle } from '@lucide/svelte';

	let { compact = false }: { compact?: boolean } = $props();

	let copiedPolicy = $state(false);

	const iamPolicy = `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "OutletSESPermissions",
      "Effect": "Allow",
      "Action": [
        "ses:SendEmail",
        "ses:SendRawEmail",
        "ses:GetSendQuota",
        "ses:GetSendStatistics",
        "ses:VerifyDomainIdentity",
        "ses:VerifyDomainDkim",
        "ses:GetIdentityVerificationAttributes",
        "ses:GetIdentityDkimAttributes",
        "ses:ListIdentities",
        "ses:DeleteIdentity",
        "ses:SetIdentityFeedbackForwardingEnabled",
        "ses:SetIdentityNotificationTopic"
      ],
      "Resource": "*"
    },
    {
      "Sid": "OutletS3BackupPermissions",
      "Effect": "Allow",
      "Action": [
        "s3:PutObject",
        "s3:GetObject",
        "s3:DeleteObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::YOUR-BACKUP-BUCKET",
        "arn:aws:s3:::YOUR-BACKUP-BUCKET/*"
      ]
    }
  ]
}`;

	async function copyPolicy() {
		await navigator.clipboard.writeText(iamPolicy);
		copiedPolicy = true;
		setTimeout(() => (copiedPolicy = false), 2000);
	}
</script>

{#if compact}
	<div class="space-y-3">
		<div class="flex items-center justify-between">
			<div class="flex items-center gap-2">
				<HelpCircle class="w-4 h-4 text-amber-500" />
				<span class="text-sm font-medium text-text">Required IAM Policy</span>
			</div>
			<button
				type="button"
				onclick={copyPolicy}
				class="text-xs text-primary hover:underline flex items-center gap-1"
			>
				{#if copiedPolicy}
					<Check class="w-3 h-3" />
					Copied!
				{:else}
					<Copy class="w-3 h-3" />
					Copy
				{/if}
			</button>
		</div>
		<p class="text-xs text-text-muted">
			Includes SES email permissions and S3 backup permissions. Replace <code
				class="bg-surface-tertiary px-1 rounded">YOUR-BACKUP-BUCKET</code
			> with your bucket name, or remove the S3 section if not using cloud backups.
		</p>
		<pre
			class="text-xs bg-surface-tertiary p-3 rounded-lg overflow-x-auto text-text-muted max-h-64 overflow-y-auto">{iamPolicy}</pre>
	</div>
{:else}
	<Card>
		<div class="flex items-center justify-between mb-4">
			<div class="flex items-center gap-2">
				<HelpCircle class="w-5 h-5 text-amber-500" />
				<h3 class="font-semibold text-text">Required IAM Policy</h3>
			</div>
			<button
				type="button"
				onclick={copyPolicy}
				class="text-sm text-primary hover:underline flex items-center gap-1"
			>
				{#if copiedPolicy}
					<Check class="w-4 h-4" />
					Copied!
				{:else}
					<Copy class="w-4 h-4" />
					Copy Policy
				{/if}
			</button>
		</div>
		<p class="text-sm text-text-muted mb-4">
			Create an IAM user in AWS with this policy attached. The policy includes permissions for SES
			email sending and optional S3 backup storage.
		</p>
		<div
			class="bg-surface-tertiary rounded-lg p-4 overflow-x-auto max-h-80 overflow-y-auto font-mono text-sm"
		>
			<pre class="text-text-muted">{iamPolicy}</pre>
		</div>
		<div class="mt-4 flex items-center gap-4">
			<a
				href="https://console.aws.amazon.com/iam/"
				target="_blank"
				rel="noopener noreferrer"
				class="inline-flex items-center gap-1 text-sm text-primary hover:underline"
			>
				Open IAM Console
				<ExternalLink class="w-3 h-3" />
			</a>
			<span class="text-xs text-text-muted">
				Replace <code class="bg-surface-secondary px-1 rounded">YOUR-BACKUP-BUCKET</code> with your
				bucket name
			</span>
		</div>
	</Card>
{/if}
