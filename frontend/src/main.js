import { GetTimers, AddTimer, UpdateTimer, RemoveTimer, GetAppInfo } from '../wailsjs/go/app/App';
import { EventsOn, BrowserOpenURL } from '../wailsjs/runtime/runtime';

const timerRows = document.getElementById('timer-rows');
const emptyState = document.getElementById('empty-state');
const addTimerBtn = document.getElementById('add-timer-btn');
const zoomOutBtn = document.getElementById('zoom-out-btn');
const zoomInBtn = document.getElementById('zoom-in-btn');

const themeSwitchBtns = document.querySelectorAll('.theme-switch-btn');
const aboutBtn = document.getElementById('about-btn');
const aboutBackdrop = document.getElementById('about-backdrop');
const aboutCloseBtn = document.getElementById('about-close-btn');
const aboutBody = document.getElementById('about-body');
const aboutRepoLink = document.getElementById('about-repo-link');

const modalBackdrop = document.getElementById('modal-backdrop');
const modalTitle = document.getElementById('modal-title');
const modalCloseBtn = document.getElementById('modal-close-btn');
const modalSubmitBtn = document.getElementById('modal-submit-btn');
const timerForm = document.getElementById('timer-form');
const nameInput = document.getElementById('timer-name');
const targetInput = document.getElementById('timer-target');
const formError = document.getElementById('form-error');

// editingID is null while adding a new timer, or the ID of the timer being
// edited — the one thing that distinguishes the two modes the shared modal
// supports (see openAdd/openEdit and the form's submit handler below).
let editingID = null;

// latestTimers holds the last list received from Go (id/name/target only —
// see TimerView on the Go side). The backend only pushes this on an actual
// Add/Update/Remove, not once a second: each entry's target is a plain
// unix-millis timestamp, so the day/hour/minute/second breakdown below is
// recomputed locally on every tick() instead of round-tripping through the
// WebView bridge every second just to redraw numbers JS can derive itself.
let latestTimers = [];

function breakdown(targetMs, now) {
    const remainingMs = targetMs - now;
    const expired = remainingMs <= 0;
    const totalSeconds = expired ? 0 : Math.floor(remainingMs / 1000);
    return {
        expired,
        days: Math.floor(totalSeconds / 86400),
        hours: Math.floor((totalSeconds % 86400) / 3600),
        minutes: Math.floor((totalSeconds % 3600) / 60),
        seconds: totalSeconds % 60,
    };
}

function tick() {
    const now = Date.now();
    render(latestTimers.map((t) => ({ ...t, ...breakdown(t.target, now) })));
}

function setTimers(timers) {
    latestTimers = timers;
    tick();
}

function render(timers) {
    timerRows.innerHTML = '';
    emptyState.hidden = timers.length > 0;

    for (const t of timers) {
        const tr = document.createElement('tr');
        if (t.expired) tr.classList.add('expired');

        tr.appendChild(cell(t.name));
        tr.appendChild(cell(t.days));
        tr.appendChild(cell(t.hours));
        tr.appendChild(cell(t.minutes));
        tr.appendChild(cell(t.seconds));

        // The flex row of buttons is a nested <div>, not styling on the <td>
        // itself: setting display:flex directly on a table cell overrides
        // its default display:table-cell, which breaks it out of the
        // table's row layout (in testing, this split each row's Actions
        // cell onto its own visual band below the rest of that row).
        const actionsCell = document.createElement('td');
        const actions = document.createElement('div');
        actions.className = 'actions-cell';

        const editBtn = document.createElement('button');
        editBtn.className = 'icon-btn';
        editBtn.type = 'button';
        editBtn.textContent = '✎';
        editBtn.setAttribute('aria-label', 'Edit timer');
        editBtn.addEventListener('click', () => openEdit(t));

        const removeBtn = document.createElement('button');
        removeBtn.className = 'icon-btn danger';
        removeBtn.type = 'button';
        removeBtn.textContent = '✕';
        removeBtn.setAttribute('aria-label', 'Remove timer');
        removeBtn.addEventListener('click', () => removeTimer(t));

        actions.appendChild(editBtn);
        actions.appendChild(removeBtn);
        actionsCell.appendChild(actions);
        tr.appendChild(actionsCell);

        timerRows.appendChild(tr);
    }
}

function cell(text) {
    const td = document.createElement('td');
    td.textContent = text;
    return td;
}

async function removeTimer(t) {
    const name = t.name || 'this timer';
    if (!confirm(`Remove "${name}"?`)) return;
    try {
        await RemoveTimer(t.id);
    } catch (err) {
        alert(`Could not remove timer: ${err}`);
    }
}

// toLocalInputValue formats a Date the way <input type="datetime-local">
// expects: local time, no timezone suffix.
function toLocalInputValue(date) {
    const pad = (n) => String(n).padStart(2, '0');
    return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

function showError(message) {
    formError.textContent = message;
    formError.hidden = false;
}

function clearError() {
    formError.hidden = true;
    formError.textContent = '';
}

function openModal() {
    modalBackdrop.hidden = false;
    nameInput.focus();
}

function closeModal() {
    modalBackdrop.hidden = true;
    clearError();
}

function openAdd() {
    editingID = null;
    modalTitle.textContent = 'Add Timer';
    modalSubmitBtn.textContent = 'Add Timer';
    nameInput.value = '';
    targetInput.value = toLocalInputValue(new Date(Date.now() + 60 * 60 * 1000));
    openModal();
}

function openEdit(t) {
    editingID = t.id;
    modalTitle.textContent = 'Edit Timer';
    modalSubmitBtn.textContent = 'Save';
    nameInput.value = t.name;
    targetInput.value = toLocalInputValue(new Date(t.target));
    openModal();
}

// Font-size zoom. --font-scale (see style.css) drives every rem-based size
// in the page, so this only ever needs to touch the one variable; unlike
// the Fyne version, no layout needs manual recomputation for it — the page
// just reflows on its own.
const FONT_SCALE_MIN = 0.7;
const FONT_SCALE_MAX = 2.5;
const FONT_SCALE_STEP = 0.1;
const FONT_SCALE_STORAGE_KEY = 'fontScale';

function clampFontScale(scale) {
    return Math.min(FONT_SCALE_MAX, Math.max(FONT_SCALE_MIN, scale));
}

function applyFontScale(scale) {
    scale = clampFontScale(scale);
    document.documentElement.style.setProperty('--font-scale', scale);
    try {
        localStorage.setItem(FONT_SCALE_STORAGE_KEY, String(scale));
    } catch {
        // Storage can be unavailable (e.g. a private/locked-down webview
        // profile) — losing the saved zoom level across restarts isn't
        // worth failing over.
    }
}

function currentFontScale() {
    const stored = parseFloat(localStorage.getItem(FONT_SCALE_STORAGE_KEY));
    return Number.isFinite(stored) ? stored : 1;
}

zoomOutBtn.addEventListener('click', () => applyFontScale(currentFontScale() - FONT_SCALE_STEP));
zoomInBtn.addEventListener('click', () => applyFontScale(currentFontScale() + FONT_SCALE_STEP));
applyFontScale(currentFontScale());

// Theme switch: light / dark / system. "system" removes data-theme so the
// plain @media (prefers-color-scheme) rule in style.css takes over; "light"
// and "dark" set data-theme explicitly, overriding the OS regardless of its
// own setting (see the :not([data-theme="light"]) guard in style.css).
const THEME_STORAGE_KEY = 'theme';

function storedTheme() {
    const stored = localStorage.getItem(THEME_STORAGE_KEY);
    return stored === 'light' || stored === 'dark' ? stored : 'system';
}

function applyTheme(theme) {
    if (theme === 'light' || theme === 'dark') {
        document.documentElement.setAttribute('data-theme', theme);
    } else {
        document.documentElement.removeAttribute('data-theme');
        theme = 'system';
    }
    try {
        localStorage.setItem(THEME_STORAGE_KEY, theme);
    } catch {
        // Same reasoning as the font-scale save above: losing the saved
        // choice across restarts isn't worth failing over.
    }
    for (const btn of themeSwitchBtns) {
        btn.classList.toggle('active', btn.dataset.themeValue === theme);
    }
}

for (const btn of themeSwitchBtns) {
    btn.addEventListener('click', () => applyTheme(btn.dataset.themeValue));
}

applyTheme(storedTheme());

// About dialog. Info itself is fetched once per open rather than cached —
// it's a single cheap call, and refetching means a change on the Go side
// (e.g. Version) shows up without needing a page reload.
function field(dl, term, value) {
    const dt = document.createElement('dt');
    dt.textContent = term;
    const dd = document.createElement('dd');
    dd.textContent = value;
    dl.appendChild(dt);
    dl.appendChild(dd);
}

async function openAbout() {
    aboutBody.innerHTML = '';
    const info = await GetAppInfo();
    field(aboutBody, 'Version', info.version);
    field(aboutBody, 'Author', info.author);
    field(aboutBody, 'License', `${info.license} © ${info.year}`);
    aboutRepoLink.onclick = () => BrowserOpenURL(info.repoUrl);
    aboutBackdrop.hidden = false;
}

function closeAbout() {
    aboutBackdrop.hidden = true;
}

aboutBtn.addEventListener('click', openAbout);
aboutCloseBtn.addEventListener('click', closeAbout);
aboutBackdrop.addEventListener('click', (ev) => {
    if (ev.target === aboutBackdrop) closeAbout();
});

addTimerBtn.addEventListener('click', openAdd);
modalCloseBtn.addEventListener('click', closeModal);

modalBackdrop.addEventListener('click', (ev) => {
    if (ev.target === modalBackdrop) closeModal();
});

document.addEventListener('keydown', (ev) => {
    if (ev.key !== 'Escape') return;
    if (!modalBackdrop.hidden) closeModal();
    if (!aboutBackdrop.hidden) closeAbout();
});

timerForm.addEventListener('submit', async (ev) => {
    ev.preventDefault();
    clearError();

    if (!targetInput.value) {
        showError('Please choose a date and time.');
        return;
    }
    const targetMs = new Date(targetInput.value).getTime();
    if (Number.isNaN(targetMs)) {
        showError('Invalid date/time.');
        return;
    }

    try {
        if (editingID) {
            await UpdateTimer(editingID, nameInput.value, targetMs);
        } else {
            await AddTimer(nameInput.value, targetMs);
        }
        closeModal();
    } catch (err) {
        showError(String(err));
    }
});

EventsOn('timers:updated', setTimers);
GetTimers().then(setTimers);
setInterval(tick, 1000);
