async function api(path) {
  const res = await fetch(path);
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

function formatDate(iso) {
  if (!iso || iso.startsWith('0001')) return '—';
  return iso.slice(0, 10);
}

function formatBytes(n) {
  if (!n) return '—';
  if (n < 1024) return n + ' B';
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + ' KB';
  return (n / (1024 * 1024)).toFixed(2) + ' MB';
}

// Tabs
document.querySelectorAll('.tab').forEach(btn => {
  btn.addEventListener('click', () => {
    document.querySelectorAll('.tab').forEach(b => b.classList.remove('active'));
    document.querySelectorAll('.panel').forEach(p => p.classList.remove('active'));
    btn.classList.add('active');
    document.getElementById('tab-' + btn.dataset.tab).classList.add('active');
  });
});

async function loadHealth() {
  const el = document.getElementById('health');
  try {
    const data = await api('/api/health');
    el.textContent = data.status === 'ok' ? '● online' : '● ' + data.status;
    el.className = 'badge ' + (data.status === 'ok' ? 'ok' : 'degraded');
  } catch {
    el.textContent = '● offline';
    el.className = 'badge error';
  }
}

async function loadOverview() {
  const data = await api('/api/overview');
  const cards = document.getElementById('overview-cards');
  const s = data.stats;
  const stack = data.stack;

  cards.innerHTML = `
    <div class="card">
      <div class="label">Configs</div>
      <div class="value">${s.configs}</div>
      <div class="meta">pools, strategies, runs</div>
    </div>
    <div class="card">
      <div class="label">Data files</div>
      <div class="value">${s.data_files}</div>
      <div class="meta">Parquet on disk</div>
    </div>
    <div class="card">
      <div class="label">Backtest runs</div>
      <div class="value">${s.runs}</div>
      <div class="meta">SQLite</div>
    </div>
    <div class="card">
      <div class="label">Stack</div>
      <div class="value" style="font-size:1rem">${stack.language}</div>
      <div class="meta">${stack.candles} · ${stack.results}</div>
    </div>
  `;
}

async function loadConfigs() {
  const data = await api('/api/configs');
  const container = document.getElementById('configs-list');
  const byCat = {};
  for (const f of data.files) {
    (byCat[f.category] ||= []).push(f);
  }

  container.innerHTML = Object.entries(byCat).map(([cat, files]) => `
    <div class="config-group">
      <h3>${cat}</h3>
      ${files.map(f => `
        <div class="config-item">
          <span class="name">${f.name}</span>
          <span class="path">${f.path}</span>
        </div>
      `).join('')}
    </div>
  `).join('');

  document.getElementById('config-details').textContent =
    JSON.stringify(data.details, null, 2);
}

async function loadData() {
  const data = await api('/api/data');
  const tbody = document.querySelector('#data-table tbody');
  const inv = data.inventory || [];

  if (inv.length === 0) {
    tbody.innerHTML = `<tr><td colspan="7"><div class="empty">No data yet. Run: make fetch-btc</div></td></tr>`;
    return;
  }

  tbody.innerHTML = inv.map(item => `
    <tr>
      <td>${item.Key?.Exchange || item.key?.exchange || '—'}</td>
      <td>${item.Key?.Symbol || item.key?.symbol || '—'}</td>
      <td>${item.Key?.Timeframe || item.key?.timeframe || '—'}</td>
      <td>${item.BarCount ?? item.bar_count ?? 0}</td>
      <td>${formatDate(item.FirstBar || item.first_bar)}</td>
      <td>${formatDate(item.LastBar || item.last_bar)}</td>
      <td>${formatBytes(item.FileSizeB ?? item.file_size_b)}</td>
    </tr>
  `).join('');
}

async function loadRuns() {
  const data = await api('/api/runs');
  const tbody = document.querySelector('#runs-table tbody');
  const runs = data.runs || [];

  if (runs.length === 0) {
    tbody.innerHTML = `<tr><td colspan="11"><div class="empty">No runs yet — run: make backtest</div></td></tr>`;
    return;
  }

  tbody.innerHTML = runs.map(r => `
    <tr>
      <td>${r.id}</td>
      <td>${r.name}</td>
      <td>${r.strategy}</td>
      <td>${r.symbol}</td>
      <td>${r.timeframe}</td>
      <td>${fmt(r.total_return_pct)}</td>
      <td>${fmt(r.sharpe_ratio)}</td>
      <td>${fmt(r.max_drawdown_pct)}</td>
      <td>${r.total_trades ?? '—'}</td>
      <td>${fmt(r.final_balance)}</td>
      <td>${r.status}</td>
    </tr>
  `).join('');
}

async function loadSweeps() {
  const data = await api('/api/sweeps');
  const tbody = document.querySelector('#sweeps-table tbody');
  const sweeps = data.sweeps || [];

  if (sweeps.length === 0) {
    tbody.innerHTML = `<tr><td colspan="13"><div class="empty">No optimization results yet — run: make optimize</div></td></tr>`;
    return;
  }

  tbody.innerHTML = sweeps.map(s => `
    <tr>
      <td>${s.run_id}</td>
      <td>${s.rank}</td>
      <td>${s.strategy}</td>
      <td>${s.symbol}</td>
      <td>${s.timeframe}</td>
      <td>${s.metric_name}</td>
      <td>${fmt(s.metric_value)}</td>
      <td>${fmt(s.total_return_pct)}</td>
      <td>${fmt(s.sharpe_ratio)}</td>
      <td>${fmt(s.max_drawdown_pct)}</td>
      <td>${s.total_trades ?? '—'}</td>
      <td>${fmt(s.final_balance)}</td>
      <td><code>${(s.params_json || '').slice(0, 80)}</code></td>
    </tr>
  `).join('');
}

function fmt(v) {
  if (v === 0 || v == null) return '—';
  return typeof v === 'number' ? v.toFixed(2) : v;
}

async function loadIndicators() {
  const symbol = document.getElementById('ind-symbol').value.trim();
  const tf = document.getElementById('ind-tf').value;
  const cards = document.getElementById('ind-latest');
  const tbody = document.querySelector('#ind-table tbody');

  try {
    const data = await api(`/api/indicators?symbol=${encodeURIComponent(symbol)}&timeframe=${encodeURIComponent(tf)}`);
    const l = data.latest || {};
    cards.innerHTML = `
      <div class="card"><div class="label">Close</div><div class="value" style="font-size:1.2rem">${fmt(l.close)}</div></div>
      <div class="card"><div class="label">EMA 12 / 26</div><div class="value" style="font-size:1rem">${fmt(l.ema_12)} / ${fmt(l.ema_26)}</div></div>
      <div class="card"><div class="label">RSI 14</div><div class="value">${fmt(l.rsi_14)}</div></div>
      <div class="card"><div class="label">MACD</div><div class="value" style="font-size:1rem">${fmt(l.macd)} / ${fmt(l.macd_signal)}</div></div>
    `;
    const recent = (data.recent || []).slice().reverse();
    tbody.innerHTML = recent.map(row => `
      <tr>
        <td>${formatDate(row.timestamp)}</td>
        <td>${fmt(row.close)}</td>
        <td>${fmt(row.ema_12)}</td>
        <td>${fmt(row.ema_26)}</td>
        <td>${fmt(row.sma_20)}</td>
        <td>${fmt(row.rsi_14)}</td>
        <td>${fmt(row.macd)}</td>
        <td>${fmt(row.macd_signal)}</td>
        <td>${fmt(row.volume_sma_20)}</td>
      </tr>
    `).join('');
  } catch (e) {
    cards.innerHTML = '';
    tbody.innerHTML = `<tr><td colspan="9"><div class="empty">${e.message || 'Load data first: make fetch-btc'}</div></td></tr>`;
  }
}

document.getElementById('ind-load').addEventListener('click', loadIndicators);

async function init() {
  await loadHealth();
  await Promise.all([loadOverview(), loadConfigs(), loadData(), loadRuns(), loadSweeps()]);
  loadIndicators().catch(() => {});
}

init().catch(err => {
  console.error(err);
  document.getElementById('health').textContent = '● error';
  document.getElementById('health').className = 'badge error';
});
