import{B as e,H as t,I as n,K as r,L as i,O as a,Q as o,R as s,Z as c,at as l,c as u,ct as d,dt as f,ht as p,it as m,l as h,lt as g,mt as _,ot as v,rt as y,st as b,tt as x,vt as S,y as ee,yt as C,z as w}from"../chunks/C_OAeTtO.js";import"../chunks/xihTtKlq.js";import{t as T}from"../chunks/CfgaxGnT.js";import{A as E,H as D,I as O,l as k,p as A,u as j,w as M,x as N,z as P}from"../chunks/xZIB709U.js";import{t as te}from"../chunks/9xkTelJZ.js";var F=e(`<p>Connect your applications to Outlet via SMTP to send transactional or marketing emails. This
			allows you to use Outlet as your SMTP relay server.</p> <p class="mt-2 text-sm"><strong>Brand isolation:</strong> </p>`,1),ne=e(`<div class="flex items-center gap-3 mb-4"><!> <h2 class="text-lg font-medium text-text">Connection Settings</h2></div> <div class="space-y-4"><div class="grid grid-cols-2 gap-4"><div><label class="form-label">SMTP Host</label> <div class="flex gap-2"><code class="flex-1 bg-surface-tertiary px-3 py-2 rounded text-sm font-mono text-text"> </code> <!></div></div> <div><label class="form-label">SMTP Port</label> <code class="block bg-surface-tertiary px-3 py-2 rounded text-sm font-mono text-text"></code></div></div> <div class="grid grid-cols-2 gap-4"><div><label class="form-label">Username</label> <div class="flex gap-2"><code class="flex-1 bg-surface-tertiary px-3 py-2 rounded text-sm font-mono text-text"> </code> <!></div> <p class="mt-1 text-xs text-text-muted">Your brand slug (required for authentication)</p></div> <div><label class="form-label">Password</label> <code class="block bg-surface-tertiary px-3 py-2 rounded text-sm font-mono text-text">YOUR_API_KEY</code> <p class="mt-1 text-xs text-text-muted"><a class="text-primary hover:underline">Get an API key</a> from Brand Settings</p></div></div> <div class="pt-4 border-t border-border"><h3 class="font-medium text-text mb-3">Encryption</h3> <div class="flex items-center gap-2"><!> <span class="text-sm text-text-muted">Encryption upgrades automatically when available</span></div></div></div>`,1),I=e(`<div class="flex items-center gap-3 mb-4"><!> <h2 class="text-lg font-medium text-text">Custom Headers</h2></div> <p class="text-sm text-text-muted mb-4">Use these custom headers to control how Outlet processes your emails.</p> <div class="overflow-x-auto"><table class="w-full text-sm"><thead><tr class="border-b border-border"><th class="text-left py-2 font-medium text-text-muted">Header</th><th class="text-left py-2 font-medium text-text-muted">Values</th><th class="text-left py-2 font-medium text-text-muted">Description</th></tr></thead><tbody class="divide-y divide-border/50"><tr><td class="py-3 font-mono text-primary whitespace-nowrap">X-Outlet-Type</td><td class="py-3"><!> <!></td><td class="py-3 text-text-muted">Email type. Default: <code class="text-xs bg-surface-tertiary px-1 rounded">transactional</code></td></tr><tr><td class="py-3 font-mono text-primary whitespace-nowrap">X-Outlet-List</td><td class="py-3"><code class="text-xs bg-surface-tertiary px-1 rounded">slug</code></td><td class="py-3 text-text-muted">Associate email with a list (for marketing). List slug is scoped to your brand.</td></tr><tr><td class="py-3 font-mono text-primary whitespace-nowrap">X-Outlet-Tags</td><td class="py-3"><code class="text-xs bg-surface-tertiary px-1 rounded">tag1,tag2,tag3</code></td><td class="py-3 text-text-muted">Comma-separated tags to apply to recipient</td></tr><tr><td class="py-3 font-mono text-primary whitespace-nowrap">X-Outlet-Template</td><td class="py-3"><code class="text-xs bg-surface-tertiary px-1 rounded">slug</code></td><td class="py-3 text-text-muted">Use a predefined email template. Template slug is scoped to your brand.</td></tr><tr><td class="py-3 font-mono text-primary whitespace-nowrap">X-Outlet-Track</td><td class="py-3"><code class="text-xs bg-surface-tertiary px-1 rounded">opens,clicks</code> <code class="text-xs bg-surface-tertiary px-1 rounded">none</code></td><td class="py-3 text-text-muted">Enable/disable tracking. Default: <code class="text-xs bg-surface-tertiary px-1 rounded">opens,clicks</code></td></tr><tr><td class="py-3 font-mono text-primary whitespace-nowrap">X-Outlet-Meta-*</td><td class="py-3"><code class="text-xs bg-surface-tertiary px-1 rounded">any value</code></td><td class="py-3 text-text-muted">Custom metadata. E.g., <code class="text-xs bg-surface-tertiary px-1 rounded">X-Outlet-Meta-Order-ID: 12345</code></td></tr></tbody></table></div>`,1),L=e(`<h2 class="text-lg font-medium text-text mb-4">Code Examples</h2> <div class="mb-4"><!></div> <!>`,1),R=e(`<h2 class="text-lg font-medium text-text mb-2">SMTP Limits</h2> <p class="text-sm text-text-muted mb-4">These are the default limits for emails sent via the SMTP ingress server. Emails exceeding
			these limits will be rejected.</p> <div class="grid grid-cols-2 gap-4 text-sm"><div class="flex justify-between p-3 bg-surface-secondary rounded"><span class="text-text-muted">Max message size</span> <span class="text-text font-medium">25 MB</span></div> <div class="flex justify-between p-3 bg-surface-secondary rounded"><span class="text-text-muted">Max recipients per message</span> <span class="text-text font-medium">100</span></div></div>`,1),z=e(`<div class="space-y-6"><!> <!> <!> <!> <!></div>`);function B(e,B){p(B,!0);let V=()=>h(T,`$page`,H),[H,re]=u(),ie=b(window.location.hostname),U=g(``),W=f(()=>V().params.brandSlug);function G(e){navigator.clipboard.writeText(e),d(U,e,!0),setTimeout(()=>d(U,``),2e3)}let K=f(()=>ie),ae=[{id:`curl`,label:`cURL`},{id:`nodejs`,label:`Node.js`},{id:`python`,label:`Python`},{id:`go`,label:`Go`},{id:`php`,label:`PHP`}],q=g(`curl`),oe=f(()=>`# Send email via SMTP using curl
curl --url "smtp://${r(K)}:587" \\
  --ssl-reqd \\
  --user "${r(W)}:YOUR_API_KEY" \\
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
EOF`),se=f(()=>`import nodemailer from 'nodemailer';

const transporter = nodemailer.createTransport({
  host: '${r(K)}',
  port: 587,
  secure: false, // STARTTLS
  auth: {
    user: '${r(W)}',
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
});`),ce=f(()=>`import smtplib
from email.mime.text import MIMEText
from email.mime.multipart import MIMEMultipart

msg = MIMEMultipart('alternative')
msg['From'] = 'you@example.com'
msg['To'] = 'recipient@example.com'
msg['Subject'] = 'Hello from Outlet'
msg['X-Outlet-Type'] = 'transactional'
msg['X-Outlet-Track'] = 'opens,clicks'

msg.attach(MIMEText('<h1>Welcome!</h1>', 'html'))

with smtplib.SMTP('${r(K)}', 587) as server:
    server.starttls()
    server.login('${r(W)}', 'YOUR_API_KEY')
    server.sendmail(msg['From'], msg['To'], msg.as_string())`),le=f(()=>`package main

import (
    "net/smtp"
)

func main() {
    auth := smtp.PlainAuth("", "${r(W)}", "YOUR_API_KEY", "${r(K)}")

    msg := []byte("From: you@example.com\\r\\n" +
        "To: recipient@example.com\\r\\n" +
        "Subject: Hello from Outlet\\r\\n" +
        "X-Outlet-Type: transactional\\r\\n" +
        "Content-Type: text/html\\r\\n" +
        "\\r\\n" +
        "<h1>Welcome!</h1>")

    err := smtp.SendMail("${r(K)}:587", auth,
        "you@example.com", []string{"recipient@example.com"}, msg)
}`),ue=f(()=>`use PHPMailer\\PHPMailer\\PHPMailer;

$mail = new PHPMailer(true);

$mail->isSMTP();
$mail->Host = '${r(K)}';
$mail->Port = 587;
$mail->SMTPAuth = true;
$mail->Username = '${r(W)}';
$mail->Password = 'YOUR_API_KEY';
$mail->SMTPSecure = PHPMailer::ENCRYPTION_STARTTLS;

$mail->setFrom('you@example.com');
$mail->addAddress('recipient@example.com');
$mail->Subject = 'Hello from Outlet';
$mail->isHTML(true);
$mail->Body = '<h1>Welcome!</h1>';

$mail->addCustomHeader('X-Outlet-Type', 'transactional');
$mail->addCustomHeader('X-Outlet-Track', 'opens,clicks');

$mail->send();`),de=f(()=>({curl:{code:r(oe),language:`bash`},nodejs:{code:r(se),language:`javascript`},python:{code:r(ce),language:`python`},go:{code:r(le),language:`go`},php:{code:r(ue),language:`php`}})),J=f(()=>r(de)[r(q)]);var Y=z();a(`1utgyeu`,e=>{c(()=>{x.title=`SMTP - Settings`})});var X=y(Y);M(X,{type:`info`,title:`SMTP Ingress`,children:(e,t)=>{var n=F(),a=v(m(n),2),c=v(y(a));C(a),o(()=>i(c,` Your SMTP username must match your brand slug (${r(W)??``}) to ensure emails are sent from the correct brand.`)),s(e,n)},$$slots:{default:!0}});var Z=v(X,2);D(Z,{children:(e,a)=>{var c=ne(),u=m(c),d=y(u);te(d,{class:`h-5 w-5 text-primary`}),S(2),C(u);var f=v(u,2),p=y(f),h=y(p),g=v(y(h),2),_=y(g),b=l(_,!0),x=v(_,2);N(x,{type:`secondary`,size:`sm`,onclick:()=>G(r(K)),children:(e,t)=>{var i=w(),a=m(i),o=e=>{P(e,{class:`h-4 w-4 text-green-500`})},c=e=>{O(e,{class:`h-4 w-4`})};n(a,e=>{r(U)===r(K)?e(o):e(c,-1)}),s(e,i)},$$slots:{default:!0}}),C(g),C(h);var T=v(h,2),E=v(y(T),2);E.textContent=`587`,C(T),C(p);var D=v(p,2),k=y(D),A=v(y(k),2),M=y(A),F=l(M,!0),I=v(M,2);N(I,{type:`secondary`,size:`sm`,onclick:()=>G(r(W)),children:(e,t)=>{var i=w(),a=m(i),o=e=>{P(e,{class:`h-4 w-4 text-green-500`})},c=e=>{O(e,{class:`h-4 w-4`})};n(a,e=>{r(U)===r(W)?e(o):e(c,-1)}),s(e,i)},$$slots:{default:!0}}),C(A),S(2),C(k);var L=v(k,2),R=v(y(L),4),z=y(R);S(),C(R),C(L),C(D);var B=v(D,2),V=v(y(B),2),H=y(V);j(H,{type:`success`,children:(e,n)=>{S();var r=t(`STARTTLS`);s(e,r)},$$slots:{default:!0}}),S(2),C(V),C(B),C(f),o(()=>{i(b,r(K)),i(F,r(W)),ee(z,`href`,`/${r(W)??``}/settings`)}),s(e,c)},$$slots:{default:!0}});var Q=v(Z,2);D(Q,{children:(e,n)=>{var r=I(),i=m(r),a=y(i);E(a,{class:`h-5 w-5 text-primary`}),S(2),C(i);var o=v(i,4),c=y(o),l=v(y(c)),u=y(l),d=v(y(u)),f=y(d);j(f,{type:`secondary`,children:(e,n)=>{S();var r=t(`transactional`);s(e,r)},$$slots:{default:!0}});var p=v(f,2);j(p,{type:`secondary`,children:(e,n)=>{S();var r=t(`marketing`);s(e,r)},$$slots:{default:!0}}),C(d),S(),C(u),S(5),C(l),C(c),C(o),s(e,r)},$$slots:{default:!0}});var $=v(Q,2);D($,{children:(e,t)=>{var n=L(),i=v(m(n),2),a=y(i);A(a,{get tabs(){return ae},variant:`pills`,get activeTab(){return r(q)},set activeTab(e){d(q,e,!0)}}),C(i);var o=v(i,2);k(o,{get code(){return r(J).code},get language(){return r(J).language}}),s(e,n)},$$slots:{default:!0}});var fe=v($,2);D(fe,{children:(e,t)=>{var n=R();S(4),s(e,n)},$$slots:{default:!0}}),C(Y),s(e,Y),_(),re()}export{B as component};