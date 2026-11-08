import{B as e,H as t,I as n,K as r,L as i,O as a,P as o,Q as s,R as c,Z as l,at as u,ct as d,dt as f,ht as p,it as m,lt as h,mt as g,ot as _,r as v,rt as y,st as b,tt as ee,vt as x,yt as S,z as C}from"../chunks/C_OAeTtO.js";import"../chunks/xihTtKlq.js";import{Cn as te,I as ne,Jt as re}from"../chunks/CgJezRIb.js";import{C as ie,H as w,I as T,N as E,O as ae,R as oe,V as D,et as O,o as se,rt as k,tt as ce,w as le}from"../chunks/BM5rBqT1.js";import{t as ue}from"../chunks/Cvoi-i3u.js";import{t as de}from"../chunks/L0WEjQit.js";import{t as fe}from"../chunks/BkzwieoE.js";import{t as pe}from"../chunks/Dn9WcqgK.js";var me=e(`<p> </p>`),he=e(`<!> Create Key`,1),ge=e(`<div class="flex justify-center py-8"><!></div>`),_e=e(`<div class="mt-6 text-center py-8 text-text-muted"><!> <p class="mt-2">No MCP keys yet</p> <p class="text-sm">Create one to connect an AI assistant over MCP</p></div>`),ve=e(`<tr class="border-b border-border/50"><td class="py-3 font-medium"> </td><td class="py-3 font-mono text-text-muted"> </td><td class="py-3 text-text-muted"> </td><td class="py-3 text-text-muted"> </td><td class="py-3 text-right"><!></td></tr>`),ye=e(`<div class="mt-6 overflow-x-auto"><table class="w-full text-sm"><thead><tr class="border-b border-border"><th class="text-left py-2 font-medium text-text-muted">Name</th><th class="text-left py-2 font-medium text-text-muted">Key</th><th class="text-left py-2 font-medium text-text-muted">Last Used</th><th class="text-left py-2 font-medium text-text-muted">Created</th><th class="text-right py-2 font-medium text-text-muted">Actions</th></tr></thead><tbody></tbody></table></div>`),be=e(`<div class="flex items-center justify-between"><div><h2 class="text-lg font-medium text-text mb-1">MCP keys</h2> <p class="text-sm text-text-muted">For AI assistants such as Claude that manage Outlet through MCP. They don't work for sending email: apps use their brand's API key, below.</p></div> <!></div> <!>`,1),xe=e(`<div class="flex items-center gap-3 mb-4"><!> <div><h2 class="text-lg font-medium text-text">Sending API and SMTP</h2> <p class="text-sm text-text-muted">How your apps send email. Each brand has its own API key, shown in that brand's settings.</p></div></div> <div class="space-y-4"><div><label class="form-label">Base URL</label> <div class="flex gap-2"><!> <!></div></div> <div><label class="form-label">Authentication</label> <p class="text-sm text-text-muted mb-2">Send the brand's API key in the X-API-Key header:</p> <pre class="bg-surface-tertiary p-3 rounded text-sm overflow-x-auto font-mono">X-API-Key: YOUR_BRAND_API_KEY</pre></div></div>`,1),Se=e(`<h2 class="text-lg font-medium text-text mb-4">Code Examples</h2> <div class="mb-4"><!></div> <!>`,1),Ce=e(`<!> <!>`,1),we=e(`<div class="flex justify-end gap-3"><!></div>`),Te=e(`<p>This is the only time you'll see this key. Copy it now and store it securely.</p>`),Ee=e(`<div class="space-y-4"><!> <div><label class="form-label">API Key</label> <div class="flex gap-2"><!> <!></div></div></div>`),De=e(`<div class="space-y-4"><div><label for="key-name" class="form-label">Key Name</label> <!> <p class="mt-1 text-xs text-text-muted">A descriptive name to identify this key</p></div></div>`),Oe=e(`<div class="space-y-6"><!> <!> <!> <!> <!></div> <!> <!>`,1);function A(e,A){p(A,!0);let j=b(window.location.origin.replace(`:5173`,`:8888`)),M=f(()=>`${j}/sdk/v1`),N=h(!0),P=h(!1),F=h(``),I=h(``),L=h(``),R=h(b([])),z=h(!1),B=h(``),V=h(``),H=h(!1),U=h(``),W=h(!1),ke=[{id:`curl`,label:`cURL`},{id:`nodejs`,label:`Node.js`},{id:`python`,label:`Python`},{id:`go`,label:`Go`},{id:`php`,label:`PHP`},{id:`smtp`,label:`SMTP`}],G=h(`curl`),Ae=f(()=>`# Send a transactional template
curl -X POST "${r(M)}/emails" \\
  -H "X-API-Key: YOUR_BRAND_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "to": "customer@example.com",
    "template_slug": "password-reset",
    "variables": { "name": "Ana", "reset_url": "https://app.example.com/reset/abc" }
  }'

# Or send without a template
curl -X POST "${r(M)}/emails" \\
  -H "X-API-Key: YOUR_BRAND_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "to": "customer@example.com",
    "subject": "Your receipt",
    "body": "<p>Thanks for your order.</p>",
    "text_body": "Thanks for your order."
  }'

# Check delivery
curl "${r(M)}/emails/MESSAGE_ID" -H "X-API-Key: YOUR_BRAND_API_KEY"`),je=f(()=>`const res = await fetch('${r(M)}/emails', {
  method: 'POST',
  headers: {
    'X-API-Key': process.env.OUTLET_API_KEY,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    to: 'customer@example.com',
    template_slug: 'password-reset',
    variables: { name: 'Ana', reset_url: 'https://app.example.com/reset/abc' }
  })
});
const result = await res.json();
if (!result.success) throw new Error(result.message); // e.g. status "suppressed"
console.log(result.message_id);`),Me=f(()=>`import os, requests

res = requests.post(
    '${r(M)}/emails',
    headers={'X-API-Key': os.environ['OUTLET_API_KEY']},
    json={
        'to': 'customer@example.com',
        'template_slug': 'password-reset',
        'variables': {'name': 'Ana', 'reset_url': 'https://app.example.com/reset/abc'},
    },
)
result = res.json()
if not result['success']:
    raise RuntimeError(result.get('message'))  # e.g. status "suppressed"
print(result['message_id'])`),Ne=f(()=>`payload, _ := json.Marshal(map[string]any{
    "to":            "customer@example.com",
    "template_slug": "password-reset",
    "variables":     map[string]string{"name": "Ana", "reset_url": "https://app.example.com/reset/abc"},
})
req, _ := http.NewRequest("POST", "${r(M)}/emails", bytes.NewReader(payload))
req.Header.Set("X-API-Key", os.Getenv("OUTLET_API_KEY"))
req.Header.Set("Content-Type", "application/json")
resp, err := http.DefaultClient.Do(req)
if err != nil {
    return err
}
defer resp.Body.Close()

var result struct {
    Success   bool   \`json:"success"\`
    MessageID string \`json:"message_id"\`
    Message   string \`json:"message"\`
}
json.NewDecoder(resp.Body).Decode(&result)`),Pe=f(()=>`<?php
$ch = curl_init('${r(M)}/emails');
curl_setopt_array($ch, [
    CURLOPT_POST => true,
    CURLOPT_RETURNTRANSFER => true,
    CURLOPT_HTTPHEADER => [
        'X-API-Key: ' . getenv('OUTLET_API_KEY'),
        'Content-Type: application/json',
    ],
    CURLOPT_POSTFIELDS => json_encode([
        'to' => 'customer@example.com',
        'template_slug' => 'password-reset',
        'variables' => ['name' => 'Ana', 'reset_url' => 'https://app.example.com/reset/abc'],
    ]),
]);
$result = json_decode(curl_exec($ch), true);`),Fe=f(()=>`Host:      ${j.replace(/^https?:\/\//,``).replace(/:\d+$/,``)}
Port:      587 (STARTTLS)
Username:  your brand slug
Password:  your brand API key

Optional headers:
  X-Outlet-Tags: welcome,onboarding
  X-Outlet-Meta-OrderID: 12345`),Ie=f(()=>({curl:{code:r(Ae),language:`bash`},nodejs:{code:r(je),language:`javascript`},python:{code:r(Me),language:`python`},go:{code:r(Ne),language:`go`},php:{code:r(Pe),language:`php`},smtp:{code:r(Fe),language:`text`}})),K=f(()=>r(Ie)[r(G)]);v(async()=>{await q()});async function q(){d(N,!0),d(F,``);try{let e=await re();d(R,e.keys||[],!0)}catch{console.log(`No API keys yet`)}finally{d(N,!1)}}async function Le(){if(!r(B).trim()){d(F,`Please enter a name for the API key`);return}d(P,!0),d(F,``);try{let e=await ne({name:r(B).trim()});d(V,e.key||``,!0),await q(),d(I,`API key created successfully. Copy it now - you won't be able to see it again!`)}catch(e){d(F,e.message||`Failed to create API key`,!0)}finally{d(P,!1)}}function Re(e){d(U,e,!0),d(H,!0)}async function ze(){d(W,!0);try{await te({},r(U)),await q(),d(I,`API key revoked`)}catch(e){d(F,e.message||`Failed to revoke API key`,!0)}finally{d(W,!1)}}function J(e){navigator.clipboard.writeText(e),d(L,e,!0),setTimeout(()=>d(L,``),2e3)}function Y(){d(z,!1),d(B,``),d(V,``)}var X=Oe();a(`1kyawqp`,e=>{l(()=>{ee.title=`API - Settings`})});var Z=m(X),Be=y(Z),Ve=e=>{D(e,{type:`error`,title:`Error`,onclose:()=>d(F,``),children:(e,t)=>{var n=me(),a=u(n,!0);s(()=>i(a,r(F))),c(e,n)},$$slots:{default:!0}})};n(Be,e=>{r(F)&&e(Ve)});var Q=_(Be,2),He=e=>{D(e,{type:`success`,title:`Success`,onclose:()=>d(I,``),children:(e,t)=>{var n=me(),a=u(n,!0);s(()=>i(a,r(I))),c(e,n)},$$slots:{default:!0}})};n(Q,e=>{r(I)&&e(He)});var Ue=_(Q,2);w(Ue,{children:(e,t)=>{var a=be(),l=m(a),f=_(y(l),2);T(f,{type:`primary`,size:`sm`,onclick:()=>d(z,!0),children:(e,t)=>{var n=he(),r=m(n);de(r,{class:`mr-1.5 h-4 w-4`}),x(),c(e,n)},$$slots:{default:!0}}),S(l);var p=_(l,2),h=e=>{var t=ge(),n=y(t);oe(n,{}),S(t),c(e,t)},g=e=>{var t=_e(),n=y(t);ue(n,{class:`mx-auto h-12 w-12 opacity-50`}),x(4),S(t),c(e,t)},v=e=>{var t=ye(),n=y(t),a=_(y(n));o(a,21,()=>r(R),e=>e.id,(e,t)=>{var n=ve(),a=y(n),o=u(a,!0),l=_(a),d=u(l),f=_(l),p=u(f,!0),m=_(f),h=u(m,!0),g=_(m),v=y(g);T(v,{type:`danger`,size:`icon`,onclick:()=>Re(r(t).id),children:(e,t)=>{fe(e,{class:`h-4 w-4`})},$$slots:{default:!0}}),S(g),S(n),s((e,n)=>{i(o,r(t).name),i(d,`${r(t).key_prefix??``}...`),i(p,e),i(h,n)},[()=>pe(r(t).last_used)||`Never`,()=>pe(r(t).created_at)||`Never`]),c(e,n)}),S(a),S(n),S(t),c(e,t)};n(p,e=>{r(N)?e(h):r(R).length===0?e(g,1):e(v,-1)}),c(e,a)},$$slots:{default:!0}});var We=_(Ue,2);w(We,{children:(e,t)=>{var i=xe(),a=m(i),o=y(a);ce(o,{class:`h-5 w-5 text-primary`}),x(2),S(a);var s=_(a,2),l=y(s),u=_(y(l),2),d=y(u);E(d,{type:`text`,get value(){return r(M)},readonly:!0,class:`font-mono text-sm`});var f=_(d,2);T(f,{type:`secondary`,onclick:()=>J(r(M)),children:(e,t)=>{var i=C(),a=m(i),o=e=>{k(e,{class:`h-4 w-4 text-green-500`})},s=e=>{O(e,{class:`h-4 w-4`})};n(a,e=>{r(L)===r(M)?e(o):e(s,-1)}),c(e,i)},$$slots:{default:!0}}),S(u),S(l),x(2),S(s),c(e,i)},$$slots:{default:!0}});var Ge=_(We,2);w(Ge,{children:(e,t)=>{var n=Se(),i=_(m(n),2),a=y(i);ae(a,{get tabs(){return ke},variant:`pills`,get activeTab(){return r(G)},set activeTab(e){d(G,e,!0)}}),S(i);var o=_(i,2);le(o,{get code(){return r(K).code},get language(){return r(K).language}}),c(e,n)},$$slots:{default:!0}}),S(Z);var $=_(Z,2);ie($,{title:`Create API Key`,onclose:Y,get show(){return r(z)},set show(e){d(z,e,!0)},footer:e=>{var a=we(),o=y(a),l=e=>{T(e,{type:`primary`,onclick:Y,children:(e,n)=>{x();var r=t(`Done`);c(e,r)},$$slots:{default:!0}})},u=e=>{var n=Ce(),a=m(n);T(a,{type:`secondary`,onclick:Y,children:(e,n)=>{x();var r=t(`Cancel`);c(e,r)},$$slots:{default:!0}});var o=_(a,2);{let e=f(()=>r(P)||!r(B).trim());T(o,{type:`primary`,onclick:Le,get disabled(){return r(e)},children:(e,n)=>{x();var a=t();s(()=>i(a,r(P)?`Creating...`:`Create Key`)),c(e,a)},$$slots:{default:!0}})}c(e,n)};n(o,e=>{r(V)?e(l):e(u,-1)}),S(a),c(e,a)},children:(e,t)=>{var i=C(),a=m(i),o=e=>{var t=Ee(),i=y(t);D(i,{type:`warning`,title:`Save your API key`,children:(e,t)=>{var n=Te();c(e,n)},$$slots:{default:!0}});var a=_(i,2),o=_(y(a),2),s=y(o);E(s,{type:`text`,get value(){return r(V)},readonly:!0,class:`font-mono text-sm`});var l=_(s,2);T(l,{type:`secondary`,onclick:()=>J(r(V)),children:(e,t)=>{var i=C(),a=m(i),o=e=>{k(e,{class:`h-4 w-4 text-green-500`})},s=e=>{O(e,{class:`h-4 w-4`})};n(a,e=>{r(L)===r(V)?e(o):e(s,-1)}),c(e,i)},$$slots:{default:!0}}),S(o),S(a),S(t),c(e,t)},s=e=>{var t=De(),n=y(t),i=_(y(n),2);E(i,{type:`text`,id:`key-name`,placeholder:`e.g., Production API, Development`,get value(){return r(B)},set value(e){d(B,e,!0)}}),x(2),S(n),S(t),c(e,t)};n(a,e=>{r(V)?e(o):e(s,-1)}),c(e,i)},$$slots:{footer:!0,default:!0}});var Ke=_($,2);{let e=f(()=>r(W)?`Revoking...`:`Revoke`);se(Ke,{title:`Revoke API Key`,description:`Are you sure you want to revoke this API key? This cannot be undone.`,get actionLabel(){return r(e)},actionType:`danger`,onAction:ze,onCancel:()=>d(H,!1),get open(){return r(H)},set open(e){d(H,e,!0)}})}c(e,X),g()}export{A as component};