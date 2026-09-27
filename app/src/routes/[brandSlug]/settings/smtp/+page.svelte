<script lang="ts">
	import { Card, Badge, Button, Alert, Tabs, CodeBlock } from '$lib/components/ui';
	import { Copy, CheckCircle, Mail, Server } from '@lucide/svelte';
	import { browser } from '$app/environment';
	import { page } from '$app/stores';

	let installDomain = $state(browser ? window.location.hostname : 'your-domain.com');
	let copiedText = $state('');

	// Get brandSlug from URL
	let brandSlug = $derived($page.params.brandSlug!);

	function copyToClipboard(text: string) {
		navigator.clipboard.writeText(text);
		copiedText = text;
		setTimeout(() => (copiedText = ''), 2000);
	}

	const smtpHost = $derived(installDomain);
	const smtpPort = 587;

	// Code example tabs
	const codeTabs = [
		{ id: 'curl', label: 'cURL' },
		{ id: 'nodejs', label: 'Node.js' },
		{ id: 'python', label: 'Python' },
		{ id: 'go', label: 'Go' },
		{ id: 'php', label: 'PHP' }
	];
	let activeCodeTab = $state('curl');

	// Code examples - using brandSlug for username
	const curlCode = $derived(`# Send email via SMTP using curl
curl --url "smtp://${smtpHost}:${smtpPort}" \\
  --ssl-reqd \\
  --user "${brandSlug}:YOUR_API_KEY" \\
  --mail-from "you@example.com" \\
  --mail-rcpt "recipient@example.com" \\
  --upload-file - << EOF
From: you@example.com
To: recipient@example.com
Subject: Hello from Outlet
Content-Type: text/html
X-Outlet-Type: transactional
X-Outlet-Track: opens,clicks
X-Outlet-Meta-User-ID: 12345

<h1>Welcome!</h1>
<p>This is a test email sent via SMTP.</p>
EOF`);

	const nodejsCode = $derived(`import nodemailer from 'nodemailer';

const transporter = nodemailer.createTransport({
  host: '${smtpHost}',
  port: ${smtpPort},
  secure: false, // STARTTLS
  auth: {
    user: '${brandSlug}',
    pass: 'YOUR_API_KEY'
  }
});

await transporter.sendMail({
  from: 'you@example.com',
  to: 'recipient@example.com',
  subject: 'Hello from Outlet',
  html: '<h1>Welcome!</h1>',
  headers: {
    'X-Outlet-Type': 'transactional',
    'X-Outlet-Track': 'opens,clicks',
    'X-Outlet-Meta-User-ID': '12345'
  }
});`);

	const pythonCode = $derived(`import smtplib
from email.mime.text import MIMEText
from email.mime.multipart import MIMEMultipart

msg = MIMEMultipart('alternative')
msg['From'] = 'you@example.com'
msg['To'] = 'recipient@example.com'
msg['Subject'] = 'Hello from Outlet'
msg['X-Outlet-Type'] = 'transactional'
msg['X-Outlet-Track'] = 'opens,clicks'

msg.attach(MIMEText('<h1>Welcome!</h1>', 'html'))

with smtplib.SMTP('${smtpHost}', ${smtpPort}) as server:
    server.starttls()
    server.login('${brandSlug}', 'YOUR_API_KEY')
    server.sendmail(msg['From'], msg['To'], msg.as_string())`);

	const goCode = $derived(`package main

import (
    "net/smtp"
)

func main() {
    auth := smtp.PlainAuth("", "${brandSlug}", "YOUR_API_KEY", "${smtpHost}")

    msg := []byte("From: you@example.com\\r\\n" +
        "To: recipient@example.com\\r\\n" +
        "Subject: Hello from Outlet\\r\\n" +
        "X-Outlet-Type: transactional\\r\\n" +
        "Content-Type: text/html\\r\\n" +
        "\\r\\n" +
        "<h1>Welcome!</h1>")

    err := smtp.SendMail("${smtpHost}:${smtpPort}", auth,
        "you@example.com", []string{"recipient@example.com"}, msg)
}`);

	const phpCode = $derived(`use PHPMailer\\PHPMailer\\PHPMailer;

$mail = new PHPMailer(true);

$mail->isSMTP();
$mail->Host = '${smtpHost}';
$mail->Port = ${smtpPort};
$mail->SMTPAuth = true;
$mail->Username = '${brandSlug}';
$mail->Password = 'YOUR_API_KEY';
$mail->SMTPSecure = PHPMailer::ENCRYPTION_STARTTLS;

$mail->setFrom('you@example.com');
$mail->addAddress('recipient@example.com');
$mail->Subject = 'Hello from Outlet';
$mail->isHTML(true);
$mail->Body = '<h1>Welcome!</h1>';

$mail->addCustomHeader('X-Outlet-Type', 'transactional');
$mail->addCustomHeader('X-Outlet-Track', 'opens,clicks');

$mail->send();`);

	let codeExamples = $derived<Record<string, { code: string; language: string }>>({
		curl: { code: curlCode, language: 'bash' },
		nodejs: { code: nodejsCode, language: 'javascript' },
		python: { code: pythonCode, language: 'python' },
		go: { code: goCode, language: 'go' },
		php: { code: phpCode, language: 'php' }
	});

	let activeExample = $derived(codeExamples[activeCodeTab]);
</script>

<svelte:head>
	<title>SMTP - Settings</title>
</svelte:head>

<div class="space-y-6">
	<Alert type="info" title="SMTP Ingress">
		<p>
			Connect your applications to Outlet via SMTP to send transactional or marketing emails. This
			allows you to use Outlet as your SMTP relay server.
		</p>
		<p class="mt-2 text-sm">
			<strong>Brand isolation:</strong> Your SMTP username must match your brand slug ({brandSlug}) to ensure emails are sent from the correct brand.
		</p>
	</Alert>

	<!-- Connection Settings -->
	<Card>
		<div class="flex items-center gap-3 mb-4">
			<Server class="h-5 w-5 text-primary" />
			<h2 class="text-lg font-medium text-text">Connection Settings</h2>
		</div>

		<div class="space-y-4">
			<div class="grid grid-cols-2 gap-4">
				<div>
					<label class="form-label">SMTP Host</label>
					<div class="flex gap-2">
						<code
							class="flex-1 bg-surface-tertiary px-3 py-2 rounded text-sm font-mono text-text"
						>
							{smtpHost}
						</code>
						<Button type="secondary" size="sm" onclick={() => copyToClipboard(smtpHost)}>
							{#if copiedText === smtpHost}
								<CheckCircle class="h-4 w-4 text-green-500" />
							{:else}
								<Copy class="h-4 w-4" />
							{/if}
						</Button>
					</div>
				</div>
				<div>
					<label class="form-label">SMTP Port</label>
					<code class="block bg-surface-tertiary px-3 py-2 rounded text-sm font-mono text-text">
						{smtpPort}
					</code>
				</div>
			</div>

			<div class="grid grid-cols-2 gap-4">
				<div>
					<label class="form-label">Username</label>
					<div class="flex gap-2">
						<code
							class="flex-1 bg-surface-tertiary px-3 py-2 rounded text-sm font-mono text-text"
						>
							{brandSlug}
						</code>
						<Button type="secondary" size="sm" onclick={() => copyToClipboard(brandSlug)}>
							{#if copiedText === brandSlug}
								<CheckCircle class="h-4 w-4 text-green-500" />
							{:else}
								<Copy class="h-4 w-4" />
							{/if}
						</Button>
					</div>
					<p class="mt-1 text-xs text-text-muted">Your brand slug (required for authentication)</p>
				</div>
				<div>
					<label class="form-label">Password</label>
					<code class="block bg-surface-tertiary px-3 py-2 rounded text-sm font-mono text-text">
						YOUR_API_KEY
					</code>
					<p class="mt-1 text-xs text-text-muted">
						<a href="/{brandSlug}/settings" class="text-primary hover:underline">Get an API key</a> from Brand Settings
					</p>
				</div>
			</div>

			<div class="pt-4 border-t border-border">
				<h3 class="font-medium text-text mb-3">Encryption</h3>
				<div class="flex items-center gap-2">
					<Badge type="success">STARTTLS</Badge>
					<span class="text-sm text-text-muted"
						>Encryption upgrades automatically when available</span
					>
				</div>
			</div>
		</div>
	</Card>

	<!-- Custom Headers -->
	<Card>
		<div class="flex items-center gap-3 mb-4">
			<Mail class="h-5 w-5 text-primary" />
			<h2 class="text-lg font-medium text-text">Custom Headers</h2>
		</div>
		<p class="text-sm text-text-muted mb-4">
			Use these custom headers to control how Outlet processes your emails.
		</p>

		<div class="overflow-x-auto">
			<table class="w-full text-sm">
				<thead>
					<tr class="border-b border-border">
						<th class="text-left py-2 font-medium text-text-muted">Header</th>
						<th class="text-left py-2 font-medium text-text-muted">Values</th>
						<th class="text-left py-2 font-medium text-text-muted">Description</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border/50">
					<tr>
						<td class="py-3 font-mono text-primary whitespace-nowrap">X-Outlet-Type</td>
						<td class="py-3">
							<Badge type="secondary">transactional</Badge>
							<Badge type="secondary">marketing</Badge>
						</td>
						<td class="py-3 text-text-muted">
							Email type. Default: <code class="text-xs bg-surface-tertiary px-1 rounded"
								>transactional</code
							>
						</td>
					</tr>
					<tr>
						<td class="py-3 font-mono text-primary whitespace-nowrap">X-Outlet-List</td>
						<td class="py-3"><code class="text-xs bg-surface-tertiary px-1 rounded">slug</code></td>
						<td class="py-3 text-text-muted">
							Associate email with a list (for marketing). List slug is scoped to your brand.
						</td>
					</tr>
					<tr>
						<td class="py-3 font-mono text-primary whitespace-nowrap">X-Outlet-Tags</td>
						<td class="py-3"
							><code class="text-xs bg-surface-tertiary px-1 rounded">tag1,tag2,tag3</code></td
						>
						<td class="py-3 text-text-muted">Comma-separated tags to apply to recipient</td>
					</tr>
					<tr>
						<td class="py-3 font-mono text-primary whitespace-nowrap">X-Outlet-Template</td>
						<td class="py-3"><code class="text-xs bg-surface-tertiary px-1 rounded">slug</code></td>
						<td class="py-3 text-text-muted">
							Use a predefined email template. Template slug is scoped to your brand.
						</td>
					</tr>
					<tr>
						<td class="py-3 font-mono text-primary whitespace-nowrap">X-Outlet-Track</td>
						<td class="py-3">
							<code class="text-xs bg-surface-tertiary px-1 rounded">opens,clicks</code>
							<code class="text-xs bg-surface-tertiary px-1 rounded">none</code>
						</td>
						<td class="py-3 text-text-muted">
							Enable/disable tracking. Default: <code
								class="text-xs bg-surface-tertiary px-1 rounded">opens,clicks</code
							>
						</td>
					</tr>
					<tr>
						<td class="py-3 font-mono text-primary whitespace-nowrap">X-Outlet-Meta-*</td>
						<td class="py-3"
							><code class="text-xs bg-surface-tertiary px-1 rounded">any value</code></td
						>
						<td class="py-3 text-text-muted"
							>Custom metadata. E.g., <code class="text-xs bg-surface-tertiary px-1 rounded"
								>X-Outlet-Meta-Order-ID: 12345</code
							></td
						>
					</tr>
				</tbody>
			</table>
		</div>
	</Card>

	<!-- Code Examples -->
	<Card>
		<h2 class="text-lg font-medium text-text mb-4">Code Examples</h2>

		<div class="mb-4">
			<Tabs tabs={codeTabs} bind:activeTab={activeCodeTab} variant="pills" />
		</div>

		<CodeBlock code={activeExample.code} language={activeExample.language} />
	</Card>

	<!-- Limits -->
	<Card>
		<h2 class="text-lg font-medium text-text mb-2">SMTP Limits</h2>
		<p class="text-sm text-text-muted mb-4">
			These are the default limits for emails sent via the SMTP ingress server. Emails exceeding
			these limits will be rejected.
		</p>
		<div class="grid grid-cols-2 gap-4 text-sm">
			<div class="flex justify-between p-3 bg-surface-secondary rounded">
				<span class="text-text-muted">Max message size</span>
				<span class="text-text font-medium">25 MB</span>
			</div>
			<div class="flex justify-between p-3 bg-surface-secondary rounded">
				<span class="text-text-muted">Max recipients per message</span>
				<span class="text-text font-medium">100</span>
			</div>
		</div>
	</Card>
</div>
