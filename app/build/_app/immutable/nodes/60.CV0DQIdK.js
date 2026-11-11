import{$ as e,B as t,L as n,O as r,Q as i,R as a,U as o,V as s,at as c,bt as l,c as u,ct as d,ft as f,gt as p,ht as m,it as h,l as g,lt as _,nt as v,ot as y,q as b,st as x,ut as S,y as ee,yt as C,z as w}from"../chunks/BKXUg6nK.js";import"../chunks/xihTtKlq.js";import{t as T}from"../chunks/qyIwehnT.js";import{H as E,L as D,O,T as k,X as A,ct as j,ot as M,rt as N,w as P}from"../chunks/DySZ2YwQ.js";import{t as te}from"../chunks/B3PAAgA-.js";var F=s(`<p>Connect your applications to Outlet via SMTP to send transactional or marketing emails. This
			allows you to use Outlet as your SMTP relay server.</p> <p class="mt-2 text-sm"><strong>Brand isolation:</strong> </p>`,1),ne=s(`<div class="flex items-center gap-3 mb-4"><!> <h2 class="text-lg font-medium text-text">Connection Settings</h2></div> <div class="space-y-4"><div class="grid grid-cols-2 gap-4"><div><label class="form-label">SMTP Host</label> <div class="flex gap-2"><code class="flex-1 bg-surface-tertiary px-3 py-2 rounded text-sm font-mono text-text"> </code> <!></div></div> <div><label class="form-label">SMTP Port</label> <code class="block bg-surface-tertiary px-3 py-2 rounded text-sm font-mono text-text"></code></div></div> <div class="grid grid-cols-2 gap-4"><div><label class="form-label">Username</label> <div class="flex gap-2"><code class="flex-1 bg-surface-tertiary px-3 py-2 rounded text-sm font-mono text-text"> </code> <!></div> <p class="mt-1 text-xs text-text-muted">Your brand slug (required for authentication)</p></div> <div><label class="form-label">Password</label> <code class="block bg-surface-tertiary px-3 py-2 rounded text-sm font-mono text-text">YOUR_API_KEY</code> <p class="mt-1 text-xs text-text-muted"><a class="text-primary hover:underline">Get an API key</a> from Brand Settings</p></div></div> <div class="pt-4 border-t border-border"><h3 class="font-medium text-text mb-3">Encryption</h3> <div class="flex items-center gap-2"><!> <span class="text-sm text-text-muted">Encryption upgrades automatically when available</span></div></div></div>`,1),I=s(`<div class="flex items-center gap-3 mb-4"><!> <h2 class="text-lg font-medium text-text">Custom Headers</h2></div> <p class="text-sm text-text-muted mb-4">Use these custom headers to control how Outlet processes your emails.</p> <div class="overflow-x-auto"><table class="w-full text-sm"><thead><tr class="border-b border-border"><th class="text-left py-2 font-medium text-text-muted">Header</th><th class="text-left py-2 font-medium text-text-muted">Values</th><th class="text-left py-2 font-medium text-text-muted">Description</th></tr></thead><tbody class="divide-y divide-border/50"><tr><td class="py-3 font-mono text-primary whitespace-nowrap">X-Outlet-Type</td><td class="py-3"><!> <!></td><td class="py-3 text-text-muted">Email type. Default: <code class="text-xs bg-surface-tertiary px-1 rounded">transactional</code>. Marketing mail goes through the brand's marketing pathway, needs <code class="text-xs bg-surface-tertiary px-1 rounded">X-Outlet-List</code>, and gets an
							unsubscribe link and one-click unsubscribe headers.</td></tr><tr><td class="py-3 font-mono text-primary whitespace-nowrap">X-Outlet-List</td><td class="py-3"><code class="text-xs bg-surface-tertiary px-1 rounded">slug</code></td><td class="py-3 text-text-muted">Marketing mail only: the list the recipient subscribed to, by slug. A recipient who isn't an
							active subscriber of the list isn't sent to.</td></tr><tr><td class="py-3 font-mono text-primary whitespace-nowrap">X-Outlet-Tags</td><td class="py-3"><code class="text-xs bg-surface-tertiary px-1 rounded">tag1,tag2,tag3</code></td><td class="py-3 text-text-muted">Comma-separated tags recorded with the send, as the API's <code class="text-xs bg-surface-tertiary px-1 rounded">tags</code></td></tr><tr><td class="py-3 font-mono text-primary whitespace-nowrap">X-Outlet-Template</td><td class="py-3"><code class="text-xs bg-surface-tertiary px-1 rounded">slug</code></td><td class="py-3 text-text-muted">Send one of your transactional templates, by slug. Its subject and body replace the message's.</td></tr><tr><td class="py-3 font-mono text-primary whitespace-nowrap">X-Outlet-Track</td><td class="py-3"><code class="text-xs bg-surface-tertiary px-1 rounded">opens,clicks</code> <code class="text-xs bg-surface-tertiary px-1 rounded">none</code></td><td class="py-3 text-text-muted">Enable/disable tracking. Default: <code class="text-xs bg-surface-tertiary px-1 rounded">opens,clicks</code></td></tr><tr><td class="py-3 font-mono text-primary whitespace-nowrap">X-Outlet-Meta-*</td><td class="py-3"><code class="text-xs bg-surface-tertiary px-1 rounded">any value</code></td><td class="py-3 text-text-muted">Custom metadata recorded with the send. E.g., <code class="text-xs bg-surface-tertiary px-1 rounded">X-Outlet-Meta-Order-ID: 12345</code></td></tr></tbody></table></div>`,1),L=s(`<h2 class="text-lg font-medium text-text mb-4">Code Examples</h2> <div class="mb-4"><!></div> <!>`,1),R=s(`<h2 class="text-lg font-medium text-text mb-2">SMTP Limits</h2> <p class="text-sm text-text-muted mb-4">These are the default limits for emails sent via the SMTP ingress server. Emails exceeding
			these limits will be rejected.</p> <div class="grid grid-cols-2 gap-4 text-sm"><div class="flex justify-between p-3 bg-surface-secondary rounded"><span class="text-text-muted">Max message size</span> <span class="text-text font-medium">25 MB</span></div> <div class="flex justify-between p-3 bg-surface-secondary rounded"><span class="text-text-muted">Max recipients per message</span> <span class="text-text font-medium">100</span></div></div>`,1),z=s(`<div class="space-y-6"><!> <!> <!> <!> <!></div>`);function B(s,B){p(B,!0);let V=()=>g(T,`$page`,H),[H,re]=u(),ie=d(window.location.hostname),U=S(``),W=f(()=>V().params.brandSlug);function G(e){navigator.clipboard.writeText(e),_(U,e,!0),setTimeout(()=>_(U,``),2e3)}let K=f(()=>ie),ae=[{id:`curl`,label:`cURL`},{id:`nodejs`,label:`Node.js`},{id:`python`,label:`Python`},{id:`go`,label:`Go`},{id:`php`,label:`PHP`}],q=S(`curl`),oe=f(()=>`# Send email via SMTP using curl
curl --url "smtp://${b(K)}:587" \\
  --ssl-reqd \\
  --user "${b(W)}:YOUR_API_KEY" \\
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
  host: '${b(K)}',
  port: 587,
  secure: false, // STARTTLS
  auth: {
    user: '${b(W)}',
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

with smtplib.SMTP('${b(K)}', 587) as server:
    server.starttls()
    server.login('${b(W)}', 'YOUR_API_KEY')
    server.sendmail(msg['From'], msg['To'], msg.as_string())`),le=f(()=>`package main

import (
    "net/smtp"
)

func main() {
    auth := smtp.PlainAuth("", "${b(W)}", "YOUR_API_KEY", "${b(K)}")

    msg := []byte("From: you@example.com\\r\\n" +
        "To: recipient@example.com\\r\\n" +
        "Subject: Hello from Outlet\\r\\n" +
        "X-Outlet-Type: transactional\\r\\n" +
        "Content-Type: text/html\\r\\n" +
        "\\r\\n" +
        "<h1>Welcome!</h1>")

    err := smtp.SendMail("${b(K)}:587", auth,
        "you@example.com", []string{"recipient@example.com"}, msg)
}`),ue=f(()=>`use PHPMailer\\PHPMailer\\PHPMailer;

$mail = new PHPMailer(true);

$mail->isSMTP();
$mail->Host = '${b(K)}';
$mail->Port = 587;
$mail->SMTPAuth = true;
$mail->Username = '${b(W)}';
$mail->Password = 'YOUR_API_KEY';
$mail->SMTPSecure = PHPMailer::ENCRYPTION_STARTTLS;

$mail->setFrom('you@example.com');
$mail->addAddress('recipient@example.com');
$mail->Subject = 'Hello from Outlet';
$mail->isHTML(true);
$mail->Body = '<h1>Welcome!</h1>';

$mail->addCustomHeader('X-Outlet-Type', 'transactional');
$mail->addCustomHeader('X-Outlet-Track', 'opens,clicks');

$mail->send();`),de=f(()=>({curl:{code:b(oe),language:`bash`},nodejs:{code:b(se),language:`javascript`},python:{code:b(ce),language:`python`},go:{code:b(le),language:`go`},php:{code:b(ue),language:`php`}})),J=f(()=>b(de)[b(q)]);var Y=z();r(`1utgyeu`,e=>{i(()=>{v.title=`SMTP - Settings`})});var X=h(Y);E(X,{type:`info`,title:`SMTP Ingress`,children:(t,n)=>{var r=F(),i=x(c(r),2),o=x(h(i));l(i),e(()=>a(o,` Your SMTP username must match your brand slug (${b(W)??``}) to ensure emails are sent from the correct brand.`)),w(t,r)},$$slots:{default:!0}});var Z=x(X,2);j(Z,{children:(r,i)=>{var s=ne(),u=c(s),d=h(u);te(d,{class:`h-5 w-5 text-primary`}),C(2),l(u);var f=x(u,2),p=h(f),m=h(p),g=x(h(m),2),_=h(g),v=y(_,!0),S=x(_,2);D(S,{type:`secondary`,size:`sm`,onclick:()=>G(b(K)),children:(e,r)=>{var i=t(),a=c(i),o=e=>{M(e,{class:`h-4 w-4 text-success`})},s=e=>{N(e,{class:`h-4 w-4`})};n(a,e=>{b(U)===b(K)?e(o):e(s,-1)}),w(e,i)},$$slots:{default:!0}}),l(g),l(m);var T=x(m,2),E=x(h(T),2);E.textContent=`587`,l(T),l(p);var O=x(p,2),A=h(O),j=x(h(A),2),P=h(j),F=y(P,!0),I=x(P,2);D(I,{type:`secondary`,size:`sm`,onclick:()=>G(b(W)),children:(e,r)=>{var i=t(),a=c(i),o=e=>{M(e,{class:`h-4 w-4 text-success`})},s=e=>{N(e,{class:`h-4 w-4`})};n(a,e=>{b(U)===b(W)?e(o):e(s,-1)}),w(e,i)},$$slots:{default:!0}}),l(j),C(2),l(A);var L=x(A,2),R=x(h(L),4),z=h(R);C(),l(R),l(L),l(O);var B=x(O,2),V=x(h(B),2),H=h(V);k(H,{type:`success`,children:(e,t)=>{C();var n=o(`STARTTLS`);w(e,n)},$$slots:{default:!0}}),C(2),l(V),l(B),l(f),e(()=>{a(v,b(K)),a(F,b(W)),ee(z,`href`,`/${b(W)??``}/settings`)}),w(r,s)},$$slots:{default:!0}});var Q=x(Z,2);j(Q,{children:(e,t)=>{var n=I(),r=c(n),i=h(r);A(i,{class:`h-5 w-5 text-primary`}),C(2),l(r);var a=x(r,4),s=h(a),u=x(h(s)),d=h(u),f=x(h(d)),p=h(f);k(p,{type:`secondary`,children:(e,t)=>{C();var n=o(`transactional`);w(e,n)},$$slots:{default:!0}});var m=x(p,2);k(m,{type:`secondary`,children:(e,t)=>{C();var n=o(`marketing`);w(e,n)},$$slots:{default:!0}}),l(f),C(),l(d),C(5),l(u),l(s),l(a),w(e,n)},$$slots:{default:!0}});var $=x(Q,2);j($,{children:(e,t)=>{var n=L(),r=x(c(n),2),i=h(r);O(i,{get tabs(){return ae},variant:`pills`,get activeTab(){return b(q)},set activeTab(e){_(q,e,!0)}}),l(r);var a=x(r,2);P(a,{get code(){return b(J).code},get language(){return b(J).language}}),w(e,n)},$$slots:{default:!0}});var fe=x($,2);j(fe,{children:(e,t)=>{var n=R();C(4),w(e,n)},$$slots:{default:!0}}),l(Y),w(s,Y),m(),re()}export{B as component};