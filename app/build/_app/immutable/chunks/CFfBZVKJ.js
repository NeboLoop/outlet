import{B as e,G as t,L as n,V as r,W as i,a,at as o,bt as s,it as c,lt as l,q as u,st as d,ut as f,yt as p,z as m}from"./BKXUg6nK.js";import"./xihTtKlq.js";import{ct as h,rt as g,st as _}from"./Do6Ocdwx.js";import{t as v}from"./D7ul0hOj.js";import{t as y}from"./kj0bUAFJ.js";var b=r(`<!> Copied!`,1),x=r(`<!> Copy`,1),S=r(`<div class="space-y-3"><div class="flex items-center justify-between"><div class="flex items-center gap-2"><!> <span class="text-sm font-medium text-text">Required IAM Policy</span></div> <button type="button" class="text-xs text-primary hover:underline flex items-center gap-1"><!></button></div> <p class="text-xs text-text-muted">Includes SES sending and domain permissions, SNS for bounce and complaint notifications, and S3 backup permissions. Replace <code class="bg-surface-tertiary px-1 rounded">YOUR-BACKUP-BUCKET</code> with your bucket name, or remove the S3 section if not using cloud backups.</p> <pre class="text-xs bg-surface-tertiary p-3 rounded-lg overflow-x-auto text-text-muted max-h-64 overflow-y-auto"></pre></div>`),C=r(`<!> Copy Policy`,1),w=r(`<div class="flex items-center justify-between mb-4"><div class="flex items-center gap-2"><!> <h3 class="font-semibold text-text">Required IAM Policy</h3></div> <button type="button" class="text-sm text-primary hover:underline flex items-center gap-1"><!></button></div> <p class="text-sm text-text-muted mb-4">Create an IAM user in AWS with this policy attached. The policy includes permissions for SES
			email sending and optional S3 backup storage.</p> <div class="bg-surface-tertiary rounded-lg p-4 overflow-x-auto max-h-80 overflow-y-auto font-mono text-sm"><pre class="text-text-muted"></pre></div> <div class="mt-4 flex items-center gap-4"><a href="https://console.aws.amazon.com/iam/" target="_blank" rel="noopener noreferrer" class="inline-flex items-center gap-1 text-sm text-primary hover:underline">Open IAM Console <!></a> <span class="text-xs text-text-muted">Replace <code class="bg-surface-secondary px-1 rounded">YOUR-BACKUP-BUCKET</code> with your
				bucket name</span></div>`,1);function T(r,i){let T=a(i,`compact`,3,!1),E=f(!1);async function D(){await navigator.clipboard.writeText(`{
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
        "ses:SetIdentityNotificationTopic",
        "ses:GetIdentityNotificationAttributes",
        "ses:SetIdentityMailFromDomain",
        "ses:GetIdentityMailFromDomainAttributes"
      ],
      "Resource": "*"
    },
    {
      "Sid": "OutletSNSBounceNotifications",
      "Effect": "Allow",
      "Action": [
        "sns:CreateTopic",
        "sns:Subscribe"
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
}`),l(E,!0),setTimeout(()=>l(E,!1),2e3)}var O=e(),k=o(O),A=e=>{var r=S(),i=c(r),a=c(i),l=c(a);v(l,{class:`w-4 h-4 text-warning`}),p(2),s(a);var f=d(a,2),h=c(f),y=e=>{var t=b(),n=o(t);_(n,{class:`w-3 h-3`}),p(),m(e,t)},C=e=>{var t=x(),n=o(t);g(n,{class:`w-3 h-3`}),p(),m(e,t)};n(h,e=>{u(E)?e(y):e(C,-1)}),s(f),s(i);var w=d(i,4);w.textContent=`{
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
        "ses:SetIdentityNotificationTopic",
        "ses:GetIdentityNotificationAttributes",
        "ses:SetIdentityMailFromDomain",
        "ses:GetIdentityMailFromDomainAttributes"
      ],
      "Resource": "*"
    },
    {
      "Sid": "OutletSNSBounceNotifications",
      "Effect": "Allow",
      "Action": [
        "sns:CreateTopic",
        "sns:Subscribe"
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
}`,s(r),t(`click`,f,D),m(e,r)},j=e=>{h(e,{children:(e,r)=>{var i=w(),a=o(i),l=c(a),f=c(l);v(f,{class:`w-5 h-5 text-warning`}),p(2),s(l);var h=d(l,2),x=c(h),S=e=>{var t=b(),n=o(t);_(n,{class:`w-4 h-4`}),p(),m(e,t)},T=e=>{var t=C(),n=o(t);g(n,{class:`w-4 h-4`}),p(),m(e,t)};n(x,e=>{u(E)?e(S):e(T,-1)}),s(h),s(a);var O=d(a,4),k=c(O);k.textContent=`{
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
        "ses:SetIdentityNotificationTopic",
        "ses:GetIdentityNotificationAttributes",
        "ses:SetIdentityMailFromDomain",
        "ses:GetIdentityMailFromDomainAttributes"
      ],
      "Resource": "*"
    },
    {
      "Sid": "OutletSNSBounceNotifications",
      "Effect": "Allow",
      "Action": [
        "sns:CreateTopic",
        "sns:Subscribe"
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
}`,s(O);var A=d(O,2),j=c(A),M=d(c(j));y(M,{class:`w-3 h-3`}),s(j),p(2),s(A),t(`click`,h,D),m(e,i)},$$slots:{default:!0}})};n(k,e=>{T()?e(A):e(j,-1)}),m(r,O)}i([`click`]);export{T as t};