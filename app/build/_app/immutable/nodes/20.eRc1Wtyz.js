import{B as e,H as t,I as n,K as r,L as i,O as a,P as o,Q as s,R as c,Z as l,at as u,ct as d,dt as f,ht as p,it as m,lt as h,mt as g,ot as _,r as v,rt as y,st as b,tt as ee,vt as x,yt as S,z as C}from"../chunks/C_OAeTtO.js";import"../chunks/xihTtKlq.js";import{Cn as te,I as ne,Jt as re}from"../chunks/DadxOX5y.js";import{C as ie,I as w,N as T,O as ae,R as oe,V as E,at as D,nt as O,o as se,rt as ce,st as k,w as le}from"../chunks/DPDoKt-b.js";import{t as ue}from"../chunks/C_9ltpny.js";import{t as de}from"../chunks/DSYgN34R.js";import{t as fe}from"../chunks/OtCddLiu.js";import{t as A}from"../chunks/Dn9WcqgK.js";var pe=e(`<p> </p>`),me=e(`<!> Create Key`,1),he=e(`<div class="flex justify-center py-8"><!></div>`),ge=e(`<div class="mt-6 text-center py-8 text-text-muted"><!> <p class="mt-2">No MCP keys yet</p> <p class="text-sm">Create one to connect an AI assistant over MCP</p></div>`),_e=e(`<tr class="border-b border-border/50"><td class="py-3 font-medium"> </td><td class="py-3 font-mono text-text-muted"> </td><td class="py-3 text-text-muted"> </td><td class="py-3 text-text-muted"> </td><td class="py-3 text-right"><!></td></tr>`),ve=e(`<div class="mt-6 overflow-x-auto"><table class="w-full text-sm"><thead><tr class="border-b border-border"><th class="text-left py-2 font-medium text-text-muted">Name</th><th class="text-left py-2 font-medium text-text-muted">Key</th><th class="text-left py-2 font-medium text-text-muted">Last Used</th><th class="text-left py-2 font-medium text-text-muted">Created</th><th class="text-right py-2 font-medium text-text-muted">Actions</th></tr></thead><tbody></tbody></table></div>`),ye=e(`<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between"><div><h2 class="text-lg font-medium text-text mb-1">MCP keys</h2> <p class="text-sm text-text-muted">For AI assistants such as Claude that manage Outlet through MCP. They don't work for sending email: apps use their brand's API key, below.</p></div> <!></div> <!>`,1),be=e(`<div class="flex items-center gap-3 mb-4"><!> <div><h2 class="text-lg font-medium text-text">Sending API and SMTP</h2> <p class="text-sm text-text-muted">How your apps send email. Each brand has its own API key, shown in that brand's settings.</p></div></div> <div class="space-y-4"><div><label class="form-label">Base URL</label> <div class="flex gap-2"><!> <!></div></div> <div><label class="form-label">Authentication</label> <p class="text-sm text-text-muted mb-2">Send the brand's API key in the X-API-Key header:</p> <pre class="bg-surface-tertiary p-3 rounded text-sm overflow-x-auto font-mono">X-API-Key: YOUR_BRAND_API_KEY</pre></div></div>`,1),xe=e(`<h2 class="text-lg font-medium text-text mb-4">Code Examples</h2> <div class="mb-4"><!></div> <!>`,1),Se=e(`<!> <!>`,1),Ce=e(`<div class="flex justify-end gap-3"><!></div>`),we=e(`<p>This is the only time you'll see this key. Copy it now and store it securely.</p>`),Te=e(`<div class="space-y-4"><!> <div><label class="form-label">API Key</label> <div class="flex gap-2"><!> <!></div></div></div>`),Ee=e(`<div class="space-y-4"><div><label for="key-name" class="form-label">Key Name</label> <!> <p class="mt-1 text-xs text-text-muted">A descriptive name to identify this key</p></div></div>`),De=e(`<div class="space-y-6"><!> <!> <!> <!> <!></div> <!> <!>`,1);function j(e,j){p(j,!0);let M=b(window.location.origin.replace(`:5173`,`:8888`)),N=f(()=>`${M}/sdk/v1`),P=h(!0),F=h(!1),I=h(``),L=h(``),R=h(``),z=h(b([])),B=h(!1),V=h(``),H=h(``),U=h(!1),W=h(``),G=h(!1),Oe=[{id:`curl`,label:`cURL`},{id:`nodejs`,label:`Node.js`},{id:`python`,label:`Python`},{id:`go`,label:`Go`},{id:`php`,label:`PHP`},{id:`smtp`,label:`SMTP`}],K=h(`curl`),ke=f(()=>`# Send a transactional template
curl -X POST "${r(N)}/emails" \\
  -H "X-API-Key: YOUR_BRAND_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "to": "customer@example.com",
    "template_slug": "password-reset",
    "variables": { "name": "Ana", "reset_url": "https://app.example.com/reset/abc" }
  }'

# Or send without a template
curl -X POST "${r(N)}/emails" \\
  -H "X-API-Key: YOUR_BRAND_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "to": "customer@example.com",
    "subject": "Your receipt",
    "body": "<p>Thanks for your order.</p>",
    "text_body": "Thanks for your order."
  }'

# Check delivery
curl "${r(N)}/emails/MESSAGE_ID" -H "X-API-Key: YOUR_BRAND_API_KEY"`),Ae=f(()=>`const res = await fetch('${r(N)}/emails', {
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
    '${r(N)}/emails',
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
req, _ := http.NewRequest("POST", "${r(N)}/emails", bytes.NewReader(payload))
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
$ch = curl_init('${r(N)}/emails');
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
  X-Outlet-Meta-OrderID: 12345`),Fe=f(()=>({curl:{code:r(ke),language:`bash`},nodejs:{code:r(Ae),language:`javascript`},python:{code:r(je),language:`python`},go:{code:r(Me),language:`go`},php:{code:r(Ne),language:`php`},smtp:{code:r(Pe),language:`text`}})),q=f(()=>r(Fe)[r(K)]);v(async()=>{await J()});async function J(){d(P,!0),d(I,``);try{let e=await re();d(z,e.keys||[],!0)}catch{console.log(`No API keys yet`)}finally{d(P,!1)}}async function Ie(){if(!r(V).trim()){d(I,`Please enter a name for the API key`);return}d(F,!0),d(I,``);try{let e=await ne({name:r(V).trim()});d(H,e.key||``,!0),await J(),d(L,`API key created successfully. Copy it now - you won't be able to see it again!`)}catch(e){d(I,e.message||`Failed to create API key`,!0)}finally{d(F,!1)}}function Le(e){d(W,e,!0),d(U,!0)}async function Re(){d(G,!0);try{await te({},r(W)),await J(),d(L,`API key revoked`)}catch(e){d(I,e.message||`Failed to revoke API key`,!0)}finally{d(G,!1)}}function Y(e){navigator.clipboard.writeText(e),d(R,e,!0),setTimeout(()=>d(R,``),2e3)}function X(){d(B,!1),d(V,``),d(H,``)}var ze=De();a(`1kyawqp`,e=>{l(()=>{ee.title=`API - Settings`})});var Z=m(ze),Be=y(Z),Ve=e=>{E(e,{type:`error`,title:`Error`,onclose:()=>d(I,``),children:(e,t)=>{var n=pe(),a=u(n,!0);s(()=>i(a,r(I))),c(e,n)},$$slots:{default:!0}})};n(Be,e=>{r(I)&&e(Ve)});var Q=_(Be,2),He=e=>{E(e,{type:`success`,title:`Success`,onclose:()=>d(L,``),children:(e,t)=>{var n=pe(),a=u(n,!0);s(()=>i(a,r(L))),c(e,n)},$$slots:{default:!0}})};n(Q,e=>{r(L)&&e(He)});var Ue=_(Q,2);k(Ue,{children:(e,t)=>{var a=ye(),l=m(a),f=_(y(l),2);w(f,{type:`primary`,size:`sm`,onclick:()=>d(B,!0),children:(e,t)=>{var n=me(),r=m(n);de(r,{class:`mr-1.5 h-4 w-4`}),x(),c(e,n)},$$slots:{default:!0}}),S(l);var p=_(l,2),h=e=>{var t=he(),n=y(t);oe(n,{}),S(t),c(e,t)},g=e=>{var t=ge(),n=y(t);ue(n,{class:`mx-auto h-12 w-12 opacity-50`}),x(4),S(t),c(e,t)},v=e=>{var t=ve(),n=y(t),a=_(y(n));o(a,21,()=>r(z),e=>e.id,(e,t)=>{var n=_e(),a=y(n),o=u(a,!0),l=_(a),d=u(l),f=_(l),p=u(f,!0),m=_(f),h=u(m,!0),g=_(m),v=y(g);w(v,{type:`danger`,size:`icon`,onclick:()=>Le(r(t).id),children:(e,t)=>{fe(e,{class:`h-4 w-4`})},$$slots:{default:!0}}),S(g),S(n),s((e,n)=>{i(o,r(t).name),i(d,`${r(t).key_prefix??``}...`),i(p,e),i(h,n)},[()=>A(r(t).last_used)||`Never`,()=>A(r(t).created_at)||`Never`]),c(e,n)}),S(a),S(n),S(t),c(e,t)};n(p,e=>{r(P)?e(h):r(z).length===0?e(g,1):e(v,-1)}),c(e,a)},$$slots:{default:!0}});var We=_(Ue,2);k(We,{children:(e,t)=>{var i=be(),a=m(i),o=y(a);ce(o,{class:`h-5 w-5 text-primary`}),x(2),S(a);var s=_(a,2),l=y(s),u=_(y(l),2),d=y(u);T(d,{type:`text`,get value(){return r(N)},readonly:!0,class:`font-mono text-sm`});var f=_(d,2);w(f,{type:`secondary`,onclick:()=>Y(r(N)),children:(e,t)=>{var i=C(),a=m(i),o=e=>{D(e,{class:`h-4 w-4 text-green-500`})},s=e=>{O(e,{class:`h-4 w-4`})};n(a,e=>{r(R)===r(N)?e(o):e(s,-1)}),c(e,i)},$$slots:{default:!0}}),S(u),S(l),x(2),S(s),c(e,i)},$$slots:{default:!0}});var Ge=_(We,2);k(Ge,{children:(e,t)=>{var n=xe(),i=_(m(n),2),a=y(i);ae(a,{get tabs(){return Oe},variant:`pills`,get activeTab(){return r(K)},set activeTab(e){d(K,e,!0)}}),S(i);var o=_(i,2);le(o,{get code(){return r(q).code},get language(){return r(q).language}}),c(e,n)},$$slots:{default:!0}}),S(Z);var $=_(Z,2);ie($,{title:`Create API Key`,onclose:X,get show(){return r(B)},set show(e){d(B,e,!0)},footer:e=>{var a=Ce(),o=y(a),l=e=>{w(e,{type:`primary`,onclick:X,children:(e,n)=>{x();var r=t(`Done`);c(e,r)},$$slots:{default:!0}})},u=e=>{var n=Se(),a=m(n);w(a,{type:`secondary`,onclick:X,children:(e,n)=>{x();var r=t(`Cancel`);c(e,r)},$$slots:{default:!0}});var o=_(a,2);{let e=f(()=>r(F)||!r(V).trim());w(o,{type:`primary`,onclick:Ie,get disabled(){return r(e)},children:(e,n)=>{x();var a=t();s(()=>i(a,r(F)?`Creating...`:`Create Key`)),c(e,a)},$$slots:{default:!0}})}c(e,n)};n(o,e=>{r(H)?e(l):e(u,-1)}),S(a),c(e,a)},children:(e,t)=>{var i=C(),a=m(i),o=e=>{var t=Te(),i=y(t);E(i,{type:`warning`,title:`Save your API key`,children:(e,t)=>{var n=we();c(e,n)},$$slots:{default:!0}});var a=_(i,2),o=_(y(a),2),s=y(o);T(s,{type:`text`,get value(){return r(H)},readonly:!0,class:`font-mono text-sm`});var l=_(s,2);w(l,{type:`secondary`,onclick:()=>Y(r(H)),children:(e,t)=>{var i=C(),a=m(i),o=e=>{D(e,{class:`h-4 w-4 text-green-500`})},s=e=>{O(e,{class:`h-4 w-4`})};n(a,e=>{r(R)===r(H)?e(o):e(s,-1)}),c(e,i)},$$slots:{default:!0}}),S(o),S(a),S(t),c(e,t)},s=e=>{var t=Ee(),n=y(t),i=_(y(n),2);T(i,{type:`text`,id:`key-name`,placeholder:`e.g., Production API, Development`,get value(){return r(V)},set value(e){d(V,e,!0)}}),x(2),S(n),S(t),c(e,t)};n(a,e=>{r(H)?e(o):e(s,-1)}),c(e,i)},$$slots:{footer:!0,default:!0}});var Ke=_($,2);{let e=f(()=>r(G)?`Revoking...`:`Revoke`);se(Ke,{title:`Revoke API Key`,description:`Are you sure you want to revoke this API key? This cannot be undone.`,get actionLabel(){return r(e)},actionType:`danger`,onAction:Re,onCancel:()=>d(U,!1),get open(){return r(U)},set open(e){d(U,e,!0)}})}c(e,ze),g()}export{j as component};