import{$ as e,B as t,L as n,R as r,V as i,at as a,bt as o,c as s,et as c,ft as l,gt as u,ht as d,it as f,l as p,lt as m,ot as h,q as g,st as _,ut as v,y,yt as b,z as x}from"../chunks/BKXUg6nK.js";import"../chunks/xihTtKlq.js";import{t as S}from"../chunks/CLRycPbe.js";import{kt as C}from"../chunks/j-DxK_jk.js";import{L as w,O as T,ct as E,rt as D,st as O,z as k}from"../chunks/0YHF6kRM.js";import{t as A}from"../chunks/kj0bUAFJ.js";import{t as j}from"../chunks/CQl7K0HT.js";var M=i(`<div class="flex justify-center py-12"><!></div>`),N=i(`<!> Copied`,1),ee=i(`<!> Copy URL`,1),te=i(`<h2 class="text-lg font-medium text-text mb-2">Hosted Subscribe Form</h2> <p class="text-sm text-text-muted mb-4">Use this URL to link to a ready-to-use subscription page for this list.</p> <div class="flex items-center gap-2 flex-wrap sm:flex-nowrap"><div class="grow basis-full sm:basis-0 min-w-0 bg-bg-secondary px-3 py-2 rounded-md overflow-hidden"><code class="text-sm font-mono text-text truncate block"> </code></div> <!> <a target="_blank" rel="noopener noreferrer" class="btn btn-secondary flex-shrink-0"><!> Preview</a></div>`,1),ne=i(`<!> Copy`,1),re=i(`<h2 class="text-lg font-medium text-text mb-2">Embeddable HTML Form</h2> <p class="text-sm text-text-muted mb-4">Copy this HTML code to embed a subscribe form on your website.</p> <div class="relative"><pre class="bg-bg-secondary p-4 rounded-md overflow-x-auto text-sm font-mono text-text whitespace-pre-wrap"> </pre> <div class="absolute top-2 right-2"><!></div></div>`,1),P=i(`<div class="bg-bg-secondary p-4 rounded-md overflow-x-auto max-w-full"><pre class="text-sm font-mono text-text whitespace-pre-wrap break-all"> </pre></div>`),F=i(`<div class="bg-bg-secondary p-4 rounded-md overflow-x-auto max-w-full"><pre class="text-sm font-mono text-text whitespace-pre-wrap break-all"> </pre></div> <p class="text-xs text-text-muted mt-2">Install: <code class="bg-bg-secondary px-1 py-0.5 rounded">npm install @outlet/sdk</code></p>`,1),I=i(`<div class="bg-bg-secondary p-4 rounded-md overflow-x-auto max-w-full"><pre class="text-sm font-mono text-text whitespace-pre-wrap break-all"> </pre></div> <p class="text-xs text-text-muted mt-2">Install: <code class="bg-bg-secondary px-1 py-0.5 rounded">go get github.com/outlet-sh/outlet/sdk/go</code></p>`,1),L=i(`<div class="bg-bg-secondary p-4 rounded-md overflow-x-auto max-w-full"><pre class="text-sm font-mono text-text whitespace-pre-wrap break-all"> </pre></div> <p class="text-xs text-text-muted mt-2">Install: <code class="bg-bg-secondary px-1 py-0.5 rounded">pip install outlet-sdk</code></p>`,1),R=i(`<div class="bg-bg-secondary p-4 rounded-md overflow-x-auto max-w-full"><pre class="text-sm font-mono text-text whitespace-pre-wrap break-all"> </pre></div> <p class="text-xs text-text-muted mt-2">Install: <code class="bg-bg-secondary px-1 py-0.5 rounded">composer require outlet/sdk</code></p>`,1),z=i(`<h2 class="text-lg font-medium text-text mb-2">API Subscription</h2> <p class="text-sm text-text-muted mb-4">To subscribe users programmatically, use the Outlet SDK or API. Replace <code class="text-xs bg-bg-secondary px-1 py-0.5 rounded">your-api-key</code> with your organization's API key.</p> <!> <div class="mt-4"><!></div>`,1),B=i(`<div class="space-y-6"><!> <!> <!></div>`);function V(i,V){u(V,!0);let H=()=>p(S,`$page`,U),[U,W]=s(),G=j(),K=l(()=>H().params.id),q=v(null),J=v(!0),Y=v(!1),X=v(!1),Z=v(`curl`);c(()=>{ie()});async function ie(){m(J,!0);try{m(q,await C({},g(K)),!0)}catch(e){console.error(`Failed to load embed code:`,e)}finally{m(J,!1)}}function Q(){if(!g(q))return;let e=`${g(q).base_url}/s/${g(q).public_id}`;navigator.clipboard.writeText(e),m(Y,!0),setTimeout(()=>{m(Y,!1)},2e3)}function ae(){g(q)&&(navigator.clipboard.writeText(g(q).html),m(X,!0),setTimeout(()=>{m(X,!1)},2e3))}var $=t(),oe=a($),se=e=>{var t=M(),n=f(t);k(n,{}),o(t),x(e,t)},ce=i=>{var s=B(),c=f(s);E(c,{children:(i,s)=>{var c=te(),l=_(a(c),4),u=f(l),d=f(u),p=h(d);o(u);var m=_(u,2);w(m,{type:`secondary`,onclick:Q,class:`flex-shrink-0`,children:(e,r)=>{var i=t(),o=a(i),s=e=>{var t=N(),n=a(t);O(n,{class:`mr-2 h-4 w-4 text-success`}),b(),x(e,t)},c=e=>{var t=ee(),n=a(t);D(n,{class:`mr-2 h-4 w-4`}),b(),x(e,t)};n(o,e=>{g(Y)?e(s):e(c,-1)}),x(e,i)},$$slots:{default:!0}});var v=_(m,2),S=f(v);A(S,{class:`mr-2 h-4 w-4`}),b(),o(v),o(l),e(()=>{r(p,`${g(q).base_url??``}/s/${g(q).public_id??``}`),y(v,`href`,`${g(q).base_url??``}/s/${g(q).public_id??``}`)}),x(i,c)},$$slots:{default:!0}});var l=_(c,2);E(l,{children:(i,s)=>{var c=re(),l=_(a(c),4),u=f(l),d=h(u,!0),p=_(u,2),m=f(p);w(m,{type:`secondary`,size:`sm`,onclick:ae,children:(e,r)=>{var i=t(),o=a(i),s=e=>{var t=N(),n=a(t);O(n,{class:`mr-1 h-3 w-3 text-success`}),b(),x(e,t)},c=e=>{var t=ne(),n=a(t);D(n,{class:`mr-1 h-3 w-3`}),b(),x(e,t)};n(o,e=>{g(X)?e(s):e(c,-1)}),x(e,i)},$$slots:{default:!0}}),o(p),o(l),e(()=>r(d,g(q).html)),x(i,c)},$$slots:{default:!0}});var u=_(l,2);E(u,{children:(t,i)=>{var s=z(),c=_(a(s),4);T(c,{tabs:[{id:`curl`,label:`cURL`},{id:`typescript`,label:`TypeScript`},{id:`go`,label:`Go`},{id:`python`,label:`Python`},{id:`php`,label:`PHP`}],variant:`pills`,get activeTab(){return g(Z)},set activeTab(e){m(Z,e,!0)}});var l=_(c,2),u=f(l),d=t=>{var n=P(),i=f(n),a=h(i,!0);o(n),e(()=>r(a,`curl -X POST ${g(q)?.base_url||`https://your-outlet-instance.com`}/sdk/v1/lists/${G.list?.slug||`your-list-slug`}/subscribe \\
  -H "Content-Type: application/json" \\
  -H "X-API-Key: your-api-key" \\
  -d '{
    "email": "user@example.com",
    "name": "John Doe"
  }'`)),x(t,n)},p=t=>{var n=F(),i=a(n),s=f(i),c=h(s,!0);o(i),b(2),e(()=>r(c,`import { Outlet } from '@outlet/sdk';

const client = new Outlet(
  'your-api-key',
  '${g(q)?.base_url||`https://your-outlet-instance.com`}'
);

await client.lists.subscribeToList('${G.list?.slug||`your-list-slug`}', {
  email: 'user@example.com',
  name: 'John Doe'
});`)),x(t,n)},v=t=>{var n=I(),i=a(n),s=f(i),c=h(s,!0);o(i),b(2),e(()=>r(c,`package main

import (
    "context"
    outlet "github.com/outlet-sh/outlet/sdk/go"
)

func main() {
    client := outlet.NewClient(
        "your-api-key",
        "${g(q)?.base_url||`https://your-outlet-instance.com`}",
    )

    _, err := client.Lists.SubscribeToList(
        context.Background(),
        "${G.list?.slug||`your-list-slug`}",
        &outlet.SubscribeRequest{
            Email: "user@example.com",
            Name:  "John Doe",
        },
    )
}`)),x(t,n)},y=t=>{var n=L(),i=a(n),s=f(i),c=h(s,!0);o(i),b(2),e(()=>r(c,`from outlet_sdk import Outlet, SubscribeRequest

client = Outlet(
    api_key="your-api-key",
    base_url="${g(q)?.base_url||`https://your-outlet-instance.com`}"
)

client.lists.subscribe_to_list(
    "${G.list?.slug||`your-list-slug`}",
    SubscribeRequest(
        email="user@example.com",
        name="John Doe"
    )
)`)),x(t,n)},S=t=>{var n=R(),i=a(n),s=f(i),c=h(s,!0);o(i),b(2),e(()=>r(c,`<?php

use Outlet\\SDK\\Client;
use Outlet\\SDK\\Types\\SubscribeRequest;

$client = new Client(
    'your-api-key',
    '${g(q)?.base_url||`https://your-outlet-instance.com`}'
);

$client->lists->subscribeToList(
    '${G.list?.slug||`your-list-slug`}',
    new SubscribeRequest(
        email: 'user@example.com',
        name: 'John Doe'
    )
);`)),x(t,n)};n(u,e=>{g(Z)===`curl`?e(d):g(Z)===`typescript`?e(p,1):g(Z)===`go`?e(v,2):g(Z)===`python`?e(y,3):g(Z)===`php`&&e(S,4)}),o(l),x(t,s)},$$slots:{default:!0}}),o(s),x(i,s)};n(oe,e=>{g(J)?e(se):g(q)&&e(ce,1)}),x(i,$),d(),W()}export{V as component};