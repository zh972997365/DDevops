import{c as i}from"./index-BvXIL9ct.js";/**
 * @license lucide-vue-next v1.0.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const u=i("download",[["path",{d:"M12 15V3",key:"m9g1x1"}],["path",{d:"M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4",key:"ih7n3h"}],["path",{d:"m7 10 5 5 5-5",key:"brsn70"}]]);function c(t){const e=t==null?"":String(t);return/[",\n]/.test(e)?`"${e.replace(/"/g,'""')}"`:e}function p(t,e,o){const s=[e.map(c).join(","),...o.map(d=>d.map(c).join(","))],r=new Blob([`\uFEFF${s.join(`\r
`)}`],{type:"text/csv;charset=utf-8"}),a=URL.createObjectURL(r),n=document.createElement("a");n.href=a,n.download=t,document.body.appendChild(n),n.click(),n.remove(),setTimeout(()=>URL.revokeObjectURL(a),1e3)}function m(){const t=new Date,e=o=>String(o).padStart(2,"0");return`${t.getFullYear()}${e(t.getMonth()+1)}${e(t.getDate())}-${e(t.getHours())}${e(t.getMinutes())}`}export{u as D,p as d,m as s};
