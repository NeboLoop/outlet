import"../chunks/DsnmJJEf.js";import{p as Ot,a as wt,d as St,e as kt,h as e,f as $,t as et,i as t,c as a,b as R,j as p,s as at,g as Ct,r as s,n as i}from"../chunks/BtjfW9_B.js";import{s as B}from"../chunks/DEsepwd4.js";import{a as c,c as st,t as L,f as _}from"../chunks/DnSY09wi.js";import{i as rt}from"../chunks/Cn_LYL0x.js";import{h as Et}from"../chunks/DvNfZKU3.js";import{s as Ht}from"../chunks/B8beEWxR.js";import{s as Xt,a as At}from"../chunks/x6Lfh__E.js";import{A as It,C as k,B as ot,g as Yt,h as jt,b as lt,i as dt}from"../chunks/DTNgJKyn.js";import{B as D}from"../chunks/D2EkXZ6e.js";import{p as Ut}from"../chunks/CYDbC9S2.js";import{S as Rt}from"../chunks/DSFCh1s-.js";import{M as Bt}from"../chunks/Bvzx66iv.js";var Lt=_(`<p>Connect your applications to Outlet via SMTP to send transactional or marketing emails. This
			allows you to use Outlet as your SMTP relay server.</p> <p class="mt-2 text-sm"><strong>Brand isolation:</strong> </p>`,1),Dt=_('<div class="flex items-center gap-3 mb-4"><!> <h2 class="text-lg font-medium text-text">Connection Settings</h2></div> <div class="space-y-4"><div class="grid grid-cols-2 gap-4"><div><label class="form-label">SMTP Host</label> <div class="flex gap-2"><code class="flex-1 bg-surface-tertiary px-3 py-2 rounded text-sm font-mono text-text"> </code> <!></div></div> <div><label class="form-label">SMTP Port</label> <code class="block bg-surface-tertiary px-3 py-2 rounded text-sm font-mono text-text"></code></div></div> <div class="grid grid-cols-2 gap-4"><div><label class="form-label">Username</label> <div class="flex gap-2"><code class="flex-1 bg-surface-tertiary px-3 py-2 rounded text-sm font-mono text-text"> </code> <!></div> <p class="mt-1 text-xs text-text-muted">Your brand slug (required for authentication)</p></div> <div><label class="form-label">Password</label> <code class="block bg-surface-tertiary px-3 py-2 rounded text-sm font-mono text-text">YOUR_API_KEY</code> <p class="mt-1 text-xs text-text-muted"><a class="text-primary hover:underline">Get an API key</a> from Brand Settings</p></div></div> <div class="pt-4 border-t border-border"><h3 class="font-medium text-text mb-3">Encryption</h3> <div class="flex items-center gap-2"><!> <span class="text-sm text-text-muted">Encryption upgrades automatically when available</span></div></div></div>',1),Ft=_('<div class="flex items-center gap-3 mb-4"><!> <h2 class="text-lg font-medium text-text">Custom Headers</h2></div> <p class="text-sm text-text-muted mb-4">Use these custom headers to control how Outlet processes your emails.</p> <div class="overflow-x-auto"><table class="w-full text-sm"><thead><tr class="border-b border-border"><th class="text-left py-2 font-medium text-text-muted">Header</th><th class="text-left py-2 font-medium text-text-muted">Values</th><th class="text-left py-2 font-medium text-text-muted">Description</th></tr></thead><tbody class="divide-y divide-border/50"><tr><td class="py-3 font-mono text-primary whitespace-nowrap">X-Outlet-Type</td><td class="py-3"><!> <!></td><td class="py-3 text-text-muted">Email type. Default: <code class="text-xs bg-surface-tertiary px-1 rounded">transactional</code></td></tr><tr><td class="py-3 font-mono text-primary whitespace-nowrap">X-Outlet-List</td><td class="py-3"><code class="text-xs bg-surface-tertiary px-1 rounded">slug</code></td><td class="py-3 text-text-muted">Associate email with a list (for marketing). List slug is scoped to your brand.</td></tr><tr><td class="py-3 font-mono text-primary whitespace-nowrap">X-Outlet-Tags</td><td class="py-3"><code class="text-xs bg-surface-tertiary px-1 rounded">tag1,tag2,tag3</code></td><td class="py-3 text-text-muted">Comma-separated tags to apply to recipient</td></tr><tr><td class="py-3 font-mono text-primary whitespace-nowrap">X-Outlet-Template</td><td class="py-3"><code class="text-xs bg-surface-tertiary px-1 rounded">slug</code></td><td class="py-3 text-text-muted">Use a predefined email template. Template slug is scoped to your brand.</td></tr><tr><td class="py-3 font-mono text-primary whitespace-nowrap">X-Outlet-Track</td><td class="py-3"><code class="text-xs bg-surface-tertiary px-1 rounded">opens,clicks</code> <code class="text-xs bg-surface-tertiary px-1 rounded">none</code></td><td class="py-3 text-text-muted">Enable/disable tracking. Default: <code class="text-xs bg-surface-tertiary px-1 rounded">opens,clicks</code></td></tr><tr><td class="py-3 font-mono text-primary whitespace-nowrap">X-Outlet-Meta-*</td><td class="py-3"><code class="text-xs bg-surface-tertiary px-1 rounded">any value</code></td><td class="py-3 text-text-muted">Custom metadata. E.g., <code class="text-xs bg-surface-tertiary px-1 rounded">X-Outlet-Meta-Order-ID: 12345</code></td></tr></tbody></table></div>',1),Kt=_('<h2 class="text-lg font-medium text-text mb-4">Code Examples</h2> <div class="mb-4"><!></div> <!>',1),Wt=_(`<h2 class="text-lg font-medium text-text mb-2">SMTP Limits</h2> <p class="text-sm text-text-muted mb-4">These are the default limits for emails sent via the SMTP ingress server. Emails exceeding
			these limits will be rejected.</p> <div class="grid grid-cols-2 gap-4 text-sm"><div class="flex justify-between p-3 bg-surface-secondary rounded"><span class="text-text-muted">Max message size</span> <span class="text-text font-medium">25 MB</span></div> <div class="flex justify-between p-3 bg-surface-secondary rounded"><span class="text-text-muted">Max recipients per message</span> <span class="text-text font-medium">100</span></div></div>`,1),zt=_('<div class="space-y-6"><!> <!> <!> <!> <!></div>');function le(it,ct){Ot(ct,!0);const nt=()=>At(Ut,"$page",mt),[mt,pt]=Xt();let ut=wt(window.location.hostname),O=at(""),n=p(()=>nt().params.brandSlug);function F(o){navigator.clipboard.writeText(o),R(O,o,!0),setTimeout(()=>R(O,""),2e3)}const m=p(()=>ut),T=587,xt=[{id:"curl",label:"cURL"},{id:"nodejs",label:"Node.js"},{id:"python",label:"Python"},{id:"go",label:"Go"},{id:"php",label:"PHP"}];let C=at("curl");const vt=p(()=>`# Send email via SMTP using curl
curl --url "smtp://${t(m)}:${T}" \\
  --ssl-reqd \\
  --user "${t(n)}:YOUR_API_KEY" \\
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
EOF`),gt=p(()=>`import nodemailer from 'nodemailer';

const transporter = nodemailer.createTransport({
  host: '${t(m)}',
  port: ${T},
  secure: false, // STARTTLS
  auth: {
    user: '${t(n)}',
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
});`),ft=p(()=>`import smtplib
from email.mime.text import MIMEText
from email.mime.multipart import MIMEMultipart

msg = MIMEMultipart('alternative')
msg['From'] = 'you@example.com'
msg['To'] = 'recipient@example.com'
msg['Subject'] = 'Hello from Outlet'
msg['X-Outlet-Type'] = 'transactional'
msg['X-Outlet-Track'] = 'opens,clicks'

msg.attach(MIMEText('<h1>Welcome!</h1>', 'html'))

with smtplib.SMTP('${t(m)}', ${T}) as server:
    server.starttls()
    server.login('${t(n)}', 'YOUR_API_KEY')
    server.sendmail(msg['From'], msg['To'], msg.as_string())`),yt=p(()=>`package main

import (
    "net/smtp"
)

func main() {
    auth := smtp.PlainAuth("", "${t(n)}", "YOUR_API_KEY", "${t(m)}")

    msg := []byte("From: you@example.com\\r\\n" +
        "To: recipient@example.com\\r\\n" +
        "Subject: Hello from Outlet\\r\\n" +
        "X-Outlet-Type: transactional\\r\\n" +
        "Content-Type: text/html\\r\\n" +
        "\\r\\n" +
        "<h1>Welcome!</h1>")

    err := smtp.SendMail("${t(m)}:${T}", auth,
        "you@example.com", []string{"recipient@example.com"}, msg)
}`),ht=p(()=>`use PHPMailer\\PHPMailer\\PHPMailer;

$mail = new PHPMailer(true);

$mail->isSMTP();
$mail->Host = '${t(m)}';
$mail->Port = ${T};
$mail->SMTPAuth = true;
$mail->Username = '${t(n)}';
$mail->Password = 'YOUR_API_KEY';
$mail->SMTPSecure = PHPMailer::ENCRYPTION_STARTTLS;

$mail->setFrom('you@example.com');
$mail->addAddress('recipient@example.com');
$mail->Subject = 'Hello from Outlet';
$mail->isHTML(true);
$mail->Body = '<h1>Welcome!</h1>';

$mail->addCustomHeader('X-Outlet-Type', 'transactional');
$mail->addCustomHeader('X-Outlet-Track', 'opens,clicks');

$mail->send();`);let bt=p(()=>({curl:{code:t(vt),language:"bash"},nodejs:{code:t(gt),language:"javascript"},python:{code:t(ft),language:"python"},go:{code:t(yt),language:"go"},php:{code:t(ht),language:"php"}})),K=p(()=>t(bt)[t(C)]);var E=zt();Et("1utgyeu",o=>{kt(()=>{Ct.title="SMTP - Settings"})});var W=a(E);It(W,{type:"info",title:"SMTP Ingress",children:(o,w)=>{var l=Lt(),r=e($(l),2),x=e(a(r));s(r),et(()=>B(x,` Your SMTP username must match your brand slug (${t(n)??""}) to ensure emails are sent from the correct brand.`)),c(o,l)},$$slots:{default:!0}});var z=e(W,2);k(z,{children:(o,w)=>{var l=Dt(),r=$(l),x=a(r);Rt(x,{class:"h-5 w-5 text-primary"}),i(2),s(r);var v=e(r,2),u=a(v),y=a(u),P=e(a(y),2),h=a(P),S=a(h,!0);s(h);var H=e(h,2);ot(H,{type:"secondary",size:"sm",onclick:()=>F(t(m)),children:(M,tt)=>{var f=st(),Y=$(f);{var j=d=>{lt(d,{class:"h-4 w-4 text-green-500"})},U=d=>{dt(d,{class:"h-4 w-4"})};rt(Y,d=>{t(O)===t(m)?d(j):d(U,!1)})}c(M,f)},$$slots:{default:!0}}),s(P),s(y);var b=e(y,2),X=e(a(b),2);X.textContent="587",s(b),s(u);var g=e(u,2),A=a(g),G=e(a(A),2),I=a(G),_t=a(I,!0);s(I);var Tt=e(I,2);ot(Tt,{type:"secondary",size:"sm",onclick:()=>F(t(n)),children:(M,tt)=>{var f=st(),Y=$(f);{var j=d=>{lt(d,{class:"h-4 w-4 text-green-500"})},U=d=>{dt(d,{class:"h-4 w-4"})};rt(Y,d=>{t(O)===t(n)?d(j):d(U,!1)})}c(M,f)},$$slots:{default:!0}}),s(G),i(2),s(A);var V=e(A,2),J=e(a(V),4),Pt=a(J);i(),s(J),s(V),s(g);var Q=e(g,2),Z=e(a(Q),2),Mt=a(Z);D(Mt,{type:"success",children:(M,tt)=>{i();var f=L("STARTTLS");c(M,f)},$$slots:{default:!0}}),i(2),s(Z),s(Q),s(v),et(()=>{B(S,t(m)),B(_t,t(n)),Ht(Pt,"href",`/${t(n)??""}/settings`)}),c(o,l)},$$slots:{default:!0}});var N=e(z,2);k(N,{children:(o,w)=>{var l=Ft(),r=$(l),x=a(r);Bt(x,{class:"h-5 w-5 text-primary"}),i(2),s(r);var v=e(r,4),u=a(v),y=e(a(u)),P=a(y),h=e(a(P)),S=a(h);D(S,{type:"secondary",children:(b,X)=>{i();var g=L("transactional");c(b,g)},$$slots:{default:!0}});var H=e(S,2);D(H,{type:"secondary",children:(b,X)=>{i();var g=L("marketing");c(b,g)},$$slots:{default:!0}}),s(h),i(),s(P),i(5),s(y),s(u),s(v),c(o,l)},$$slots:{default:!0}});var q=e(N,2);k(q,{children:(o,w)=>{var l=Kt(),r=e($(l),2),x=a(r);Yt(x,{get tabs(){return xt},variant:"pills",get activeTab(){return t(C)},set activeTab(u){R(C,u,!0)}}),s(r);var v=e(r,2);jt(v,{get code(){return t(K).code},get language(){return t(K).language}}),c(o,l)},$$slots:{default:!0}});var $t=e(q,2);k($t,{children:(o,w)=>{var l=Wt();i(4),c(o,l)},$$slots:{default:!0}}),s(E),c(it,E),St(),pt()}export{le as component};
