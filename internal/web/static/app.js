/**
 * dev-assist web UI — vanilla JS SPA
 * No frameworks, no build step. Talks to the Go backend at /api/tools.
 */

'use strict';

// ── Category colour map (mirrors TUI menu.go palette) ─────────────────────
const CAT_COLOR = {
  'SSL & Certificates': '#00f5ff',
  'Auth & Tokens':      '#ff2d95',
  'Network':            '#39ff14',
  'Data':               '#ffe600',
};
function catColor(cat) { return CAT_COLOR[cat] || '#bf5fff'; }

// ── App state ─────────────────────────────────────────────────────────────
const state = {
  tools:    [],    // all tools from API
  filtered: [],    // after search filter
  selected: null,  // currently displayed tool (object)
  values:   {},    // { toolId: [inputVal, ...] }
};

// ── DOM refs ──────────────────────────────────────────────────────────────
const $ = id => document.getElementById(id);
const elSearch      = $('search');
const elSearchClear = $('search-clear');
const elToolList    = $('tool-list');
const elWelcome     = $('welcome');
const elToolPanel   = $('tool-panel');
const elToolCatBadge= $('tool-category-badge');
const elToolName    = $('tool-name');
const elToolDesc    = $('tool-desc');
const elCliHint     = $('cli-hint');
const elCliHintWrap = $('cli-hint-wrap');
const elToolForm    = $('tool-form');
const elRunBtn      = $('run-btn');
const elRunLabel    = $('run-label');
const elRunSpinner  = $('run-spinner');
const elFormError   = $('form-error');
const elResultWrap  = $('result-wrap');
const elResultBadge = $('result-badge');
const elResultOutput= $('result-output');
const elCopyBtn     = $('copy-btn');
const elDocsBtn     = $('docs-btn');
const elDocsPanel   = $('docs-panel');

// ── Bootstrap ─────────────────────────────────────────────────────────────
async function init() {
  try {
    const res = await fetch('/api/tools');
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    state.tools = await res.json();
    state.filtered = state.tools;
    renderSidebar();
    // Auto-select the first tool
    if (state.tools.length > 0) selectTool(state.tools[0].id);
  } catch (e) {
    elToolList.innerHTML = `<div class="no-results">⚠ Failed to load tools: ${e.message}</div>`;
  }
}

// ── Sidebar ───────────────────────────────────────────────────────────────
function renderSidebar() {
  elToolList.innerHTML = '';

  if (state.filtered.length === 0) {
    elToolList.innerHTML = '<div class="no-results">✗ No tools match — clear the filter</div>';
    return;
  }

  // Group by category preserving registry order
  const groups = [];
  const seen   = {};
  for (const t of state.filtered) {
    if (!seen[t.category]) {
      seen[t.category] = true;
      groups.push({ cat: t.category, tools: [] });
    }
    groups[groups.length - 1].tools.push(t);
  }

  // Handle the case where filtered results span non-adjacent categories
  const byKey = {};
  for (const t of state.filtered) {
    if (!byKey[t.category]) byKey[t.category] = [];
    byKey[t.category].push(t);
  }

  // Rebuild in a single ordered pass keyed by first occurrence
  const orderedCats = [];
  const seen2 = new Set();
  for (const t of state.filtered) {
    if (!seen2.has(t.category)) { seen2.add(t.category); orderedCats.push(t.category); }
  }

  const frag = document.createDocumentFragment();
  for (const cat of orderedCats) {
    const color = catColor(cat);

    // Category header
    const header = document.createElement('div');
    header.className = 'category-header';
    const pill = document.createElement('span');
    pill.className = 'category-pill';
    pill.textContent = cat;
    pill.style.background = color;
    header.appendChild(pill);
    frag.appendChild(header);

    // Tool items
    for (const t of byKey[cat]) {
      const item = document.createElement('div');
      item.className = 'tool-item' + (state.selected?.id === t.id ? ' active' : '');
      item.dataset.id = t.id;
      item.innerHTML = `<span class="arrow">▶</span>${escHtml(t.name)}`;
      item.addEventListener('click', () => selectTool(t.id));
      frag.appendChild(item);
    }
  }
  elToolList.appendChild(frag);
}

// ── Tool selection ─────────────────────────────────────────────────────────
function selectTool(id) {
  const tool = state.tools.find(t => t.id === id);
  if (!tool) return;
  state.selected = tool;

  // Sidebar active state (clear docs button too)
  document.querySelectorAll('.tool-item').forEach(el => {
    el.classList.toggle('active', el.dataset.id === id);
  });
  elDocsBtn.classList.remove('active');

  // Show tool panel, hide welcome + docs
  elWelcome.classList.add('hidden');
  elDocsPanel.classList.add('hidden');
  elToolPanel.classList.remove('hidden');

  // Header
  const color = catColor(tool.category);
  elToolCatBadge.textContent = tool.category;
  elToolCatBadge.style.background = color;
  elToolName.textContent = tool.name;
  elToolDesc.textContent = tool.description;

  // CLI hint
  elCliHint.textContent = buildCLIHint(tool);

  // Reset result
  elResultWrap.classList.add('hidden');
  elResultWrap.classList.remove('error');
  elResultOutput.textContent = '';
  elFormError.textContent = '';

  // Build form
  renderForm(tool);
}

// ── Form builder ───────────────────────────────────────────────────────────
function renderForm(tool) {
  elToolForm.innerHTML = '';

  if (!state.values[tool.id]) {
    state.values[tool.id] = tool.inputs.map(def => def.default || '');
  }

  tool.inputs.forEach((def, idx) => {
    const wrap = document.createElement('div');
    wrap.className = 'field-wrap';

    // Label
    const label = document.createElement('label');
    label.className = 'field-label';
    label.textContent = def.label;
    if (def.required) {
      const star = document.createElement('span');
      star.className = 'field-required';
      star.textContent = '*';
      label.appendChild(star);
    }
    wrap.appendChild(label);

    if (def.options && def.options.length > 0) {
      // Option toggle group
      const group = document.createElement('div');
      group.className = 'option-group';

      const currentVal = state.values[tool.id][idx] || def.default || def.options[0];

      def.options.forEach(opt => {
        const btn = document.createElement('button');
        btn.type = 'button';
        btn.className = 'option-btn' + (opt === currentVal ? ' active' : '');
        btn.textContent = opt;
        if (opt === def.default) btn.classList.add('option-default-dot');
        btn.addEventListener('click', () => {
          group.querySelectorAll('.option-btn').forEach(b => b.classList.remove('active'));
          btn.classList.add('active');
          state.values[tool.id][idx] = opt;
        });
        group.appendChild(btn);
      });

      // Ensure state reflects the active option
      state.values[tool.id][idx] = currentVal;
      wrap.appendChild(group);

    } else if (def.multiline) {
      const ta = document.createElement('textarea');
      ta.className = 'field-textarea';
      ta.placeholder = def.placeholder || '';
      ta.rows = 6;
      ta.value = state.values[tool.id][idx] || '';
      ta.addEventListener('input', () => { state.values[tool.id][idx] = ta.value; });
      ta.addEventListener('keydown', e => {
        if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') { e.preventDefault(); runTool(); }
      });
      wrap.appendChild(ta);

    } else {
      const inp = document.createElement('input');
      inp.type = 'text';
      inp.className = 'field-input';
      inp.placeholder = def.placeholder || '';
      inp.value = state.values[tool.id][idx] || '';
      inp.addEventListener('input', () => { state.values[tool.id][idx] = inp.value; });
      inp.addEventListener('keydown', e => {
        if (e.key === 'Enter' || ((e.ctrlKey || e.metaKey) && e.key === 'Enter')) {
          e.preventDefault(); runTool();
        }
      });
      wrap.appendChild(inp);
    }

    elToolForm.appendChild(wrap);
  });
}

// ── Run ────────────────────────────────────────────────────────────────────
async function runTool() {
  const tool = state.selected;
  if (!tool) return;

  // Validate required fields
  const inputs = state.values[tool.id] || tool.inputs.map(() => '');
  for (let i = 0; i < tool.inputs.length; i++) {
    const def = tool.inputs[i];
    if (def.required && !inputs[i]?.trim()) {
      elFormError.textContent = `"${def.label}" is required`;
      // Focus the field
      const fields = elToolForm.querySelectorAll('input, textarea');
      if (fields[i]) fields[i].focus();
      return;
    }
  }
  elFormError.textContent = '';

  // Loading state
  setLoading(true);
  elResultWrap.classList.add('hidden');

  try {
    const res = await fetch(`/api/tools/${tool.id}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ inputs }),
    });
    const data = await res.json();

    elResultWrap.classList.remove('hidden', 'error');
    if (data.error) {
      elResultWrap.classList.add('error');
      elResultBadge.textContent = '✗ ERROR';
      elResultOutput.textContent = data.error;
    } else {
      elResultBadge.textContent = '✓ RESULT';
      elResultOutput.textContent = data.output;
    }
    elResultWrap.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
  } catch (e) {
    elResultWrap.classList.remove('hidden');
    elResultWrap.classList.add('error');
    elResultBadge.textContent = '✗ ERROR';
    elResultOutput.textContent = `Network error: ${e.message}`;
  } finally {
    setLoading(false);
  }
}

function setLoading(on) {
  elRunBtn.disabled = on;
  elRunLabel.classList.toggle('hidden', on);
  elRunSpinner.classList.toggle('hidden', !on);
}

// ── Copy ───────────────────────────────────────────────────────────────────
elCopyBtn.addEventListener('click', async () => {
  const text = elResultOutput.textContent;
  try {
    await navigator.clipboard.writeText(text);
    elCopyBtn.classList.add('copied');
    elCopyBtn.textContent = '✓ Copied';
    setTimeout(() => {
      elCopyBtn.classList.remove('copied');
      elCopyBtn.innerHTML = `<svg viewBox="0 0 16 16" fill="currentColor">
        <path d="M4 2a2 2 0 0 1 2-2h6a2 2 0 0 1 2 2v10a2 2 0 0 1-2
                 2H6a2 2 0 0 1-2-2V2zm2 0v10h6V2H6zM2 4a2 2 0 0 0-2
                 2v8a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2v-1H8v1H2V6h1V4H2z"/>
      </svg>Copy`;
    }, 2000);
  } catch (_) { /* clipboard not available */ }
});

// ── Search / filter ────────────────────────────────────────────────────────
elSearch.addEventListener('input', () => {
  const q = elSearch.value.trim().toLowerCase();
  elSearchClear.classList.toggle('hidden', q === '');

  if (!q) {
    state.filtered = state.tools;
  } else {
    state.filtered = state.tools.filter(t =>
      t.name.toLowerCase().includes(q)       ||
      t.category.toLowerCase().includes(q)   ||
      t.description.toLowerCase().includes(q)
    );
  }
  renderSidebar();
});

elSearchClear.addEventListener('click', () => {
  elSearch.value = '';
  elSearch.dispatchEvent(new Event('input'));
  elSearch.focus();
});

// ── Run button ─────────────────────────────────────────────────────────────
elRunBtn.addEventListener('click', runTool);

// ── Docs button ────────────────────────────────────────────────────────────
elDocsBtn.addEventListener('click', showDocs);

// ── Global Ctrl+Enter shortcut ─────────────────────────────────────────────
document.addEventListener('keydown', e => {
  if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
    const active = document.activeElement;
    const inTextarea = active && active.tagName === 'TEXTAREA';
    if (!inTextarea) { e.preventDefault(); runTool(); }
  }
});

// ── Docs & Install panel ───────────────────────────────────────────────────
function showDocs() {
  state.selected = null;
  document.querySelectorAll('.tool-item').forEach(el => el.classList.remove('active'));
  elDocsBtn.classList.add('active');
  elWelcome.classList.add('hidden');
  elToolPanel.classList.add('hidden');
  elDocsPanel.classList.remove('hidden');

  if (elDocsPanel.dataset.rendered) return; // build once
  elDocsPanel.dataset.rendered = '1';

  // Build CLI reference rows from live registry
  const rows = state.tools.map(t => `
    <tr>
      <td class="td-id">${escHtml(t.id)}</td>
      <td class="td-name">${escHtml(t.name)}</td>
      <td class="td-cli">${escHtml(buildCLIHint(t))}</td>
    </tr>`).join('');

  elDocsPanel.innerHTML = `
<h1 class="docs-title">Docs &amp; Install</h1>

<section class="docs-section">
  <h2>Quick Install</h2>
  <h3>Download pre-built binary</h3>
  <code class="docs-code"># macOS — Apple Silicon
curl -Lo dev-assist https://github.com/datsabk/dev-assist/releases/latest/download/dev-assist-darwin-arm64
chmod +x dev-assist &amp;&amp; mv dev-assist /usr/local/bin/

# macOS — Intel
curl -Lo dev-assist https://github.com/datsabk/dev-assist/releases/latest/download/dev-assist-darwin-amd64
chmod +x dev-assist &amp;&amp; mv dev-assist /usr/local/bin/

# Linux — amd64
curl -Lo dev-assist https://github.com/datsabk/dev-assist/releases/latest/download/dev-assist-linux-amd64
chmod +x dev-assist &amp;&amp; mv dev-assist /usr/local/bin/

# Linux — arm64
curl -Lo dev-assist https://github.com/datsabk/dev-assist/releases/latest/download/dev-assist-linux-arm64
chmod +x dev-assist &amp;&amp; mv dev-assist /usr/local/bin/</code>
  <h3>Build from source (Go 1.21+)</h3>
  <code class="docs-code">git clone https://github.com/datsabk/dev-assist.git
cd dev-assist
make build          # current platform → bin/dev-assist
make install        # installs to $GOPATH/bin</code>
</section>

<section class="docs-section">
  <h2>Launch</h2>
  <code class="docs-code">dev-assist            # open interactive TUI
dev-assist --help     # list all subcommands
dev-assist web        # start this web UI on http://localhost:8080
dev-assist web --port 9000 --host 0.0.0.0   # expose publicly</code>
</section>

<section class="docs-section">
  <h2>TUI Keyboard Shortcuts</h2>
  <h3>Menu &amp; navigation</h3>
  <code class="docs-code">Type           Filter tools by name, category, or description
↑ ↓  /  j k   Navigate the list
Enter          Select a tool
q              Quit</code>
  <h3>Input screen</h3>
  <code class="docs-code">Tab / Shift+Tab   Next / previous field
Enter             Run  (single-line fields)
Ctrl+R            Run  (always — use inside multiline text areas)
Ctrl+F            Toggle file-path mode for the current field
← →               Cycle option toggles
Esc               Back to menu</code>
  <h3>Result screen</h3>
  <code class="docs-code">↑ ↓  /  j k   Scroll output
g / G          Jump to top / bottom
Esc            Back to input
m              Back to main menu
q              Quit</code>
</section>

<section class="docs-section">
  <h2>CLI Reference</h2>
  <p>Every tool is also a subcommand. Pass <code style="color:var(--cyan)">--help</code> to any subcommand for its full flag list.</p>
  <table class="docs-table">
    <thead>
      <tr>
        <th>Subcommand</th>
        <th>Tool</th>
        <th>Example</th>
      </tr>
    </thead>
    <tbody>${rows}</tbody>
  </table>
</section>`;
}

// ── CLI hint generator ─────────────────────────────────────────────────────
// Shows all available flags with simple <placeholder> names — no option lists,
// no values, so the hint stays readable regardless of how many choices exist.
function buildCLIHint(tool) {
  const parts = [`dev-assist ${tool.id}`];
  for (const def of tool.inputs) {
    if (!def.flag_name) continue;
    parts.push(`--${def.flag_name} <${def.flag_name}>`);
  }
  return parts.join(' ');
}

// ── Helpers ────────────────────────────────────────────────────────────────
function escHtml(s) {
  return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
}

// ── Start ──────────────────────────────────────────────────────────────────
init();
