import"./DsnmJJEf.js";import{f as d,c as e,n as a,r as t,h as m,i as P,b as O,s as L}from"./BtjfW9_B.js";import{d as M}from"./DEsepwd4.js";import{c as q,a as r,f as u}from"./DnSY09wi.js";import{p as N,i as E}from"./Cn_LYL0x.js";import{C as Q,j as U,i as B}from"./DTNgJKyn.js";import{C as R}from"./D9rf7MwP.js";import{E as z}from"./YIvrxY7h.js";var W=u("<!> Copied!",1),H=u("<!> Copy",1),J=u('<div class="space-y-3"><div class="flex items-center justify-between"><div class="flex items-center gap-2"><!> <span class="text-sm font-medium text-text">Required IAM Policy</span></div> <button type="button" class="text-xs text-primary hover:underline flex items-center gap-1"><!></button></div> <p class="text-xs text-text-muted">Includes SES email permissions and S3 backup permissions. Replace <code class="bg-surface-tertiary px-1 rounded">YOUR-BACKUP-BUCKET</code> with your bucket name, or remove the S3 section if not using cloud backups.</p> <pre class="text-xs bg-surface-tertiary p-3 rounded-lg overflow-x-auto text-text-muted max-h-64 overflow-y-auto"></pre></div>'),X=u("<!> Copied!",1),Z=u("<!> Copy Policy",1),$=u(`<div class="flex items-center justify-between mb-4"><div class="flex items-center gap-2"><!> <h3 class="font-semibold text-text">Required IAM Policy</h3></div> <button type="button" class="text-sm text-primary hover:underline flex items-center gap-1"><!></button></div> <p class="text-sm text-text-muted mb-4">Create an IAM user in AWS with this policy attached. The policy includes permissions for SES
			email sending and optional S3 backup storage.</p> <div class="bg-surface-tertiary rounded-lg p-4 overflow-x-auto max-h-80 overflow-y-auto font-mono text-sm"><pre class="text-text-muted"></pre></div> <div class="mt-4 flex items-center gap-4"><a href="https://console.aws.amazon.com/iam/" target="_blank" rel="noopener noreferrer" class="inline-flex items-center gap-1 text-sm text-primary hover:underline">Open IAM Console <!></a> <span class="text-xs text-text-muted">Replace <code class="bg-surface-secondary px-1 rounded">YOUR-BACKUP-BUCKET</code> with your
				bucket name</span></div>`,1);function ce(D,K){let G=N(K,"compact",3,!1),b=L(!1);const T=`{
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
}`;async function g(){await navigator.clipboard.writeText(T),O(b,!0),setTimeout(()=>O(b,!1),2e3)}var h=q(),j=d(h);{var V=o=>{var f=J(),S=e(f),c=e(S),p=e(c);R(p,{class:"w-4 h-4 text-amber-500"}),a(2),t(c);var i=m(c,2);i.__click=g;var _=e(i);{var v=n=>{var s=W(),y=d(s);U(y,{class:"w-3 h-3"}),a(),r(n,s)},w=n=>{var s=H(),y=d(s);B(y,{class:"w-3 h-3"}),a(),r(n,s)};E(_,n=>{P(b)?n(v):n(w,!1)})}t(i),t(S);var A=m(S,4);A.textContent=`{
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
}`,t(f),r(o,f)},Y=o=>{Q(o,{children:(f,S)=>{var c=$(),p=d(c),i=e(p),_=e(i);R(_,{class:"w-5 h-5 text-amber-500"}),a(2),t(i);var v=m(i,2);v.__click=g;var w=e(v);{var A=l=>{var x=X(),C=d(x);U(C,{class:"w-4 h-4"}),a(),r(l,x)},n=l=>{var x=Z(),C=d(x);B(C,{class:"w-4 h-4"}),a(),r(l,x)};E(w,l=>{P(b)?l(A):l(n,!1)})}t(v),t(p);var s=m(p,4),y=e(s);y.textContent=`{
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
}`,t(s);var k=m(s,2),I=e(k),F=m(e(I));z(F,{class:"w-3 h-3"}),t(I),a(2),t(k),r(f,c)},$$slots:{default:!0}})};E(j,o=>{G()?o(V):o(Y,!1)})}r(D,h)}M(["click"]);export{ce as A};
