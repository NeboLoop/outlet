import"../chunks/DsnmJJEf.js";import{p as $e,u as Ce,b as k,i as e,f as s,d as Te,j as Le,s as A,c as r,r as a,h as _,n as g,t as T}from"../chunks/BtjfW9_B.js";import{s as L}from"../chunks/DEsepwd4.js";import{c as $,a as t,f as u}from"../chunks/DnSY09wi.js";import{i as y}from"../chunks/Cn_LYL0x.js";import{s as Se}from"../chunks/B8beEWxR.js";import{s as Pe,a as Re}from"../chunks/x6Lfh__E.js";import{p as De}from"../chunks/CYDbC9S2.js";import{W as Ie}from"../chunks/DfxTvd9F.js";import{C as Y,B as le,j as ie,i as ne,g as Oe}from"../chunks/DTNgJKyn.js";import{L as He}from"../chunks/OUNBezzj.js";import{g as qe}from"../chunks/DAYpR8bb.js";import{E as Ee}from"../chunks/YIvrxY7h.js";var Je=u('<div class="flex justify-center py-12"><!></div>'),Ue=u("<!> Copied",1),je=u("<!> Copy URL",1),Ae=u('<h2 class="text-lg font-medium text-text mb-2">Hosted Subscribe Form</h2> <p class="text-sm text-text-muted mb-4">Use this URL to link to a ready-to-use subscription page for this list.</p> <div class="flex items-center gap-2 flex-wrap sm:flex-nowrap"><div class="flex-1 min-w-0 bg-bg-secondary px-3 py-2 rounded-md overflow-hidden"><code class="text-sm font-mono text-text truncate block"> </code></div> <!> <a target="_blank" rel="noopener noreferrer" class="btn btn-secondary flex-shrink-0"><!> Preview</a></div>',1),Be=u("<!> Copied",1),ze=u("<!> Copy",1),Fe=u('<h2 class="text-lg font-medium text-text mb-2">Embeddable HTML Form</h2> <p class="text-sm text-text-muted mb-4">Copy this HTML code to embed a subscribe form on your website.</p> <div class="relative"><pre class="bg-bg-secondary p-4 rounded-md overflow-x-auto text-sm font-mono text-text whitespace-pre-wrap"> </pre> <div class="absolute top-2 right-2"><!></div></div>',1),Ke=u('<div class="bg-bg-secondary p-4 rounded-md overflow-x-auto max-w-full"><pre class="text-sm font-mono text-text whitespace-pre-wrap break-all"> </pre></div>'),Me=u('<div class="bg-bg-secondary p-4 rounded-md overflow-x-auto max-w-full"><pre class="text-sm font-mono text-text whitespace-pre-wrap break-all"> </pre></div> <p class="text-xs text-text-muted mt-2">Install: <code class="bg-bg-secondary px-1 py-0.5 rounded">npm install @outlet/sdk</code></p>',1),Ne=u('<div class="bg-bg-secondary p-4 rounded-md overflow-x-auto max-w-full"><pre class="text-sm font-mono text-text whitespace-pre-wrap break-all"> </pre></div> <p class="text-xs text-text-muted mt-2">Install: <code class="bg-bg-secondary px-1 py-0.5 rounded">go get github.com/localrivet/outlet/sdk/go</code></p>',1),Ge=u('<div class="bg-bg-secondary p-4 rounded-md overflow-x-auto max-w-full"><pre class="text-sm font-mono text-text whitespace-pre-wrap break-all"> </pre></div> <p class="text-xs text-text-muted mt-2">Install: <code class="bg-bg-secondary px-1 py-0.5 rounded">pip install outlet-sdk</code></p>',1),We=u('<div class="bg-bg-secondary p-4 rounded-md overflow-x-auto max-w-full"><pre class="text-sm font-mono text-text whitespace-pre-wrap break-all"> </pre></div> <p class="text-xs text-text-muted mt-2">Install: <code class="bg-bg-secondary px-1 py-0.5 rounded">composer require outlet/sdk</code></p>',1),Xe=u(`<h2 class="text-lg font-medium text-text mb-2">API Subscription</h2> <p class="text-sm text-text-muted mb-4">To subscribe users programmatically, use the Outlet SDK or API. Replace <code class="text-xs bg-bg-secondary px-1 py-0.5 rounded">your-api-key</code> with your organization's API key.</p> <!> <div class="mt-4"><!></div>`,1),Qe=u('<div class="space-y-6"><!> <!> <!></div>');function dt(ue,de){$e(de,!0);const ce=()=>Re(De,"$page",pe),[pe,me]=Pe(),q=qe();let ve=Le(()=>ce().params.id),o=A(null),M=A(!0),N=A(!1),G=A(!1),C=A("curl");Ce(()=>{fe()});async function fe(){k(M,!0);try{k(o,await Ie({},e(ve)),!0)}catch(c){console.error("Failed to load embed code:",c)}finally{k(M,!1)}}function be(){if(!e(o))return;const c=`${e(o).base_url}/s/${e(o).public_id}`;navigator.clipboard.writeText(c),k(N,!0),setTimeout(()=>{k(N,!1)},2e3)}function xe(){e(o)&&(navigator.clipboard.writeText(e(o).html),k(G,!0),setTimeout(()=>{k(G,!1)},2e3))}var Z=$(),_e=s(Z);{var ge=c=>{var S=Je(),W=r(S);He(W,{}),a(S),t(c,S)},ye=c=>{var S=$(),W=s(S);{var he=X=>{var Q=Qe(),ee=r(Q);Y(ee,{children:(E,re)=>{var h=Ae(),w=_(s(h),4),p=r(w),P=r(p),R=r(P);a(P),a(p);var D=_(p,2);le(D,{type:"secondary",onclick:be,class:"flex-shrink-0",children:(b,I)=>{var O=$(),m=s(O);{var n=l=>{var x=Ue(),v=s(x);ie(v,{class:"mr-2 h-4 w-4 text-green-500"}),g(),t(l,x)},d=l=>{var x=je(),v=s(x);ne(v,{class:"mr-2 h-4 w-4"}),g(),t(l,x)};y(m,l=>{e(N)?l(n):l(d,!1)})}t(b,O)},$$slots:{default:!0}});var i=_(D,2),f=r(i);Ee(f,{class:"mr-2 h-4 w-4"}),g(),a(i),a(w),T(()=>{L(R,`${e(o).base_url??""}/s/${e(o).public_id??""}`),Se(i,"href",`${e(o).base_url??""}/s/${e(o).public_id??""}`)}),t(E,h)},$$slots:{default:!0}});var te=_(ee,2);Y(te,{children:(E,re)=>{var h=Fe(),w=_(s(h),4),p=r(w),P=r(p,!0);a(p);var R=_(p,2),D=r(R);le(D,{type:"secondary",size:"sm",onclick:xe,children:(i,f)=>{var b=$(),I=s(b);{var O=n=>{var d=Be(),l=s(d);ie(l,{class:"mr-1 h-3 w-3 text-green-500"}),g(),t(n,d)},m=n=>{var d=ze(),l=s(d);ne(l,{class:"mr-1 h-3 w-3"}),g(),t(n,d)};y(I,n=>{e(G)?n(O):n(m,!1)})}t(i,b)},$$slots:{default:!0}}),a(R),a(w),T(()=>L(P,e(o).html)),t(E,h)},$$slots:{default:!0}});var we=_(te,2);Y(we,{children:(E,re)=>{var h=Xe(),w=_(s(h),4);Oe(w,{tabs:[{id:"curl",label:"cURL"},{id:"typescript",label:"TypeScript"},{id:"go",label:"Go"},{id:"python",label:"Python"},{id:"php",label:"PHP"}],variant:"pills",get activeTab(){return e(C)},set activeTab(i){k(C,i,!0)}});var p=_(w,2),P=r(p);{var R=i=>{var f=Ke(),b=r(f),I=r(b,!0);a(b),a(f),T(()=>L(I,`curl -X POST ${e(o)?.base_url||"https://your-outlet-instance.com"}/api/sdk/v1/lists/${q.list?.slug||"your-list-slug"}/subscribe \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer your-api-key" \\
  -d '{
    "email": "user@example.com",
    "name": "John Doe"
  }'`)),t(i,f)},D=i=>{var f=$(),b=s(f);{var I=m=>{var n=Me(),d=s(n),l=r(d),x=r(l,!0);a(l),a(d),g(2),T(()=>L(x,`import { Outlet } from '@outlet/sdk';

const client = new Outlet(
  'your-api-key',
  '${e(o)?.base_url||"https://your-outlet-instance.com"}'
);

await client.lists.subscribeToList('${q.list?.slug||"your-list-slug"}', {
  email: 'user@example.com',
  name: 'John Doe'
});`)),t(m,n)},O=m=>{var n=$(),d=s(n);{var l=v=>{var J=Ne(),B=s(J),z=r(B),V=r(z,!0);a(z),a(B),g(2),T(()=>L(V,`package main

import (
    "context"
    outlet "github.com/localrivet/outlet/sdk/go"
)

func main() {
    client := outlet.NewClient(
        "your-api-key",
        "${e(o)?.base_url||"https://your-outlet-instance.com"}",
    )

    _, err := client.Lists.SubscribeToList(
        context.Background(),
        "${q.list?.slug||"your-list-slug"}",
        &outlet.SubscribeRequest{
            Email: "user@example.com",
            Name:  "John Doe",
        },
    )
}`)),t(v,J)},x=v=>{var J=$(),B=s(J);{var z=H=>{var U=Ge(),F=s(U),K=r(F),j=r(K,!0);a(K),a(F),g(2),T(()=>L(j,`from outlet_sdk import Outlet, SubscribeRequest

client = Outlet(
    api_key="your-api-key",
    base_url="${e(o)?.base_url||"https://your-outlet-instance.com"}"
)

client.lists.subscribe_to_list(
    "${q.list?.slug||"your-list-slug"}",
    SubscribeRequest(
        email="user@example.com",
        name="John Doe"
    )
)`)),t(H,U)},V=H=>{var U=$(),F=s(U);{var K=j=>{var ae=We(),se=s(ae),oe=r(se),ke=r(oe,!0);a(oe),a(se),g(2),T(()=>L(ke,`<?php

use Outlet\\SDK\\Client;
use Outlet\\SDK\\Types\\SubscribeRequest;

$client = new Client(
    'your-api-key',
    '${e(o)?.base_url||"https://your-outlet-instance.com"}'
);

$client->lists->subscribeToList(
    '${q.list?.slug||"your-list-slug"}',
    new SubscribeRequest(
        email: 'user@example.com',
        name: 'John Doe'
    )
);`)),t(j,ae)};y(F,j=>{e(C)==="php"&&j(K)},!0)}t(H,U)};y(B,H=>{e(C)==="python"?H(z):H(V,!1)},!0)}t(v,J)};y(d,v=>{e(C)==="go"?v(l):v(x,!1)},!0)}t(m,n)};y(b,m=>{e(C)==="typescript"?m(I):m(O,!1)},!0)}t(i,f)};y(P,i=>{e(C)==="curl"?i(R):i(D,!1)})}a(p),t(E,h)},$$slots:{default:!0}}),a(Q),t(X,Q)};y(W,X=>{e(o)&&X(he)},!0)}t(c,S)};y(_e,c=>{e(M)?c(ge):c(ye,!1)})}t(ue,Z),Te(),me()}export{dt as component};
