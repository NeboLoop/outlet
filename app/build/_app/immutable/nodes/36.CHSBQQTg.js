import{$ as e,B as t,I as n,K as r,L as i,Q as a,R as o,at as s,c,ct as l,dt as u,ht as d,it as f,l as p,lt as m,mt as h,ot as g,rt as _,vt as v,y,yt as b,z as x}from"../chunks/C_OAeTtO.js";import"../chunks/xihTtKlq.js";import{t as S}from"../chunks/CfgaxGnT.js";import{W as C}from"../chunks/DucSFBgu.js";import{B as w,H as T,I as E,S as ee,p as D,x as O}from"../chunks/xZIB709U.js";import{t as k}from"../chunks/MsGSxNwW.js";import{t as A}from"../chunks/C3k8igxJ.js";var j=t(`<div class="flex justify-center py-12"><!></div>`),M=t(`<!> Copied`,1),te=t(`<!> Copy URL`,1),ne=t(`<h2 class="text-lg font-medium text-text mb-2">Hosted Subscribe Form</h2> <p class="text-sm text-text-muted mb-4">Use this URL to link to a ready-to-use subscription page for this list.</p> <div class="flex items-center gap-2 flex-wrap sm:flex-nowrap"><div class="flex-1 min-w-0 bg-bg-secondary px-3 py-2 rounded-md overflow-hidden"><code class="text-sm font-mono text-text truncate block"> </code></div> <!> <a target="_blank" rel="noopener noreferrer" class="btn btn-secondary flex-shrink-0"><!> Preview</a></div>`,1),re=t(`<!> Copy`,1),N=t(`<h2 class="text-lg font-medium text-text mb-2">Embeddable HTML Form</h2> <p class="text-sm text-text-muted mb-4">Copy this HTML code to embed a subscribe form on your website.</p> <div class="relative"><pre class="bg-bg-secondary p-4 rounded-md overflow-x-auto text-sm font-mono text-text whitespace-pre-wrap"> </pre> <div class="absolute top-2 right-2"><!></div></div>`,1),P=t(`<div class="bg-bg-secondary p-4 rounded-md overflow-x-auto max-w-full"><pre class="text-sm font-mono text-text whitespace-pre-wrap break-all"> </pre></div>`),F=t(`<div class="bg-bg-secondary p-4 rounded-md overflow-x-auto max-w-full"><pre class="text-sm font-mono text-text whitespace-pre-wrap break-all"> </pre></div> <p class="text-xs text-text-muted mt-2">Install: <code class="bg-bg-secondary px-1 py-0.5 rounded">npm install @outlet/sdk</code></p>`,1),I=t(`<div class="bg-bg-secondary p-4 rounded-md overflow-x-auto max-w-full"><pre class="text-sm font-mono text-text whitespace-pre-wrap break-all"> </pre></div> <p class="text-xs text-text-muted mt-2">Install: <code class="bg-bg-secondary px-1 py-0.5 rounded">go get github.com/localrivet/outlet/sdk/go</code></p>`,1),L=t(`<div class="bg-bg-secondary p-4 rounded-md overflow-x-auto max-w-full"><pre class="text-sm font-mono text-text whitespace-pre-wrap break-all"> </pre></div> <p class="text-xs text-text-muted mt-2">Install: <code class="bg-bg-secondary px-1 py-0.5 rounded">pip install outlet-sdk</code></p>`,1),R=t(`<div class="bg-bg-secondary p-4 rounded-md overflow-x-auto max-w-full"><pre class="text-sm font-mono text-text whitespace-pre-wrap break-all"> </pre></div> <p class="text-xs text-text-muted mt-2">Install: <code class="bg-bg-secondary px-1 py-0.5 rounded">composer require outlet/sdk</code></p>`,1),z=t(`<h2 class="text-lg font-medium text-text mb-2">API Subscription</h2> <p class="text-sm text-text-muted mb-4">To subscribe users programmatically, use the Outlet SDK or API. Replace <code class="text-xs bg-bg-secondary px-1 py-0.5 rounded">your-api-key</code> with your organization's API key.</p> <!> <div class="mt-4"><!></div>`,1),B=t(`<div class="space-y-6"><!> <!> <!></div>`);function V(t,V){d(V,!0);let H=()=>p(S,`$page`,U),[U,W]=c(),G=A(),K=u(()=>H().params.id),q=m(null),J=m(!0),Y=m(!1),X=m(!1),Z=m(`curl`);e(()=>{Q()});async function Q(){l(J,!0);try{l(q,await C({},r(K)),!0)}catch(e){console.error(`Failed to load embed code:`,e)}finally{l(J,!1)}}function ie(){if(!r(q))return;let e=`${r(q).base_url}/s/${r(q).public_id}`;navigator.clipboard.writeText(e),l(Y,!0),setTimeout(()=>{l(Y,!1)},2e3)}function ae(){r(q)&&(navigator.clipboard.writeText(r(q).html),l(X,!0),setTimeout(()=>{l(X,!1)},2e3))}var $=x(),oe=f($),se=e=>{var t=j(),n=_(t);ee(n,{}),b(t),o(e,t)},ce=e=>{var t=B(),c=_(t);T(c,{children:(e,t)=>{var c=ne(),l=g(f(c),4),u=_(l),d=_(u),p=s(d);b(u);var m=g(u,2);O(m,{type:`secondary`,onclick:ie,class:`flex-shrink-0`,children:(e,t)=>{var i=x(),a=f(i),s=e=>{var t=M(),n=f(t);w(n,{class:`mr-2 h-4 w-4 text-green-500`}),v(),o(e,t)},c=e=>{var t=te(),n=f(t);E(n,{class:`mr-2 h-4 w-4`}),v(),o(e,t)};n(a,e=>{r(Y)?e(s):e(c,-1)}),o(e,i)},$$slots:{default:!0}});var h=g(m,2),S=_(h);k(S,{class:`mr-2 h-4 w-4`}),v(),b(h),b(l),a(()=>{i(p,`${r(q).base_url??``}/s/${r(q).public_id??``}`),y(h,`href`,`${r(q).base_url??``}/s/${r(q).public_id??``}`)}),o(e,c)},$$slots:{default:!0}});var u=g(c,2);T(u,{children:(e,t)=>{var c=N(),l=g(f(c),4),u=_(l),d=s(u,!0),p=g(u,2),m=_(p);O(m,{type:`secondary`,size:`sm`,onclick:ae,children:(e,t)=>{var i=x(),a=f(i),s=e=>{var t=M(),n=f(t);w(n,{class:`mr-1 h-3 w-3 text-green-500`}),v(),o(e,t)},c=e=>{var t=re(),n=f(t);E(n,{class:`mr-1 h-3 w-3`}),v(),o(e,t)};n(a,e=>{r(X)?e(s):e(c,-1)}),o(e,i)},$$slots:{default:!0}}),b(p),b(l),a(()=>i(d,r(q).html)),o(e,c)},$$slots:{default:!0}});var d=g(u,2);T(d,{children:(e,t)=>{var c=z(),u=g(f(c),4);D(u,{tabs:[{id:`curl`,label:`cURL`},{id:`typescript`,label:`TypeScript`},{id:`go`,label:`Go`},{id:`python`,label:`Python`},{id:`php`,label:`PHP`}],variant:`pills`,get activeTab(){return r(Z)},set activeTab(e){l(Z,e,!0)}});var d=g(u,2),p=_(d),m=e=>{var t=P(),n=_(t),c=s(n,!0);b(t),a(()=>i(c,`curl -X POST ${r(q)?.base_url||`https://your-outlet-instance.com`}/api/sdk/v1/lists/${G.list?.slug||`your-list-slug`}/subscribe \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer your-api-key" \\
  -d '{
    "email": "user@example.com",
    "name": "John Doe"
  }'`)),o(e,t)},h=e=>{var t=F(),n=f(t),c=_(n),l=s(c,!0);b(n),v(2),a(()=>i(l,`import { Outlet } from '@outlet/sdk';

const client = new Outlet(
  'your-api-key',
  '${r(q)?.base_url||`https://your-outlet-instance.com`}'
);

await client.lists.subscribeToList('${G.list?.slug||`your-list-slug`}', {
  email: 'user@example.com',
  name: 'John Doe'
});`)),o(e,t)},y=e=>{var t=I(),n=f(t),c=_(n),l=s(c,!0);b(n),v(2),a(()=>i(l,`package main

import (
    "context"
    outlet "github.com/localrivet/outlet/sdk/go"
)

func main() {
    client := outlet.NewClient(
        "your-api-key",
        "${r(q)?.base_url||`https://your-outlet-instance.com`}",
    )

    _, err := client.Lists.SubscribeToList(
        context.Background(),
        "${G.list?.slug||`your-list-slug`}",
        &outlet.SubscribeRequest{
            Email: "user@example.com",
            Name:  "John Doe",
        },
    )
}`)),o(e,t)},x=e=>{var t=L(),n=f(t),c=_(n),l=s(c,!0);b(n),v(2),a(()=>i(l,`from outlet_sdk import Outlet, SubscribeRequest

client = Outlet(
    api_key="your-api-key",
    base_url="${r(q)?.base_url||`https://your-outlet-instance.com`}"
)

client.lists.subscribe_to_list(
    "${G.list?.slug||`your-list-slug`}",
    SubscribeRequest(
        email="user@example.com",
        name="John Doe"
    )
)`)),o(e,t)},S=e=>{var t=R(),n=f(t),c=_(n),l=s(c,!0);b(n),v(2),a(()=>i(l,`<?php

use Outlet\\SDK\\Client;
use Outlet\\SDK\\Types\\SubscribeRequest;

$client = new Client(
    'your-api-key',
    '${r(q)?.base_url||`https://your-outlet-instance.com`}'
);

$client->lists->subscribeToList(
    '${G.list?.slug||`your-list-slug`}',
    new SubscribeRequest(
        email: 'user@example.com',
        name: 'John Doe'
    )
);`)),o(e,t)};n(p,e=>{r(Z)===`curl`?e(m):r(Z)===`typescript`?e(h,1):r(Z)===`go`?e(y,2):r(Z)===`python`?e(x,3):r(Z)===`php`&&e(S,4)}),b(d),o(e,c)},$$slots:{default:!0}}),b(t),o(e,t)};n(oe,e=>{r(J)?e(se):r(q)&&e(ce,1)}),o(t,$),h(),W()}export{V as component};