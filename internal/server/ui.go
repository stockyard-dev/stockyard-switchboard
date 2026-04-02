package server

import "net/http"

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashHTML))
}

const dashHTML = `<!DOCTYPE html>
<html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Switchboard</title>
<style>
:root{--bg:#1a1410;--bg2:#241e18;--bg3:#2e261e;--rust:#c45d2c;--rl:#e8753a;--leather:#a0845c;--cream:#f0e6d3;--cd:#bfb5a3;--cm:#7a7060;--gold:#d4a843;--green:#4a9e5c;--red:#c44040;--mono:'JetBrains Mono',monospace;--serif:'Libre Baskerville',Georgia,serif}
*{margin:0;padding:0;box-sizing:border-box}body{background:var(--bg);color:var(--cream);font-family:var(--mono);font-size:13px}
a{color:var(--rl);text-decoration:none}a:hover{color:var(--gold)}
.hdr{padding:.7rem 1.2rem;border-bottom:1px solid var(--bg3);display:flex;justify-content:space-between;align-items:center}
.hdr h1{font-family:var(--serif);font-size:1rem}.hdr h1 span{color:var(--rl)}
.stats{font-size:.7rem;color:var(--leather)}.stats b{color:var(--cream);font-weight:600}
.main{max-width:700px;margin:0 auto;padding:1.5rem}
.card{background:var(--bg2);border:1px solid var(--bg3);padding:.8rem 1rem;margin-bottom:.5rem;display:flex;justify-content:space-between;align-items:center}
.card-title{font-size:.8rem;font-weight:600}.card-sub{font-size:.65rem;color:var(--cd)}
.btn{font-family:var(--mono);font-size:.7rem;padding:.3rem .6rem;border:1px solid;cursor:pointer;background:transparent}
.btn-p{border-color:var(--rust);color:var(--rl)}.btn-p:hover{background:var(--rust);color:var(--cream)}
.btn-d{border-color:var(--bg3);color:var(--cm)}.btn-d:hover{border-color:var(--red);color:var(--red)}
input{background:var(--bg);border:1px solid var(--bg3);color:var(--cream);padding:.4rem .6rem;font-family:var(--mono);font-size:.8rem;width:100%;outline:none;margin-bottom:.5rem}
input:focus{border-color:var(--rust)}
.empty{text-align:center;padding:2rem;color:var(--cm);font-style:italic;font-family:var(--serif)}
</style>
<link href="https://fonts.googleapis.com/css2?family=Libre+Baskerville:ital@0;1&family=JetBrains+Mono:wght@400;600&display=swap" rel="stylesheet">
</head><body>
<div class="hdr"><h1><span>Switchboard</span></h1><div class="stats">Total: <b id="ct">-</b></div></div>
<div class="main">
<div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:1rem">
<span style="font-size:.65rem;letter-spacing:2px;text-transform:uppercase;color:var(--rust)">All services</span>
<button class="btn btn-p" onclick="showCreate()">+ New</button>
</div>
<div id="list"></div>
</div>
<script>
async function load(){const r=await fetch('/api/services');const d=await r.json();document.getElementById('ct').textContent=d.count;
const el=document.getElementById('list');if(!d.services.length){el.innerHTML='<div class="empty">No services yet.</div>';return}
el.innerHTML=d.services.map(e=>'<div class="card"><div><div class="card-title">'+esc(e.name||e.title||e.id)+'</div><div class="card-sub">'+esc(e.created_at)+'</div></div><button class="btn btn-d" onclick="del(\''+e.id+'\')">Delete</button></div>').join('')}
function esc(s){return(s||'').replace(/&/g,'&amp;').replace(/</g,'&lt;')}
function showCreate(){const n=prompt('Name:');if(!n)return;fetch('/api/services',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name:n})}).then(load)}
async function del(id){if(!confirm('Delete?'))return;await fetch('/api/services/'+id,{method:'DELETE'});load()}
load();setInterval(load,30000)
</script></body></html>` + "`"
