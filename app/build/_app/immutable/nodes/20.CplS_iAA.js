import{$ as e,B as t,L as n,O as r,P as i,Q as a,R as o,U as s,V as c,at as l,bt as u,ct as d,ft as f,gt as p,ht as m,it as h,lt as g,nt as ee,ot as _,q as v,r as te,st as y,ut as b,yt as x,z as S}from"../chunks/BKXUg6nK.js";import"../chunks/xihTtKlq.js";import{Fn as ne,R as re,in as ie}from"../chunks/j-DxK_jk.js";import{C as ae,H as C,L as w,O as oe,P as T,ct as E,it as se,o as ce,ot as D,rt as O,w as le,z as ue}from"../chunks/Do6Ocdwx.js";import{t as de}from"../chunks/DAYO7flF.js";import{t as fe}from"../chunks/TdWo_hAD.js";import{t as pe}from"../chunks/D1g8G3sD.js";import{t as k}from"../chunks/Dn9WcqgK.js";var A=c(`<p> </p>`),me=c(`<!> Create Key`,1),he=c(`<div class="flex justify-center py-8"><!></div>`),ge=c(`<div class="mt-6 text-center py-8 text-text-muted"><!> <p class="mt-2">No MCP keys yet</p> <p class="text-sm">Create one to connect an AI assistant over MCP</p></div>`),_e=c(`<tr class="border-b border-border/50"><td class="py-3 font-medium"> </td><td class="py-3 font-mono text-text-muted"> </td><td class="py-3 text-text-muted"> </td><td class="py-3 text-text-muted"> </td><td class="py-3 text-right"><!></td></tr>`),ve=c(`<div class="mt-6 overflow-x-auto"><table class="w-full text-sm"><thead><tr class="border-b border-border"><th class="text-left py-2 font-medium text-text-muted">Name</th><th class="text-left py-2 font-medium text-text-muted">Key</th><th class="text-left py-2 font-medium text-text-muted">Last Used</th><th class="text-left py-2 font-medium text-text-muted">Created</th><th class="text-right py-2 font-medium text-text-muted">Actions</th></tr></thead><tbody></tbody></table></div>`),ye=c(`<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between"><div><h2 class="text-lg font-medium text-text mb-1">MCP keys</h2> <p class="text-sm text-text-muted">For AI assistants such as Claude that manage Outlet through MCP. They don't work for sending email: apps use their brand's API key, below.</p></div> <!></div> <!>`,1),be=c(`<div class="flex items-center gap-3 mb-4"><!> <div><h2 class="text-lg font-medium text-text">Sending API and SMTP</h2> <p class="text-sm text-text-muted">How your apps send email. Each brand has its own API key, shown in that brand's settings.</p></div></div> <div class="space-y-4"><div><label class="form-label">Base URL</label> <div class="flex gap-2"><!> <!></div></div> <div><label class="form-label">Authentication</label> <p class="text-sm text-text-muted mb-2">Send the brand's API key in the X-API-Key header:</p> <pre class="bg-surface-tertiary p-3 rounded text-sm overflow-x-auto font-mono">X-API-Key: YOUR_BRAND_API_KEY</pre></div></div>`,1),xe=c(`<h2 class="text-lg font-medium text-text mb-4">Code Examples</h2> <div class="mb-4"><!></div> <!>`,1),Se=c(`<!> <!>`,1),Ce=c(`<div class="flex justify-end gap-3"><!></div>`),we=c(`<p>This is the only time you'll see this key. Copy it now and store it securely.</p>`),Te=c(`<div class="space-y-4"><!> <div><label class="form-label">API Key</label> <div class="flex gap-2"><!> <!></div></div></div>`),Ee=c(`<div class="space-y-4"><div><label for="key-name" class="form-label">Key Name</label> <!> <p class="mt-1 text-xs text-text-muted">A descriptive name to identify this key</p></div></div>`),De=c(`<div class="space-y-6"><!> <!> <!> <!> <!></div> <!> <!>`,1);function j(c,j){p(j,!0);let M=d(window.location.origin.replace(`:5173`,`:8888`)),N=f(()=>`${M}/sdk/v1`),P=b(!0),F=b(!1),I=b(``),L=b(``),R=b(``),z=b(d([])),B=b(!1),V=b(``),H=b(``),U=b(!1),W=b(``),G=b(!1),Oe=[{id:`curl`,label:`cURL`},{id:`nodejs`,label:`Node.js`},{id:`python`,label:`Python`},{id:`go`,label:`Go`},{id:`php`,label:`PHP`},{id:`smtp`,label:`SMTP`}],K=b(`curl`),ke=f(()=>`# Send a transactional template
curl -X POST "${v(N)}/emails" \\
  -H "X-API-Key: YOUR_BRAND_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "to": "customer@example.com",
    "template_slug": "password-reset",
    "variables": { "name": "Ana", "reset_url": "https://app.example.com/reset/abc" }
  }'

# Or send without a template
curl -X POST "${v(N)}/emails" \\
  -H "X-API-Key: YOUR_BRAND_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "to": "customer@example.com",
    "subject": "Your receipt",
    "body": "<p>Thanks for your order.</p>",
    "text_body": "Thanks for your order."
  }'

# Check delivery
curl "${v(N)}/emails/MESSAGE_ID" -H "X-API-Key: YOUR_BRAND_API_KEY"`),Ae=f(()=>`const res = await fetch('${v(N)}/emails', {
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
console.log(result.message_id);`),je=f(()=>`import os, requests

res = requests.post(
    '${v(N)}/emails',
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
print(result['message_id'])`),Me=f(()=>`payload, _ := json.Marshal(map[string]any{
    "to":            "customer@example.com",
    "template_slug": "password-reset",
    "variables":     map[string]string{"name": "Ana", "reset_url": "https://app.example.com/reset/abc"},
})
req, _ := http.NewRequest("POST", "${v(N)}/emails", bytes.NewReader(payload))
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
json.NewDecoder(resp.Body).Decode(&result)`),Ne=f(()=>`<?php
$ch = curl_init('${v(N)}/emails');
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
$result = json_decode(curl_exec($ch), true);`),Pe=f(()=>`Host:      ${M.replace(/^https?:\/\//,``).replace(/:\d+$/,``)}
Port:      587 (STARTTLS)
Username:  your brand slug
Password:  your brand API key

Optional headers:
  X-Outlet-Tags: welcome,onboarding
  X-Outlet-Meta-OrderID: 12345`),Fe=f(()=>({curl:{code:v(ke),language:`bash`},nodejs:{code:v(Ae),language:`javascript`},python:{code:v(je),language:`python`},go:{code:v(Me),language:`go`},php:{code:v(Ne),language:`php`},smtp:{code:v(Pe),language:`text`}})),q=f(()=>v(Fe)[v(K)]);te(async()=>{await J()});async function J(){g(P,!0),g(I,``);try{let e=await ie();g(z,e.keys||[],!0)}catch{console.log(`No API keys yet`)}finally{g(P,!1)}}async function Ie(){if(!v(V).trim()){g(I,`Please enter a name for the API key`);return}g(F,!0),g(I,``);try{let e=await re({name:v(V).trim()});g(H,e.key||``,!0),await J(),g(L,`API key created successfully. Copy it now - you won't be able to see it again!`)}catch(e){g(I,e.message||`Failed to create API key`,!0)}finally{g(F,!1)}}function Le(e){g(W,e,!0),g(U,!0)}async function Re(){g(G,!0);try{await ne({},v(W)),await J(),g(L,`API key revoked`)}catch(e){g(I,e.message||`Failed to revoke API key`,!0)}finally{g(G,!1)}}function Y(e){navigator.clipboard.writeText(e),g(R,e,!0),setTimeout(()=>g(R,``),2e3)}function X(){g(B,!1),g(V,``),g(H,``)}var Z=De();r(`1kyawqp`,e=>{a(()=>{ee.title=`API - Settings`})});var Q=l(Z),ze=h(Q),Be=t=>{C(t,{type:`error`,title:`Error`,onclose:()=>g(I,``),children:(t,n)=>{var r=A(),i=_(r,!0);e(()=>o(i,v(I))),S(t,r)},$$slots:{default:!0}})};n(ze,e=>{v(I)&&e(Be)});var Ve=y(ze,2),He=t=>{C(t,{type:`success`,title:`Success`,onclose:()=>g(L,``),children:(t,n)=>{var r=A(),i=_(r,!0);e(()=>o(i,v(L))),S(t,r)},$$slots:{default:!0}})};n(Ve,e=>{v(L)&&e(He)});var Ue=y(Ve,2);E(Ue,{children:(t,r)=>{var a=ye(),s=l(a),c=y(h(s),2);w(c,{type:`primary`,size:`sm`,onclick:()=>g(B,!0),children:(e,t)=>{var n=me(),r=l(n);fe(r,{class:`mr-1.5 h-4 w-4`}),x(),S(e,n)},$$slots:{default:!0}}),u(s);var d=y(s,2),f=e=>{var t=he(),n=h(t);ue(n,{}),u(t),S(e,t)},p=e=>{var t=ge(),n=h(t);de(n,{class:`mx-auto h-12 w-12 opacity-50`}),x(4),u(t),S(e,t)},m=t=>{var n=ve(),r=h(n),a=y(h(r));i(a,21,()=>v(z),e=>e.id,(t,n)=>{var r=_e(),i=h(r),a=_(i,!0),s=y(i),c=_(s),l=y(s),d=_(l,!0),f=y(l),p=_(f,!0),m=y(f),g=h(m);w(g,{type:`danger`,size:`icon`,onclick:()=>Le(v(n).id),children:(e,t)=>{pe(e,{class:`h-4 w-4`})},$$slots:{default:!0}}),u(m),u(r),e((e,t)=>{o(a,v(n).name),o(c,`${v(n).key_prefix??``}...`),o(d,e),o(p,t)},[()=>k(v(n).last_used)||`Never`,()=>k(v(n).created_at)||`Never`]),S(t,r)}),u(a),u(r),u(n),S(t,n)};n(d,e=>{v(P)?e(f):v(z).length===0?e(p,1):e(m,-1)}),S(t,a)},$$slots:{default:!0}});var We=y(Ue,2);E(We,{children:(e,r)=>{var i=be(),a=l(i),o=h(a);se(o,{class:`h-5 w-5 text-primary`}),x(2),u(a);var s=y(a,2),c=h(s),d=y(h(c),2),f=h(d);T(f,{type:`text`,get value(){return v(N)},readonly:!0,class:`font-mono text-sm`});var p=y(f,2);w(p,{type:`secondary`,onclick:()=>Y(v(N)),children:(e,r)=>{var i=t(),a=l(i),o=e=>{D(e,{class:`h-4 w-4 text-success`})},s=e=>{O(e,{class:`h-4 w-4`})};n(a,e=>{v(R)===v(N)?e(o):e(s,-1)}),S(e,i)},$$slots:{default:!0}}),u(d),u(c),x(2),u(s),S(e,i)},$$slots:{default:!0}});var Ge=y(We,2);E(Ge,{children:(e,t)=>{var n=xe(),r=y(l(n),2),i=h(r);oe(i,{get tabs(){return Oe},variant:`pills`,get activeTab(){return v(K)},set activeTab(e){g(K,e,!0)}}),u(r);var a=y(r,2);le(a,{get code(){return v(q).code},get language(){return v(q).language}}),S(e,n)},$$slots:{default:!0}}),u(Q);var $=y(Q,2);ae($,{title:`Create API Key`,onclose:X,get show(){return v(B)},set show(e){g(B,e,!0)},footer:t=>{var r=Ce(),i=h(r),a=e=>{w(e,{type:`primary`,onclick:X,children:(e,t)=>{x();var n=s(`Done`);S(e,n)},$$slots:{default:!0}})},c=t=>{var n=Se(),r=l(n);w(r,{type:`secondary`,onclick:X,children:(e,t)=>{x();var n=s(`Cancel`);S(e,n)},$$slots:{default:!0}});var i=y(r,2);{let t=f(()=>v(F)||!v(V).trim());w(i,{type:`primary`,onclick:Ie,get disabled(){return v(t)},children:(t,n)=>{x();var r=s();e(()=>o(r,v(F)?`Creating...`:`Create Key`)),S(t,r)},$$slots:{default:!0}})}S(t,n)};n(i,e=>{v(H)?e(a):e(c,-1)}),u(r),S(t,r)},children:(e,r)=>{var i=t(),a=l(i),o=e=>{var r=Te(),i=h(r);C(i,{type:`warning`,title:`Save your API key`,children:(e,t)=>{var n=we();S(e,n)},$$slots:{default:!0}});var a=y(i,2),o=y(h(a),2),s=h(o);T(s,{type:`text`,get value(){return v(H)},readonly:!0,class:`font-mono text-sm`});var c=y(s,2);w(c,{type:`secondary`,onclick:()=>Y(v(H)),children:(e,r)=>{var i=t(),a=l(i),o=e=>{D(e,{class:`h-4 w-4 text-success`})},s=e=>{O(e,{class:`h-4 w-4`})};n(a,e=>{v(R)===v(H)?e(o):e(s,-1)}),S(e,i)},$$slots:{default:!0}}),u(o),u(a),u(r),S(e,r)},s=e=>{var t=Ee(),n=h(t),r=y(h(n),2);T(r,{type:`text`,id:`key-name`,placeholder:`e.g., Production API, Development`,get value(){return v(V)},set value(e){g(V,e,!0)}}),x(2),u(n),u(t),S(e,t)};n(a,e=>{v(H)?e(o):e(s,-1)}),S(e,i)},$$slots:{footer:!0,default:!0}});var Ke=y($,2);{let e=f(()=>v(G)?`Revoking...`:`Revoke`);ce(Ke,{title:`Revoke API Key`,description:`Are you sure you want to revoke this API key? This cannot be undone.`,get actionLabel(){return v(e)},actionType:`danger`,onAction:Re,onCancel:()=>g(U,!1),get open(){return v(U)},set open(e){g(U,e,!0)}})}S(c,Z),m()}export{j as component};