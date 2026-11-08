import"../chunks/DsnmJJEf.js";import{a as qe}from"../chunks/Bfof7OBB.js";import{p as Fe,a as xe,s as x,f as C,d as Ge,e as ze,b as a,h as n,i as e,c as r,j as P,g as Je,n as K,r as s,t as B}from"../chunks/BtjfW9_B.js";import{s as j}from"../chunks/DEsepwd4.js";import{i as R}from"../chunks/Cn_LYL0x.js";import{e as Ve}from"../chunks/B8beEWxR.js";import{a as i,c as q,f as u,t as se}from"../chunks/DnSY09wi.js";import{h as Qe}from"../chunks/DvNfZKU3.js";import{C as re,M as We,e as Ze,B as O,f as et,I as oe,g as tt,h as at,b as be,i as Pe,A as le}from"../chunks/DTNgJKyn.js";import{L as st}from"../chunks/OUNBezzj.js";import{b as rt,d as ot,e as lt}from"../chunks/DfxTvd9F.js";import{P as nt}from"../chunks/XV4ArKKN.js";import{K as it}from"../chunks/kfxxp4Fw.js";import{T as dt}from"../chunks/CRTodBPi.js";var ct=u("<p> </p>"),pt=u("<p> </p>"),ut=u("<!> Create Key",1),mt=u('<div class="flex justify-center py-8"><!></div>'),vt=u('<div class="mt-6 text-center py-8 text-text-muted"><!> <p class="mt-2">No MCP keys yet</p> <p class="text-sm">Create one to connect an AI assistant over MCP</p></div>'),yt=u('<tr class="border-b border-border/50"><td class="py-3 font-medium"> </td><td class="py-3 font-mono text-text-muted"> </td><td class="py-3 text-text-muted"> </td><td class="py-3 text-text-muted"> </td><td class="py-3 text-right"><!></td></tr>'),_t=u('<div class="mt-6 overflow-x-auto"><table class="w-full text-sm"><thead><tr class="border-b border-border"><th class="text-left py-2 font-medium text-text-muted">Name</th><th class="text-left py-2 font-medium text-text-muted">Key</th><th class="text-left py-2 font-medium text-text-muted">Last Used</th><th class="text-left py-2 font-medium text-text-muted">Created</th><th class="text-right py-2 font-medium text-text-muted">Actions</th></tr></thead><tbody></tbody></table></div>'),ft=u(`<div class="flex items-center justify-between"><div><h2 class="text-lg font-medium text-text mb-1">MCP keys</h2> <p class="text-sm text-text-muted">For AI assistants such as Claude that manage Outlet through MCP. They don't work for sending email: apps use their brand's API key, below.</p></div> <!></div> <!>`,1),ht=u(`<div class="flex items-center gap-3 mb-4"><!> <div><h2 class="text-lg font-medium text-text">Sending API and SMTP</h2> <p class="text-sm text-text-muted">How your apps send email. Each brand has its own API key, shown in that brand's settings.</p></div></div> <div class="space-y-4"><div><label class="form-label">Base URL</label> <div class="flex gap-2"><!> <!></div></div> <div><label class="form-label">Authentication</label> <p class="text-sm text-text-muted mb-2">Send the brand's API key in the X-API-Key header:</p> <pre class="bg-surface-tertiary p-3 rounded text-sm overflow-x-auto font-mono">X-API-Key: YOUR_BRAND_API_KEY</pre></div></div>`,1),gt=u('<h2 class="text-lg font-medium text-text mb-4">Code Examples</h2> <div class="mb-4"><!></div> <!>',1),xt=u("<!> <!>",1),bt=u('<div class="flex justify-end gap-3"><!></div>'),Pt=u("<p>This is the only time you'll see this key. Copy it now and store it securely.</p>"),$t=u('<div class="space-y-4"><!> <div><label class="form-label">API Key</label> <div class="flex gap-2"><!> <!></div></div></div>'),At=u('<div class="space-y-4"><div><label for="key-name" class="form-label">Key Name</label> <!> <p class="mt-1 text-xs text-text-muted">A descriptive name to identify this key</p></div></div>'),wt=u('<div class="space-y-6"><!> <!> <!> <!> <!></div> <!> <!>',1);function Nt($e,Ae){Fe(Ae,!0);let ne=xe(window.location.origin.replace(":5173",":8888")),$=P(()=>`${ne}/sdk/v1`),F=x(!0),N=x(!1),I=x(""),M=x(""),X=x(""),G=x(xe([])),Y=x(!1),D=x(""),E=x(""),H=x(!1),ie=x(""),z=x(!1);const we=[{id:"curl",label:"cURL"},{id:"nodejs",label:"Node.js"},{id:"python",label:"Python"},{id:"go",label:"Go"},{id:"php",label:"PHP"},{id:"smtp",label:"SMTP"}];let J=x("curl");const Te=P(()=>`# Send a transactional template
curl -X POST "${e($)}/emails" \\
  -H "X-API-Key: YOUR_BRAND_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "to": "customer@example.com",
    "template_slug": "password-reset",
    "variables": { "name": "Ana", "reset_url": "https://app.example.com/reset/abc" }
  }'

# Or send without a template
curl -X POST "${e($)}/emails" \\
  -H "X-API-Key: YOUR_BRAND_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "to": "customer@example.com",
    "subject": "Your receipt",
    "body": "<p>Thanks for your order.</p>",
    "text_body": "Thanks for your order."
  }'

# Check delivery
curl "${e($)}/emails/MESSAGE_ID" -H "X-API-Key: YOUR_BRAND_API_KEY"`),Ce=P(()=>`const res = await fetch('${e($)}/emails', {
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
console.log(result.message_id);`),Ie=P(()=>`import os, requests

res = requests.post(
    '${e($)}/emails',
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
print(result['message_id'])`),ke=P(()=>`payload, _ := json.Marshal(map[string]any{
    "to":            "customer@example.com",
    "template_slug": "password-reset",
    "variables":     map[string]string{"name": "Ana", "reset_url": "https://app.example.com/reset/abc"},
})
req, _ := http.NewRequest("POST", "${e($)}/emails", bytes.NewReader(payload))
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
json.NewDecoder(resp.Body).Decode(&result)`),Ke=P(()=>`<?php
$ch = curl_init('${e($)}/emails');
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
$result = json_decode(curl_exec($ch), true);`),Re=P(()=>`Host:      ${ne.replace(/^https?:\/\//,"").replace(/:\d+$/,"")}
Port:      587 (STARTTLS)
Username:  your brand slug
Password:  your brand API key

Optional headers:
  X-Outlet-Tags: welcome,onboarding
  X-Outlet-Meta-OrderID: 12345`);let Ee=P(()=>({curl:{code:e(Te),language:"bash"},nodejs:{code:e(Ce),language:"javascript"},python:{code:e(Ie),language:"python"},go:{code:e(ke),language:"go"},php:{code:e(Ke),language:"php"},smtp:{code:e(Re),language:"text"}})),de=P(()=>e(Ee)[e(J)]);qe(async()=>{await V()});async function V(){a(F,!0),a(I,"");try{const t=await rt();a(G,t.keys||[],!0)}catch{console.log("No API keys yet")}finally{a(F,!1)}}async function Se(){if(!e(D).trim()){a(I,"Please enter a name for the API key");return}a(N,!0),a(I,"");try{const t=await lt({name:e(D).trim()});a(E,t.key||"",!0),await V(),a(M,"API key created successfully. Copy it now - you won't be able to see it again!")}catch(t){a(I,t.message||"Failed to create API key",!0)}finally{a(N,!1)}}function je(t){a(ie,t,!0),a(H,!0)}async function Oe(){a(z,!0);try{await ot({},e(ie)),await V(),a(M,"API key revoked")}catch(t){a(I,t.message||"Failed to revoke API key",!0)}finally{a(z,!1)}}function ce(t){navigator.clipboard.writeText(t),a(X,t,!0),setTimeout(()=>a(X,""),2e3)}function Q(){a(Y,!1),a(D,""),a(E,"")}function pe(t){return t?new Date(t).toLocaleDateString("en-US",{month:"short",day:"numeric",year:"numeric"}):"Never"}var ue=wt();Qe("1kyawqp",t=>{ze(()=>{Je.title="API - Settings"})});var W=C(ue),me=r(W);{var De=t=>{le(t,{type:"error",title:"Error",onclose:()=>a(I,""),children:(y,m)=>{var o=ct(),f=r(o,!0);s(o),B(()=>j(f,e(I))),i(y,o)},$$slots:{default:!0}})};R(me,t=>{e(I)&&t(De)})}var ve=n(me,2);{var Me=t=>{le(t,{type:"success",title:"Success",onclose:()=>a(M,""),children:(y,m)=>{var o=pt(),f=r(o,!0);s(o),B(()=>j(f,e(M))),i(y,o)},$$slots:{default:!0}})};R(ve,t=>{e(M)&&t(Me)})}var ye=n(ve,2);re(ye,{children:(t,y)=>{var m=ft(),o=C(m),f=n(r(o),2);O(f,{type:"primary",size:"sm",onclick:()=>a(Y,!0),children:(l,d)=>{var c=ut(),b=C(c);nt(b,{class:"mr-1.5 h-4 w-4"}),K(),i(l,c)},$$slots:{default:!0}}),s(o);var A=n(o,2);{var v=l=>{var d=mt(),c=r(d);st(c,{}),s(d),i(l,d)},p=l=>{var d=q(),c=C(d);{var b=h=>{var g=vt(),k=r(g);it(k,{class:"mx-auto h-12 w-12 opacity-50"}),K(4),s(g),i(h,g)},w=h=>{var g=_t(),k=r(g),_=n(r(k));Ve(_,21,()=>e(G),U=>U.id,(U,S)=>{var L=yt(),T=r(L),Ne=r(T,!0);s(T);var Z=n(T),Xe=r(Z);s(Z);var ee=n(Z),Ye=r(ee,!0);s(ee);var te=n(ee),He=r(te,!0);s(te);var he=n(te),Be=r(he);O(Be,{type:"danger",size:"icon",onclick:()=>je(e(S).id),children:(ae,ge)=>{dt(ae,{class:"h-4 w-4"})},$$slots:{default:!0}}),s(he),s(L),B((ae,ge)=>{j(Ne,e(S).name),j(Xe,`${e(S).key_prefix??""}...`),j(Ye,ae),j(He,ge)},[()=>pe(e(S).last_used),()=>pe(e(S).created_at)]),i(U,L)}),s(_),s(k),s(g),i(h,g)};R(c,h=>{e(G).length===0?h(b):h(w,!1)},!0)}i(l,d)};R(A,l=>{e(F)?l(v):l(p,!1)})}i(t,m)},$$slots:{default:!0}});var _e=n(ye,2);re(_e,{children:(t,y)=>{var m=ht(),o=C(m),f=r(o);et(f,{class:"h-5 w-5 text-primary"}),K(2),s(o);var A=n(o,2),v=r(A),p=n(r(v),2),l=r(p);oe(l,{type:"text",get value(){return e($)},readonly:!0,class:"font-mono text-sm"});var d=n(l,2);O(d,{type:"secondary",onclick:()=>ce(e($)),children:(c,b)=>{var w=q(),h=C(w);{var g=_=>{be(_,{class:"h-4 w-4 text-green-500"})},k=_=>{Pe(_,{class:"h-4 w-4"})};R(h,_=>{e(X)===e($)?_(g):_(k,!1)})}i(c,w)},$$slots:{default:!0}}),s(p),s(v),K(2),s(A),i(t,m)},$$slots:{default:!0}});var Ue=n(_e,2);re(Ue,{children:(t,y)=>{var m=gt(),o=n(C(m),2),f=r(o);tt(f,{get tabs(){return we},variant:"pills",get activeTab(){return e(J)},set activeTab(v){a(J,v,!0)}}),s(o);var A=n(o,2);at(A,{get code(){return e(de).code},get language(){return e(de).language}}),i(t,m)},$$slots:{default:!0}}),s(W);var fe=n(W,2);We(fe,{title:"Create API Key",onclose:Q,get show(){return e(Y)},set show(y){a(Y,y,!0)},footer:y=>{var m=bt(),o=r(m);{var f=v=>{O(v,{type:"primary",onclick:Q,children:(p,l)=>{K();var d=se("Done");i(p,d)},$$slots:{default:!0}})},A=v=>{var p=xt(),l=C(p);O(l,{type:"secondary",onclick:Q,children:(c,b)=>{K();var w=se("Cancel");i(c,w)},$$slots:{default:!0}});var d=n(l,2);{let c=P(()=>e(N)||!e(D).trim());O(d,{type:"primary",onclick:Se,get disabled(){return e(c)},children:(b,w)=>{K();var h=se();B(()=>j(h,e(N)?"Creating...":"Create Key")),i(b,h)},$$slots:{default:!0}})}i(v,p)};R(o,v=>{e(E)?v(f):v(A,!1)})}s(m),i(y,m)},children:(y,m)=>{var o=q(),f=C(o);{var A=p=>{var l=$t(),d=r(l);le(d,{type:"warning",title:"Save your API key",children:(g,k)=>{var _=Pt();i(g,_)},$$slots:{default:!0}});var c=n(d,2),b=n(r(c),2),w=r(b);oe(w,{type:"text",get value(){return e(E)},readonly:!0,class:"font-mono text-sm"});var h=n(w,2);O(h,{type:"secondary",onclick:()=>ce(e(E)),children:(g,k)=>{var _=q(),U=C(_);{var S=T=>{be(T,{class:"h-4 w-4 text-green-500"})},L=T=>{Pe(T,{class:"h-4 w-4"})};R(U,T=>{e(X)===e(E)?T(S):T(L,!1)})}i(g,_)},$$slots:{default:!0}}),s(b),s(c),s(l),i(p,l)},v=p=>{var l=At(),d=r(l),c=n(r(d),2);oe(c,{type:"text",id:"key-name",placeholder:"e.g., Production API, Development",get value(){return e(D)},set value(b){a(D,b,!0)}}),K(2),s(d),s(l),i(p,l)};R(f,p=>{e(E)?p(A):p(v,!1)})}i(y,o)},$$slots:{footer:!0,default:!0}});var Le=n(fe,2);{let t=P(()=>e(z)?"Revoking...":"Revoke");Ze(Le,{title:"Revoke API Key",description:"Are you sure you want to revoke this API key? This cannot be undone.",get actionLabel(){return e(t)},actionType:"danger",onAction:Oe,onCancel:()=>a(H,!1),get open(){return e(H)},set open(y){a(H,y,!0)}})}i($e,ue),Ge()}export{Nt as component};
