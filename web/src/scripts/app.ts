import uPlot from 'uplot';
import 'uplot/dist/uPlot.min.css';

const els = {
    searchForm: document.getElementById('searchForm'),
    steamIdInput: document.getElementById('steamIdInput'),
    searchBtn: document.getElementById('searchBtn'),
    searchBtnIcon: document.getElementById('searchBtnIcon'),
    searchError: document.getElementById('searchError'),
    resultChip: document.getElementById('resultChip'),
    chipName: document.getElementById('chipName'),
    chipSteamLink: document.getElementById('chipSteamLink'),
    chipCsfloatLink: document.getElementById('chipCsfloatLink'),
    chipWhen: document.getElementById('chipWhen'),
    chipVerdict: document.getElementById('chipVerdict'),
    kpiIndexed: document.getElementById('kpiIndexed'),
    kpiFlagged: document.getElementById('kpiFlagged'),
    kpiFlagged24h: document.getElementById('kpiFlagged24h'),
    chartContainer: document.getElementById('chartContainer'),
    reversalsBody: document.getElementById('reversalsBody'),
    loadMoreBtn: document.getElementById('loadMoreBtn'),
};

const fmtNumber = new Intl.NumberFormat('en-US');

function formatDate(ms) {
    if (!ms) return '';
    const d = new Date(ms);
    return d.toLocaleDateString('en-US', { month: 'short', day: '2-digit', timeZone: 'UTC' })
        + ', '
        + d.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false, timeZone: 'UTC' });
}

function setKpi(el, value) {
    el.textContent = fmtNumber.format(value);
    el.classList.remove('skeleton');
}

function setKpiError(el) {
    el.textContent = '—';
    el.classList.add('skeleton');
}

async function loadSummary() {
    try {
        const r = await fetch('/api/v1/stats/summary');
        if (!r.ok) throw new Error('summary ' + r.status);
        const data = await r.json();
        setKpi(els.kpiIndexed, data.steam_ids_searched ?? 0);
        setKpi(els.kpiFlagged, data.traders_flagged ?? 0);
        setKpi(els.kpiFlagged24h, data.traders_flagged_24h ?? 0);
    } catch (err) {
        console.error('summary load failed:', err);
        setKpiError(els.kpiIndexed);
        setKpiError(els.kpiFlagged);
        setKpiError(els.kpiFlagged24h);
    }
}

// Annotation chips that float above the Reversal Graph at the date
// of the event. Edit this array directly. Each entry needs `date`
// (YYYY-MM-DD, UTC) and `title` (short, shown on the chip).
// `description` and `url` are optional and surface in the hover
// popover. Chips only render for events that fall inside the
// currently-selected period (7d / 30d / 3m / 6m / 1y). When two
// chips would overlap, the later one auto-stacks onto a row below;
// if more than 3 rows are needed the oldest chips are dropped
// (check the browser console). Re-rendered on each chart redraw
// (period change) and on resize. See PRD §6.3 for editorial intent.
const CS2_EVENTS = [
    {
        date: '2026-05-20',
        title: 'Trade Revert Update',
        description: 'Valve enabled merchant-led trade reversals for CS2, broadening the set of disputes that route through reverse.watch.',
        url: '',
    },
    {
        date: '2026-03-08',
        title: 'Anti-Cheat Wave',
        description: 'Large VAC ban wave hit account-sharing rings; downstream effect on reversal volume as compromised accounts were flagged.',
        url: '',
    },
    {
        date: '2025-10-22',
        title: 'Retake Update',
        description: 'New Retake game mode re-introduced + users can now use covert skins to trade up to a knife or gloves',
        url: '',
    },
];

const cs2Events = CS2_EVENTS.filter(e => e && e.date && e.title);

function clearEventChips() {
    els.chartContainer.querySelectorAll('.event-chip').forEach(c => c.remove());
}

let eventChipsRaf = 0;
function renderEventChips(chart) {
    // Defer to the next animation frame so uPlot's layout (and
    // its valToPos scale math) is fully settled. Without this,
    // calling renderEventChips synchronously right after
    // `new uPlot(...)` can hit a stale/zero-width bbox on the
    // first paint after a period switch, returning positions
    // that fall outside the visible container.
    cancelAnimationFrame(eventChipsRaf);
    eventChipsRaf = requestAnimationFrame(() => paintEventChips(chart));
}

function paintEventChips(chart) {
    clearEventChips();
    if (!chart || !cs2Events.length) return;

    const xs = chart.data[0];
    if (!xs || !xs.length) return;
    const xMin = xs[0];
    const xMax = xs[xs.length - 1];

    const inWindow = cs2Events
        .map(e => ({ ...e, ts: Date.parse(e.date + 'T00:00:00Z') / 1000 }))
        .filter(e => Number.isFinite(e.ts) && e.ts >= xMin && e.ts <= xMax)
        .sort((a, b) => a.ts - b.ts);
    if (!inWindow.length) return;

    // Row-stacking collision detection. Up to 3 rows; oldest
    // overflow is dropped (with a console warning) so the user
    // knows to space events out in the JSON.
    const MIN_GAP = 110;
    const ROW_HEIGHT = 26;
    const MAX_ROWS = 3;
    const placed = [];
    let dropped = 0;
    // valToPos (canvasPixels=false) returns coordinates relative to the
    // plot area, but chips are absolutely positioned relative to the
    // chart container, so add the plot-area left inset. bbox.left is in
    // canvas pixels, so divide by devicePixelRatio to get CSS px. Mirrors
    // how the area-fill gradient anchors to u.bbox.top.
    const pxRatio = window.devicePixelRatio || 1;
    const xOffset = chart.bbox.left / pxRatio;
    for (const ev of inWindow) {
        const x = chart.valToPos(ev.ts, 'x') + xOffset;
        if (!Number.isFinite(x)) continue;
        let row = 0;
        while (row < MAX_ROWS && placed.some(p => p.row === row && Math.abs(p.x - x) < MIN_GAP)) {
            row++;
        }
        if (row >= MAX_ROWS) { dropped++; continue; }
        placed.push({ ev, x, row });
    }
    if (dropped > 0) {
        console.warn(
            `cs2-events: ${dropped} chip(s) hidden due to overlap. ` +
            `Edit the CS2_EVENTS array in web/src/scripts/app.ts or narrow the chart period to fit more.`
        );
    }

    for (const { ev, x, row } of placed) {
        const chip = document.createElement('div');
        chip.className = 'event-chip';
        chip.style.left = x + 'px';
        chip.style.top = (8 + row * ROW_HEIGHT) + 'px';
        chip.tabIndex = 0;
        chip.setAttribute('role', 'note');
        chip.setAttribute('aria-label', `${ev.title}, ${ev.date}`);
        chip.innerHTML = `
            <span class="event-chip-title"></span>
            <div class="event-popover">
                <div class="event-pop-date"></div>
                <div class="event-pop-title"></div>
                <div class="event-pop-desc"></div>
            </div>`;
        chip.querySelector('.event-chip-title').textContent = ev.title;
        chip.querySelector('.event-pop-date').textContent = formatEventDate(ev.ts);
        chip.querySelector('.event-pop-title').textContent = ev.title;
        const descEl = chip.querySelector('.event-pop-desc');
        if (ev.description) {
            descEl.textContent = ev.description;
        } else {
            descEl.remove();
        }
        if (ev.url) {
            const a = document.createElement('a');
            a.className = 'event-pop-link';
            a.href = ev.url;
            a.target = '_blank';
            a.rel = 'noopener';
            a.textContent = 'Read more →';
            chip.querySelector('.event-popover').appendChild(a);
        }
        els.chartContainer.appendChild(chip);
    }
}

function formatEventDate(ts) {
    const d = new Date(ts * 1000);
    return d.toLocaleDateString('en-US', {
        weekday: 'long', year: 'numeric', month: 'long', day: 'numeric', timeZone: 'UTC',
    });
}

let chartInstance = null;
let chartTooltip = null;

function showChartEmpty(message) {
    els.chartContainer.innerHTML = '';
    els.chartContainer.classList.add('is-empty');
    els.chartContainer.textContent = message;
}

function ensureTooltip() {
    if (chartTooltip) return chartTooltip;
    chartTooltip = document.createElement('div');
    chartTooltip.className = 'chart-tooltip';
    chartTooltip.innerHTML = `
        <div class="tt-date"></div>
        <div class="tt-count">
            <span class="tt-value">0</span>
            <span class="tt-label">reversals</span>
        </div>`;
    els.chartContainer.appendChild(chartTooltip);
    return chartTooltip;
}

function renderChart(daily, days) {
    els.chartContainer.classList.remove('is-empty', 'is-error');
    if (chartInstance) {
        chartInstance.destroy();
        chartInstance = null;
    }
    els.chartContainer.innerHTML = '';
    chartTooltip = null;

    if (!daily || daily.length === 0) {
        showChartEmpty('No reversal data for this window yet.');
        return;
    }

    const xs = new Array(daily.length);
    const ys = new Array(daily.length);
    for (let i = 0; i < daily.length; i++) {
        // Parse YYYY-MM-DD as UTC midnight to match server bucketing.
        xs[i] = Date.parse(daily[i].date + 'T00:00:00Z') / 1000;
        ys[i] = daily[i].count;
    }

    // For ≤60 days, show "MMM DD" on the x-axis; for longer windows,
    // collapse to month-only so 6m/1y don't overlap labels.
    const showDayOnAxis = days <= 60;

    // Pair the label format with a matching increment ladder so
    // uPlot never picks a sub-month increment when we're rendering
    // month-only labels (otherwise the 3m view would draw 14-day
    // ticks that all render as "Apr Apr May May Jun Jun"). Above
    // 30 days uPlot does month-aware iteration, so ticks land on
    // the 1st/cadence-aligned points of each month.
    const xIncrs = showDayOnAxis
        ? [86400, 2 * 86400, 7 * 86400, 14 * 86400]
        : [30 * 86400, 61 * 86400, 91 * 86400, 182 * 86400];

    const opts = {
        width: els.chartContainer.getBoundingClientRect().width || 800,
        height: els.chartContainer.getBoundingClientRect().height || 360,
        // Force uPlot to compute tick splits in UTC so they line up
        // exactly with our UTC-midnight data x-values. Without this,
        // uPlot uses local time — in any non-UTC zone, the "May 24"
        // tick lands 1–12 hours off from the May-24 data point and
        // the tick label can read "May 23" while the tooltip reads
        // "May 24".
        tzDate: (ts) => uPlot.tzDate(new Date(ts * 1000), 'Etc/UTC'),
        cursor: {
            drag: { x: false, y: false },
            points: { size: 8, stroke: '#1d9bf0', fill: '#0c0d11', width: 2 },
        },
        scales: { x: { time: true }, y: { range: (u, dataMin, dataMax) => [0, Math.max(dataMax, 1) * 1.1] } },
        axes: [
            {
                stroke: '#5a6068',
                grid: { stroke: 'rgba(255,255,255,0.04)', width: 1 },
                ticks: { show: false },
                space: 70,
                incrs: xIncrs,
                values: (u, splits) => splits.map(s => {
                    const d = new Date(s * 1000);
                    return showDayOnAxis
                        ? d.toLocaleDateString('en-US', { month: 'short', day: '2-digit', timeZone: 'UTC' })
                        : d.toLocaleDateString('en-US', { month: 'short', timeZone: 'UTC' });
                }),
            },
            {
                stroke: '#5a6068',
                grid: { stroke: 'rgba(255,255,255,0.04)', width: 1 },
                ticks: { show: false },
                size: 50,
                values: (u, splits) => splits.map(s => fmtNumber.format(Math.round(s))),
            }
        ],
        series: [
            {},
            {
                label: 'Reversals',
                stroke: '#1d9bf0',
                width: 2,
                fill: (u) => {
                    const ctx = u.ctx;
                    const grad = ctx.createLinearGradient(0, u.bbox.top, 0, u.bbox.top + u.bbox.height);
                    grad.addColorStop(0, 'rgba(29, 155, 240, 0.30)');
                    grad.addColorStop(1, 'rgba(29, 155, 240, 0.00)');
                    return grad;
                },
                points: { show: false },
            }
        ],
        legend: { show: false },
        hooks: {
            setCursor: [
                (u) => {
                    const tt = ensureTooltip();
                    const idx = u.cursor.idx;
                    if (idx == null || u.cursor.left < 0 || u.cursor.top < 0) {
                        tt.classList.remove('visible');
                        return;
                    }
                    const ts = u.data[0][idx];
                    const ct = u.data[1][idx];
                    const d = new Date(ts * 1000);
                    tt.querySelector('.tt-date').textContent =
                        d.toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: '2-digit', timeZone: 'UTC' });
                    tt.querySelector('.tt-value').textContent = fmtNumber.format(ct);
                    tt.querySelector('.tt-label').textContent = ct === 1 ? 'reversal' : 'reversals';

                    // valToPos (canvasPixels=false) is relative to the plot
                    // area; the tooltip is positioned relative to the chart
                    // container, so add the plot-area inset (bbox is in canvas
                    // pixels, so divide by devicePixelRatio for CSS px).
                    const pxRatio = window.devicePixelRatio || 1;
                    const x = u.valToPos(ts, 'x') + u.bbox.left / pxRatio;
                    const y = u.valToPos(ct, 'y') + u.bbox.top / pxRatio;
                    // Clamp x so the tooltip stays inside the chart bounds.
                    const rect = els.chartContainer.getBoundingClientRect();
                    const ttRect = tt.getBoundingClientRect();
                    const half = ttRect.width / 2;
                    const clampedX = Math.max(half + 4, Math.min(rect.width - half - 4, x));
                    tt.style.left = clampedX + 'px';
                    tt.style.top = y + 'px';
                    tt.classList.add('visible');
                }
            ],
            setSeries: [
                (u, _seriesIdx, opts) => {
                    if (opts && opts.show === false && chartTooltip) {
                        chartTooltip.classList.remove('visible');
                    }
                }
            ],
        },
    };

    chartInstance = new uPlot(opts, [xs, ys], els.chartContainer);
    renderEventChips(chartInstance);
}

els.chartContainer.addEventListener('mouseleave', () => {
    if (chartTooltip) chartTooltip.classList.remove('visible');
});

function resizeChart() {
    if (!chartInstance) return;
    const rect = els.chartContainer.getBoundingClientRect();
    chartInstance.setSize({ width: rect.width, height: rect.height });
    renderEventChips(chartInstance);
}

// Picker state. dailyFetchSeq protects against out-of-order
// responses when the user clicks several buttons quickly: only
// the latest request's render is allowed to win.
const DEFAULT_DAYS = 30;
let currentDays = DEFAULT_DAYS;
let dailyFetchSeq = 0;

async function loadDaily(days) {
    const seq = ++dailyFetchSeq;
    try {
        const r = await fetch('/api/v1/stats/reversals/daily?days=' + days);
        if (seq !== dailyFetchSeq) return;
        if (!r.ok) throw new Error('daily ' + r.status);
        const json = await r.json();
        if (seq !== dailyFetchSeq) return;
        renderChart(json.data || [], days);
    } catch (err) {
        if (seq !== dailyFetchSeq) return;
        console.error('daily load failed:', err);
        if (chartInstance) {
            chartInstance.destroy();
            chartInstance = null;
        }
        chartTooltip = null;
        els.chartContainer.classList.add('is-error');
        els.chartContainer.classList.remove('is-empty');
        els.chartContainer.textContent = 'Could not load chart.';
    }
}

function wirePeriodPicker() {
    const buttons = document.querySelectorAll('.period-btn');
    buttons.forEach(btn => {
        btn.addEventListener('click', () => {
            const days = parseInt(btn.dataset.days, 10);
            if (!Number.isFinite(days) || days === currentDays) return;
            currentDays = days;
            buttons.forEach(b => {
                const active = b === btn;
                b.classList.toggle('active', active);
                b.setAttribute('aria-pressed', active ? 'true' : 'false');
            });
            loadDaily(days);
        });
    });
}

// The /recent endpoint already returns up to 100 rows in one call,
// so "Load More" is purely client-side reveal (+10 rows per click)
// until all fetched rows are visible. True server-side pagination
// is deferred to v1.1 (PRD §6.4).
const PAGE_SIZE = 10;
const recentState = {
    rows: [],
    shown: 0,
};

async function loadRecent() {
    try {
        const r = await fetch('/api/v1/reversals/recent?limit=100');
        if (!r.ok) throw new Error('recent ' + r.status);
        const json = await r.json();
        recentState.rows = json.data || [];
        recentState.shown = 0;
        renderInitialReversals();
    } catch (err) {
        console.error('recent load failed:', err);
        els.reversalsBody.innerHTML =
            '<tr><td colspan="3" class="table-empty">Could not load reversals.</td></tr>';
        els.loadMoreBtn.hidden = true;
    }
}

function renderInitialReversals() {
    els.reversalsBody.innerHTML = '';
    recentState.shown = 0;

    if (!recentState.rows.length) {
        els.reversalsBody.innerHTML =
            '<tr><td colspan="3" class="table-empty">No reversals reported yet.</td></tr>';
        els.loadMoreBtn.hidden = true;
        return;
    }

    appendNextPage();
}

function appendNextPage() {
    const start = recentState.shown;
    const end = Math.min(start + PAGE_SIZE, recentState.rows.length);
    if (start >= end) return;

    const frag = document.createDocumentFragment();
    for (let i = start; i < end; i++) {
        frag.appendChild(buildReversalRow(recentState.rows[i]));
    }
    els.reversalsBody.appendChild(frag);
    recentState.shown = end;

    els.loadMoreBtn.hidden = recentState.shown >= recentState.rows.length;
}

function buildReversalRow(row) {
    const tr = document.createElement('tr');

    const tdTrader = document.createElement('td');
    tdTrader.className = 'col-trader';
    tdTrader.innerHTML = `
        <span class="trader-cell">
            <span class="trader-avatar" aria-hidden="true">
                <span class="material-symbols-outlined">person</span>
            </span>
            <span class="trader-name"></span>
        </span>`;
    // steam_id is a uint64 string from the API; keep it as a string
    // so the full precision survives (never parse it as a Number).
    const steamId = String(row.steam_id);
    tdTrader.querySelector('.trader-name').textContent = steamId;

    const tdId = document.createElement('td');
    tdId.className = 'col-steam-id';
    const span = document.createElement('span');
    span.className = 'steam-id-mono';
    span.textContent = steamId;
    tdId.appendChild(span);

    const tdDate = document.createElement('td');
    tdDate.className = 'col-date';
    // PRD §6.4: order by created_at DESC; "Date Added" semantic.
    tdDate.textContent = formatDate(row.created_at);

    tr.appendChild(tdTrader);
    tr.appendChild(tdId);
    tr.appendChild(tdDate);
    return tr;
}

els.loadMoreBtn.addEventListener('click', () => {
    appendNextPage();
});

function clearChip() {
    els.resultChip.classList.remove('visible', 'flagged', 'clear');
    document.body.classList.remove('flagged', 'clear-result');
}

function showError(msg) {
    els.searchError.textContent = msg;
    els.searchError.classList.add('visible');
}

function clearError() {
    els.searchError.classList.remove('visible');
    els.searchError.textContent = '';
}

function showChip({ steamId, flagged, lastReversalMs }) {
    // Steam IDs are uint64 and exceed Number.MAX_SAFE_INTEGER, so
    // they must stay strings end-to-end — never coerced through
    // Number/parseInt — or the profile links lose precision.
    const id = String(steamId);
    els.chipName.textContent = id;
    els.chipSteamLink.href = 'https://steamcommunity.com/profiles/' + encodeURIComponent(id);
    els.chipCsfloatLink.href = 'https://csfloat.com/stall/' + encodeURIComponent(id);
    els.chipWhen.textContent = flagged && lastReversalMs
        ? 'Last reversal ' + formatDate(lastReversalMs)
        : '';

    if (flagged) {
        els.resultChip.classList.remove('clear');
        els.resultChip.classList.add('flagged', 'visible');
        els.chipVerdict.innerHTML =
            '<span class="material-symbols-outlined">priority_high</span>Reversals Found';
        document.body.classList.add('flagged');
        document.body.classList.remove('clear-result');
    } else {
        els.resultChip.classList.remove('flagged');
        els.resultChip.classList.add('clear', 'visible');
        els.chipVerdict.innerHTML =
            '<span class="material-symbols-outlined">check</span>No Reversals Found';
        document.body.classList.add('clear-result');
        document.body.classList.remove('flagged');
    }
}

els.searchForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    clearError();
    clearChip();

    const steamId = els.steamIdInput.value.trim();
    if (!steamId) return;

    els.searchBtn.disabled = true;
    els.searchBtnIcon.outerHTML = '<span class="loading" id="searchBtnIcon"></span>';

    try {
        const r = await fetch('/api/v1/users/' + encodeURIComponent(steamId));
        if (!r.ok) {
            let msg = 'Lookup failed (' + r.status + ').';
            try {
                const err = await r.json();
                if (err && (err.message || err.code)) {
                    msg = [err.code, err.message].filter(Boolean).join(' — ');
                    if (err.details) msg += ' (' + err.details + ')';
                }
            } catch (_) {}
            throw new Error(msg);
        }
        const data = await r.json();
        showChip({
            steamId: data.steam_id,
            flagged: !!data.has_reversed,
            lastReversalMs: data.last_reversal_timestamp || 0,
        });
    } catch (err) {
        showError(err.message || 'Lookup failed.');
    } finally {
        els.searchBtn.disabled = false;
        const icon = document.createElement('span');
        icon.className = 'material-symbols-outlined';
        icon.id = 'searchBtnIcon';
        icon.textContent = 'arrow_forward';
        const existing = document.getElementById('searchBtnIcon');
        if (existing) existing.replaceWith(icon);
        els.searchBtnIcon = icon;
    }
});

els.steamIdInput.addEventListener('input', () => {
    clearError();
    if (els.resultChip.classList.contains('visible')) clearChip();
});

window.addEventListener('resize', resizeChart);
wirePeriodPicker();

loadSummary();
loadDaily(currentDays);
loadRecent();
