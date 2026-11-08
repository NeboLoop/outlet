import{B as e,H as t,I as n,K as r,L as i,O as a,P as o,Q as s,R as c,Z as l,at as u,ct as d,dt as f,ht as p,it as m,lt as h,mt as g,ot as _,r as v,rt as y,st as b,tt as ee,vt as x,yt as S,z as C}from"../chunks/C_OAeTtO.js";import"../chunks/xihTtKlq.js";import{Bt as te,j as ne,ln as re}from"../chunks/DhR1LtZA.js";import{$ as w,C as ie,I as T,N as E,O as ae,R as oe,V as D,at as O,et as se,nt as k,o as ce,w as le}from"../chunks/CxyUe1D2.js";import{t as ue}from"../chunks/DWMfdcG_.js";import{t as de}from"../chunks/DlBSIydL.js";import{t as fe}from"../chunks/jrndOowp.js";import{t as A}from"../chunks/Dn9WcqgK.js";var j=e(`<p> </p>`),pe=e(`<!> Create Key`,1),me=e(`<div class="flex justify-center py-8"><!></div>`),he=e(`<div class="mt-6 text-center py-8 text-text-muted"><!> <p class="mt-2">No MCP keys yet</p> <p class="text-sm">Create one to connect an AI assistant over MCP</p></div>`),ge=e(`<tr class="border-b border-border/50"><td class="py-3 font-medium"> </td><td class="py-3 font-mono text-text-muted"> </td><td class="py-3 text-text-muted"> </td><td class="py-3 text-text-muted"> </td><td class="py-3 text-right"><!></td></tr>`),_e=e(`<div class="mt-6 overflow-x-auto"><table class="w-full text-sm"><thead><tr class="border-b border-border"><th class="text-left py-2 font-medium text-text-muted">Name</th><th class="text-left py-2 font-medium text-text-muted">Key</th><th class="text-left py-2 font-medium text-text-muted">Last Used</th><th class="text-left py-2 font-medium text-text-muted">Created</th><th class="text-right py-2 font-medium text-text-muted">Actions</th></tr></thead><tbody></tbody></table></div>`),ve=e(`<div class="flex items-center justify-between"><div><h2 class="text-lg font-medium text-text mb-1">MCP keys</h2> <p class="text-sm text-text-muted">For AI assistants such as Claude that manage Outlet through MCP. They don't work for sending email: apps use their brand's API key, below.</p></div> <!></div> <!>`,1),ye=e(`<div class="flex items-center gap-3 mb-4"><!> <div><h2 class="text-lg font-medium text-text">Sending API and SMTP</h2> <p class="text-sm text-text-muted">How your apps send email. Each brand has its own API key, shown in that brand's settings.</p></div></div> <div class="space-y-4"><div><label class="form-label">Base URL</label> <div class="flex gap-2"><!> <!></div></div> <div><label class="form-label">Authentication</label> <p class="text-sm text-text-muted mb-2">Send the brand's API key in the X-API-Key header:</p> <pre class="bg-surface-tertiary p-3 rounded text-sm overflow-x-auto font-mono">X-API-Key: YOUR_BRAND_API_KEY</pre></div></div>`,1),be=e(`<h2 class="text-lg font-medium text-text mb-4">Code Examples</h2> <div class="mb-4"><!></div> <!>`,1),xe=e(`<!> <!>`,1),Se=e(`<div class="flex justify-end gap-3"><!></div>`),Ce=e(`<p>This is the only time you'll see this key. Copy it now and store it securely.</p>`),we=e(`<div class="space-y-4"><!> <div><label class="form-label">API Key</label> <div class="flex gap-2"><!> <!></div></div></div>`),Te=e(`<div class="space-y-4"><div><label for="key-name" class="form-label">Key Name</label> <!> <p class="mt-1 text-xs text-text-muted">A descriptive name to identify this key</p></div></div>`),Ee=e(`<div class="space-y-6"><!> <!> <!> <!> <!></div> <!> <!>`,1);function M(e,M){p(M,!0);let N=b(window.location.origin.replace(`:5173`,`:8888`)),P=f(()=>`${N}/sdk/v1`),F=h(!0),I=h(!1),L=h(``),R=h(``),z=h(``),B=h(b([])),V=h(!1),H=h(``),U=h(``),W=h(!1),G=h(``),K=h(!1),De=[{id:`curl`,label:`cURL`},{id:`nodejs`,label:`Node.js`},{id:`python`,label:`Python`},{id:`go`,label:`Go`},{id:`php`,label:`PHP`},{id:`smtp`,label:`SMTP`}],q=h(`curl`),Oe=f(()=>`# Send a transactional template
curl -X POST "${r(P)}/emails" \\
  -H "X-API-Key: YOUR_BRAND_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "to": "customer@example.com",
    "template_slug": "password-reset",
    "variables": { "name": "Ana", "reset_url": "https://app.example.com/reset/abc" }
  }'

# Or send without a template
curl -X POST "${r(P)}/emails" \\
  -H "X-API-Key: YOUR_BRAND_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "to": "customer@example.com",
    "subject": "Your receipt",
    "body": "<p>Thanks for your order.</p>",
    "text_body": "Thanks for your order."
  }'

# Check delivery
curl "${r(P)}/emails/MESSAGE_ID" -H "X-API-Key: YOUR_BRAND_API_KEY"`),ke=f(()=>`const res = await fetch('${r(P)}/emails', {
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
console.log(result.message_id);`),Ae=f(()=>`import os, requests

res = requests.post(
    '${r(P)}/emails',
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
print(result['message_id'])`),je=f(()=>`payload, _ := json.Marshal(map[string]any{
    "to":            "customer@example.com",
    "template_slug": "password-reset",
    "variables":     map[string]string{"name": "Ana", "reset_url": "https://app.example.com/reset/abc"},
})
req, _ := http.NewRequest("POST", "${r(P)}/emails", bytes.NewReader(payload))
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
json.NewDecoder(resp.Body).Decode(&result)`),Me=f(()=>`<?php
$ch = curl_init('${r(P)}/emails');
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
$result = json_decode(curl_exec($ch), true);`),Ne=f(()=>`Host:      ${N.replace(/^https?:\/\//,``).replace(/:\d+$/,``)}
Port:      587 (STARTTLS)
Username:  your brand slug
Password:  your brand API key

Optional headers:
  X-Outlet-Tags: welcome,onboarding
  X-Outlet-Meta-OrderID: 12345`),Pe=f(()=>({curl:{code:r(Oe),language:`bash`},nodejs:{code:r(ke),language:`javascript`},python:{code:r(Ae),language:`python`},go:{code:r(je),language:`go`},php:{code:r(Me),language:`php`},smtp:{code:r(Ne),language:`text`}})),J=f(()=>r(Pe)[r(q)]);v(async()=>{await Y()});async function Y(){d(F,!0),d(L,``);try{let e=await te();d(B,e.keys||[],!0)}catch{console.log(`No API keys yet`)}finally{d(F,!1)}}async function Fe(){if(!r(H).trim()){d(L,`Please enter a name for the API key`);return}d(I,!0),d(L,``);try{let e=await ne({name:r(H).trim()});d(U,e.key||``,!0),await Y(),d(R,`API key created successfully. Copy it now - you won't be able to see it again!`)}catch(e){d(L,e.message||`Failed to create API key`,!0)}finally{d(I,!1)}}function Ie(e){d(G,e,!0),d(W,!0)}async function Le(){d(K,!0);try{await re({},r(G)),await Y(),d(R,`API key revoked`)}catch(e){d(L,e.message||`Failed to revoke API key`,!0)}finally{d(K,!1)}}function X(e){navigator.clipboard.writeText(e),d(z,e,!0),setTimeout(()=>d(z,``),2e3)}function Z(){d(V,!1),d(H,``),d(U,``)}var Q=Ee();a(`1kyawqp`,e=>{l(()=>{ee.title=`API - Settings`})});var $=m(Q),Re=y($),ze=e=>{D(e,{type:`error`,title:`Error`,onclose:()=>d(L,``),children:(e,t)=>{var n=j(),a=u(n,!0);s(()=>i(a,r(L))),c(e,n)},$$slots:{default:!0}})};n(Re,e=>{r(L)&&e(ze)});var Be=_(Re,2),Ve=e=>{D(e,{type:`success`,title:`Success`,onclose:()=>d(R,``),children:(e,t)=>{var n=j(),a=u(n,!0);s(()=>i(a,r(R))),c(e,n)},$$slots:{default:!0}})};n(Be,e=>{r(R)&&e(Ve)});var He=_(Be,2);O(He,{children:(e,t)=>{var a=ve(),l=m(a),f=_(y(l),2);T(f,{type:`primary`,size:`sm`,onclick:()=>d(V,!0),children:(e,t)=>{var n=pe(),r=m(n);de(r,{class:`mr-1.5 h-4 w-4`}),x(),c(e,n)},$$slots:{default:!0}}),S(l);var p=_(l,2),h=e=>{var t=me(),n=y(t);oe(n,{}),S(t),c(e,t)},g=e=>{var t=he(),n=y(t);ue(n,{class:`mx-auto h-12 w-12 opacity-50`}),x(4),S(t),c(e,t)},v=e=>{var t=_e(),n=y(t),a=_(y(n));o(a,21,()=>r(B),e=>e.id,(e,t)=>{var n=ge(),a=y(n),o=u(a,!0),l=_(a),d=u(l),f=_(l),p=u(f,!0),m=_(f),h=u(m,!0),g=_(m),v=y(g);T(v,{type:`danger`,size:`icon`,onclick:()=>Ie(r(t).id),children:(e,t)=>{fe(e,{class:`h-4 w-4`})},$$slots:{default:!0}}),S(g),S(n),s((e,n)=>{i(o,r(t).name),i(d,`${r(t).key_prefix??``}...`),i(p,e),i(h,n)},[()=>A(r(t).last_used)||`Never`,()=>A(r(t).created_at)||`Never`]),c(e,n)}),S(a),S(n),S(t),c(e,t)};n(p,e=>{r(F)?e(h):r(B).length===0?e(g,1):e(v,-1)}),c(e,a)},$$slots:{default:!0}});var Ue=_(He,2);O(Ue,{children:(e,t)=>{var i=ye(),a=m(i),o=y(a);se(o,{class:`h-5 w-5 text-primary`}),x(2),S(a);var s=_(a,2),l=y(s),u=_(y(l),2),d=y(u);E(d,{type:`text`,get value(){return r(P)},readonly:!0,class:`font-mono text-sm`});var f=_(d,2);T(f,{type:`secondary`,onclick:()=>X(r(P)),children:(e,t)=>{var i=C(),a=m(i),o=e=>{k(e,{class:`h-4 w-4 text-green-500`})},s=e=>{w(e,{class:`h-4 w-4`})};n(a,e=>{r(z)===r(P)?e(o):e(s,-1)}),c(e,i)},$$slots:{default:!0}}),S(u),S(l),x(2),S(s),c(e,i)},$$slots:{default:!0}});var We=_(Ue,2);O(We,{children:(e,t)=>{var n=be(),i=_(m(n),2),a=y(i);ae(a,{get tabs(){return De},variant:`pills`,get activeTab(){return r(q)},set activeTab(e){d(q,e,!0)}}),S(i);var o=_(i,2);le(o,{get code(){return r(J).code},get language(){return r(J).language}}),c(e,n)},$$slots:{default:!0}}),S($);var Ge=_($,2);ie(Ge,{title:`Create API Key`,onclose:Z,get show(){return r(V)},set show(e){d(V,e,!0)},footer:e=>{var a=Se(),o=y(a),l=e=>{T(e,{type:`primary`,onclick:Z,children:(e,n)=>{x();var r=t(`Done`);c(e,r)},$$slots:{default:!0}})},u=e=>{var n=xe(),a=m(n);T(a,{type:`secondary`,onclick:Z,children:(e,n)=>{x();var r=t(`Cancel`);c(e,r)},$$slots:{default:!0}});var o=_(a,2);{let e=f(()=>r(I)||!r(H).trim());T(o,{type:`primary`,onclick:Fe,get disabled(){return r(e)},children:(e,n)=>{x();var a=t();s(()=>i(a,r(I)?`Creating...`:`Create Key`)),c(e,a)},$$slots:{default:!0}})}c(e,n)};n(o,e=>{r(U)?e(l):e(u,-1)}),S(a),c(e,a)},children:(e,t)=>{var i=C(),a=m(i),o=e=>{var t=we(),i=y(t);D(i,{type:`warning`,title:`Save your API key`,children:(e,t)=>{var n=Ce();c(e,n)},$$slots:{default:!0}});var a=_(i,2),o=_(y(a),2),s=y(o);E(s,{type:`text`,get value(){return r(U)},readonly:!0,class:`font-mono text-sm`});var l=_(s,2);T(l,{type:`secondary`,onclick:()=>X(r(U)),children:(e,t)=>{var i=C(),a=m(i),o=e=>{k(e,{class:`h-4 w-4 text-green-500`})},s=e=>{w(e,{class:`h-4 w-4`})};n(a,e=>{r(z)===r(U)?e(o):e(s,-1)}),c(e,i)},$$slots:{default:!0}}),S(o),S(a),S(t),c(e,t)},s=e=>{var t=Te(),n=y(t),i=_(y(n),2);E(i,{type:`text`,id:`key-name`,placeholder:`e.g., Production API, Development`,get value(){return r(H)},set value(e){d(H,e,!0)}}),x(2),S(n),S(t),c(e,t)};n(a,e=>{r(U)?e(o):e(s,-1)}),c(e,i)},$$slots:{footer:!0,default:!0}});var Ke=_(Ge,2);{let e=f(()=>r(K)?`Revoking...`:`Revoke`);ce(Ke,{title:`Revoke API Key`,description:`Are you sure you want to revoke this API key? This cannot be undone.`,get actionLabel(){return r(e)},actionType:`danger`,onAction:Le,onCancel:()=>d(W,!1),get open(){return r(W)},set open(e){d(W,e,!0)}})}c(e,Q),g()}export{M as component};