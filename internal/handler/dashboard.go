package handler

import (
	"net/http"
)

const mapHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>StudEx Location Tracker</title>
<link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css" />
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;background:#0f172a;color:#e2e8f0;height:100vh;display:flex;flex-direction:column}
header{background:#1e293b;padding:12px 24px;display:flex;align-items:center;justify-content:space-between;border-bottom:1px solid #334155}
header h1{font-size:18px;font-weight:600;color:#38bdf8}
.stats{display:flex;gap:16px;font-size:13px;color:#94a3b8}
.stats span{color:#38bdf8;font-weight:600}
#map{flex:1}
#sidebar{position:absolute;top:60px;right:12px;width:280px;max-height:calc(100vh - 80px);overflow-y:auto;background:#1e293b;border-radius:8px;border:1px solid #334155;z-index:1000}
#sidebar h2{font-size:14px;padding:12px;border-bottom:1px solid #334155;color:#94a3b8}
.entity-item{padding:10px 12px;border-bottom:1px solid #1e293b;font-size:12px;cursor:pointer;transition:background .2s}
.entity-item:hover{background:#334155}
.entity-item .ref-id{color:#38bdf8;font-weight:600;font-size:13px}
.entity-item .coords{color:#94a3b8;margin-top:2px}
.entity-item .time{color:#64748b;margin-top:2px;font-size:11px}
.controls{display:flex;gap:8px;align-items:center}
.controls input{background:#0f172a;border:1px solid #334155;color:#e2e8f0;padding:4px 8px;border-radius:4px;width:60px;font-size:12px}
.controls button{background:#38bdf8;color:#0f172a;border:none;padding:4px 12px;border-radius:4px;font-size:12px;font-weight:600;cursor:pointer}
.controls button:hover{background:#7dd3fc}
.controls label{font-size:12px;color:#94a3b8}
</style>
</head>
<body>
<header>
<h1>StudEx Location Tracker</h1>
<div class="stats">
<div>Tracked: <span id="count">0</span></div>
<div>Updated: <span id="lastUpdate">-</span></div>
<div class="controls">
<label>Refresh(s):</label>
<input type="number" id="interval" value="5" min="1" max="60">
<button onclick="setInterval()">Apply</button>
</div>
</div>
</header>
<div id="map"></div>
<div id="sidebar">
<h2>Tracked Entities</h2>
<div id="entityList"></div>
</div>
<script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"></script>
<script>
const YOGYA = [-7.7756, 110.3808];
let map, markers = {}, refreshMs = 5000, timer;

map = L.map('map').setView(YOGYA, 14);
L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    attribution: '&copy; OpenStreetMap'
}).addTo(map);

async function fetchLocations() {
    try {
        const r = await fetch('/location');
        const data = await r.json();
        document.getElementById('count').textContent = data.length;
        document.getElementById('lastUpdate').textContent = new Date().toLocaleTimeString();

        const ids = new Set();
        data.forEach(e => {
            ids.add(e.ref_id);
            if (markers[e.ref_id]) {
                markers[e.ref_id].setLatLng([e.latitude, e.longitude]);
                markers[e.ref_id].setPopupContent(popupContent(e));
            } else {
                const m = L.marker([e.latitude, e.longitude]).addTo(map);
                m.bindPopup(popupContent(e));
                markers[e.ref_id] = m;
            }
        });
        Object.keys(markers).forEach(id => {
            if (!ids.has(id)) { map.removeLayer(markers[id]); delete markers[id]; }
        });

        const list = document.getElementById('entityList');
        list.innerHTML = data.length === 0 ? '<div style="padding:12px;color:#64748b;font-size:12px">No entities tracked</div>' :
            data.map(e => '<div class="entity-item" onclick="focusEntity(\''+e.ref_id+'\')">' +
                '<div class="ref-id">' + e.ref_id + '</div>' +
                '<div class="coords">' + e.latitude.toFixed(6) + ', ' + e.longitude.toFixed(6) + '</div>' +
                '<div class="time">' + new Date(e.updated_at).toLocaleTimeString() + '</div>' +
                '</div>').join('');
    } catch(err) { console.error(err); }
}

function popupContent(e) {
    return '<b>' + e.ref_id + '</b><br>' +
        e.latitude.toFixed(6) + ', ' + e.longitude.toFixed(6) + '<br>' +
        '<small>' + new Date(e.updated_at).toLocaleString() + '</small>';
}

function focusEntity(id) {
    if (markers[id]) {
        map.setView(markers[id].getLatLng(), 16);
        markers[id].openPopup();
    }
}

function startPolling() {
    clearInterval(timer);
    const s = parseInt(document.getElementById('interval').value) || 5;
    refreshMs = s * 1000;
    timer = setInterval(fetchLocations, refreshMs);
}

fetchLocations();
startPolling();
</script>
</body>
</html>`

func MapDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(mapHTML))
}
