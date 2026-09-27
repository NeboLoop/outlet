import{B as e,I as t,K as n,R as r,U as i,W as a,a as o,ct as s,it as c,lt as l,ot as u,rt as d,vt as f,yt as p,z as m}from"./C_OAeTtO.js";import"./xihTtKlq.js";import{B as h,H as g,I as _}from"./xZIB709U.js";import{t as v}from"./5NkVJIBI.js";import{t as y}from"./MsGSxNwW.js";var b=e(`<!> Copied!`,1),x=e(`<!> Copy`,1),S=e(`<div class="space-y-3"><div class="flex items-center justify-between"><div class="flex items-center gap-2"><!> <span class="text-sm font-medium text-text">Required IAM Policy</span></div> <button type="button" class="text-xs text-primary hover:underline flex items-center gap-1"><!></button></div> <p class="text-xs text-text-muted">Includes SES email permissions and S3 backup permissions. Replace <code class="bg-surface-tertiary px-1 rounded">YOUR-BACKUP-BUCKET</code> with your bucket name, or remove the S3 section if not using cloud backups.</p> <pre class="text-xs bg-surface-tertiary p-3 rounded-lg overflow-x-auto text-text-muted max-h-64 overflow-y-auto"></pre></div>`),C=e(`<!> Copy Policy`,1),w=e(`<div class="flex items-center justify-between mb-4"><div class="flex items-center gap-2"><!> <h3 class="font-semibold text-text">Required IAM Policy</h3></div> <button type="button" class="text-sm text-primary hover:underline flex items-center gap-1"><!></button></div> <p class="text-sm text-text-muted mb-4">Create an IAM user in AWS with this policy attached. The policy includes permissions for SES
			email sending and optional S3 backup storage.</p> <div class="bg-surface-tertiary rounded-lg p-4 overflow-x-auto max-h-80 overflow-y-auto font-mono text-sm"><pre class="text-text-muted"></pre></div> <div class="mt-4 flex items-center gap-4"><a href="https://console.aws.amazon.com/iam/" target="_blank" rel="noopener noreferrer" class="inline-flex items-center gap-1 text-sm text-primary hover:underline">Open IAM Console <!></a> <span class="text-xs text-text-muted">Replace <code class="bg-surface-secondary px-1 rounded">YOUR-BACKUP-BUCKET</code> with your
				bucket name</span></div>`,1);function T(e,i){let T=o(i,`compact`,3,!1),E=l(!1);async function D(){await navigator.clipboard.writeText(`{
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
}`),s(E,!0),setTimeout(()=>s(E,!1),2e3)}var O=m(),k=c(O),A=e=>{var i=S(),o=d(i),s=d(o),l=d(s);v(l,{class:`w-4 h-4 text-amber-500`}),f(2),p(s);var m=u(s,2),g=d(m),y=e=>{var t=b(),n=c(t);h(n,{class:`w-3 h-3`}),f(),r(e,t)},C=e=>{var t=x(),n=c(t);_(n,{class:`w-3 h-3`}),f(),r(e,t)};t(g,e=>{n(E)?e(y):e(C,-1)}),p(m),p(o);var w=u(o,4);w.textContent=`{
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
}`,p(i),a(`click`,m,D),r(e,i)},j=e=>{g(e,{children:(e,i)=>{var o=w(),s=c(o),l=d(s),m=d(l);v(m,{class:`w-5 h-5 text-amber-500`}),f(2),p(l);var g=u(l,2),x=d(g),S=e=>{var t=b(),n=c(t);h(n,{class:`w-4 h-4`}),f(),r(e,t)},T=e=>{var t=C(),n=c(t);_(n,{class:`w-4 h-4`}),f(),r(e,t)};t(x,e=>{n(E)?e(S):e(T,-1)}),p(g),p(s);var O=u(s,4),k=d(O);k.textContent=`{
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
}`,p(O);var A=u(O,2),j=d(A),M=u(d(j));y(M,{class:`w-3 h-3`}),p(j),f(2),p(A),a(`click`,g,D),r(e,o)},$$slots:{default:!0}})};t(k,e=>{T()?e(A):e(j,-1)}),r(e,O)}i([`click`]);export{T as t};