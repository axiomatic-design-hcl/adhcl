(()=>{"use strict";
const nav=document.querySelector(".tabs"),links=[...nav.querySelectorAll("a")],panels=links.map(a=>document.getElementById(a.hash.slice(1)));
nav.setAttribute("role","tablist");
links.forEach((a,i)=>{a.setAttribute("role","tab");a.id="tab-"+panels[i].id;a.setAttribute("aria-controls",panels[i].id);panels[i].setAttribute("role","tabpanel");panels[i].setAttribute("aria-labelledby",a.id);panels[i].tabIndex=-1;});
function show(panel){links.forEach((a,i)=>{const on=panels[i]===panel;a.setAttribute("aria-selected",String(on));a.tabIndex=on?0:-1;panels[i].hidden=!on;});}
function route(){const id=decodeURIComponent(location.hash.slice(1)),el=id&&document.getElementById(id),panel=(el&&el.closest(".panel"))||panels[0];show(panel);if(el&&el!==panel){el.scrollIntoView({block:"center"});}else{window.scrollTo(0,0);}}
function open(a){if(location.hash!==a.hash)history.pushState(null,"",a.hash);route();a.focus();}
links.forEach((a,i)=>a.addEventListener("keydown",e=>{const k={ArrowRight:i+1,ArrowLeft:i-1,Home:0,End:links.length-1}[e.key];if(k===undefined)return;e.preventDefault();open(links[(k+links.length)%links.length]);}));
links.forEach(a=>a.addEventListener("click",e=>{if(e.button||e.metaKey||e.ctrlKey||e.shiftKey||e.altKey)return;e.preventDefault();open(a);}));
window.addEventListener("hashchange",route);window.addEventListener("popstate",route);route();
const map=document.getElementById("map");if(!map)return;
const detail=document.getElementById("detail"),data=JSON.parse(document.getElementById("impact-data").textContent),impacts=data.impacts||[],nodes=map.querySelectorAll(".node"),edges=map.querySelectorAll(".edge,.map-label.q");
const fields=[["Affected","Affected"],["Possibly affected","Possible"],["Needs","Needs"],["Scenarios","Scenarios"],["Modules","Modules"],["Process variables","PVs"]].filter(([,key])=>impacts.some(x=>x[key]&&x[key].length));
function select(id){const impact=impacts.find(x=>x.DP===id);if(!impact)return;
 const ids=new Set([id,...(impact.Affected||[]),...(impact.Possible||[]),...(impact.Needs||[]),...(impact.Scenarios||[]),...(impact.Modules||[]),...(impact.PVs||[])]);
 map.classList.add("active");
 nodes.forEach(n=>{n.classList.toggle("hit",ids.has(n.dataset.id));n.classList.toggle("sel",n.dataset.id===id);if(n.hasAttribute("aria-pressed"))n.setAttribute("aria-pressed",String(n.dataset.id===id));});
 edges.forEach(e=>e.classList.toggle("hit",ids.has(e.dataset.from)&&ids.has(e.dataset.to)&&(e.dataset.from===id||e.dataset.to===id||!e.classList.contains("q"))));
 detail.replaceChildren();const title=document.createElement("h3");title.textContent=id;detail.append(title);
 for(const [label,key] of fields){const values=impact[key],p=document.createElement("p");p.textContent=label+": "+(values&&values.length?values.join(", "):"None recorded");detail.append(p);}}
nodes.forEach(n=>{if(!n.hasAttribute("aria-pressed"))return;n.addEventListener("click",()=>select(n.dataset.id));n.addEventListener("keydown",e=>{if(e.key==="Enter"||e.key===" "){e.preventDefault();select(n.dataset.id);}});});
if(data.initial)select(data.initial);
})();
