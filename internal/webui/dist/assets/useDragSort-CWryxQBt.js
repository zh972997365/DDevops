import{c as s,r as l,O as j}from"./index-BvXIL9ct.js";/**
 * @license lucide-vue-next v1.0.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const T=s("grip-vertical",[["circle",{cx:"9",cy:"12",r:"1",key:"1vctgf"}],["circle",{cx:"9",cy:"5",r:"1",key:"hp0tcf"}],["circle",{cx:"9",cy:"19",r:"1",key:"fkjjf6"}],["circle",{cx:"15",cy:"12",r:"1",key:"1tmaij"}],["circle",{cx:"15",cy:"5",r:"1",key:"19l28e"}],["circle",{cx:"15",cy:"19",r:"1",key:"f4zoj3"}]]);/**
 * @license lucide-vue-next v1.0.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const I=s("pencil",[["path",{d:"M21.174 6.812a1 1 0 0 0-3.986-3.987L3.842 16.174a2 2 0 0 0-.5.83l-1.321 4.352a.5.5 0 0 0 .623.622l4.353-1.32a2 2 0 0 0 .83-.497z",key:"1a8usu"}],["path",{d:"m15 5 4 4",key:"1mk7zo"}]]);/**
 * @license lucide-vue-next v1.0.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const w=s("trash-2",[["path",{d:"M10 11v6",key:"nco0om"}],["path",{d:"M14 11v6",key:"outv1u"}],["path",{d:"M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6",key:"miytrc"}],["path",{d:"M3 6h18",key:"d0wm0j"}],["path",{d:"M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2",key:"e791ji"}]]);function z({all:o,reorder:p,load:y}){const u=j(),a=l(""),r=l("");function k(e,t){a.value=e.id,t.dataTransfer.effectAllowed="move",t.dataTransfer.setData("text/plain",e.id)}function h(e,t){!a.value||a.value===e.id||(t.preventDefault(),t.dataTransfer.dropEffect="move",r.value=e.id)}function m(e){r.value===e.id&&(r.value="")}async function g(e,t){t.preventDefault();const i=a.value,d=e.id;if(a.value="",r.value="",!i||i===d)return;const n=[...o.value],f=n.findIndex(c=>c.id===i),v=n.findIndex(c=>c.id===d);if(f<0||v<0)return;n.splice(v,0,...n.splice(f,1));const D=o.value;o.value=n;try{await p(n.map(c=>c.id)),u.success("顺序已保存")}catch(c){o.value=D,u.error(c.message),await y()}}function x(){a.value="",r.value=""}return{dragging:a,over:r,onDragStart:k,onDragOver:h,onDragLeave:m,onDrop:g,onDragEnd:x}}export{T as G,I as P,w as T,z as u};
