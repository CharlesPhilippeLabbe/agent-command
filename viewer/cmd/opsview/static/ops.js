// Ops page (ops.html): every operation's board, plus a panel showing one
// order or plan, selected by
// /ops.html?order=<order path>[&report=<report path>]|plan=<plan path>
// [&all=1][&project=<name>]. Loaded after app.js.

// ---- page state in the URL ---------------------------------------------------

function currentPanel() { return param("order") || param("plan"); }

// The current URL with params changed; an empty value removes the param.
function urlWith(params) {
    const url = new URL(location.href);
    for (const [k, v] of Object.entries(params)) {
        if (v) url.searchParams.set(k, v);
        else url.searchParams.delete(k);
    }
    return url;
}

// The board's filter, ?all=1 and ?project=<name>. The panel sends it too, so
// its prev/next links step through the same operations the board shows.
function filterQuery() {
    const q = new URLSearchParams();
    if (param("all")) q.set("all", "1");
    if (param("project")) q.set("project", param("project"));
    return q;
}

function opsURL() {
    const q = filterQuery();
    return "/api/ops" + (q.size ? "?" + q : "");
}

// The panel shows an order, one of its reports, or a plan; "" when nothing
// is selected.
function panelURL() {
    const q = filterQuery();
    if (param("order")) {
        q.set("path", param("order"));
        if (param("report")) q.set("report", param("report"));
        return "/api/order?" + q;
    }
    if (param("plan")) {
        q.set("path", param("plan"));
        return "/api/plan?" + q;
    }
    return "";
}

// #panel's htmx:configRequest: fetch the selection, or empty the panel when
// there is none. Requests from inside the panel (note forms and buttons)
// bubble here too; they keep their own path.
function panelRequest(evt, panel) {
    if (evt.detail.elt !== panel) return;
    const url = panelURL();
    if (url) {
        evt.detail.path = url;
    } else {
        evt.preventDefault();
        panel.replaceChildren();
    }
}

// Filters replace the history entry: they narrow the view, they don't move.
function setFilter(params) {
    history.replaceState(null, "", urlWith(params));
    reload("#ops");
}

// ---- panel -------------------------------------------------------------------

// Opening something new starts the panel at its top; a hot-reload re-render
// of the same thing keeps the scroll position.
let panelToTop = false;

// A panel link replaces the selection and keeps the filter.
function openPanel(href) {
    const q = new URL(href, location.href).searchParams;
    showPanel(urlWith({ order: q.get("order"), report: q.get("report"), plan: q.get("plan") }));
}

function closePanel() {
    showPanel(urlWith({ order: "", report: "", plan: "" }));
}

function showPanel(url) {
    history.pushState(null, "", url);
    panelToTop = true;
    reload("#panel");
    markActive();
}

// data-panel links switch the panel in place. data-doc links (other markdown
// inside a rendered order) stay plain links to the docs page.
document.addEventListener("click", (evt) => {
    const a = evt.target.closest("a[data-panel]");
    if (!a || !plainClick(evt)) return;
    evt.preventDefault();
    openPanel(a.href);
});
window.addEventListener("popstate", () => {
    reload("#panel");
    markActive();
});

// Panel width: "" (default), "half" or "full", set on <body> for app.css and
// remembered per browser.
let panelSizeNow = "";
try { panelSizeNow = localStorage.getItem("panelSize") || ""; } catch (err) {}

function setPanelSize(size) {
    panelSizeNow = size;
    try { size ? localStorage.setItem("panelSize", size) : localStorage.removeItem("panelSize"); } catch (err) {}
    applyPanelSize();
}
function applyPanelSize() {
    if (panelSizeNow) document.body.dataset.panelSize = panelSizeNow;
    else delete document.body.dataset.panelSize;
    markActive();
}
document.addEventListener("DOMContentLoaded", applyPanelSize);

// Bring every control in line with the URL and the panel size, after any
// render or navigation.
function markActive() {
    const selected = currentPanel();
    // app.css hides the board behind a full panel off this flag, not off
    // :has(~ #panel:not(:empty)), which Safari doesn't re-evaluate when the
    // panel's content changes.
    document.body.toggleAttribute("data-panel-open", !!selected);
    markEach("#ops [data-panel][data-path]", (el) => el.dataset.path === selected);
    markEach("#panel .panel-tabs a", (a) => a.dataset.tab === param("report"));
    markEach("#panel .sizes button", (b) => b.dataset.size === panelSizeNow);
    markEach("#panel .annotate", () => annotateOn);
    // Taps only add notes where the panel offers the toggle: not on a
    // closed operation.
    document.body.toggleAttribute("data-annotate", annotateOn && !!document.querySelector("#panel .annotate"));
    markFilter();
}

function markFilter() {
    document.querySelectorAll('#projects input[name="project"]').forEach((r) => (r.checked = r.value === param("project")));
    const closed = document.getElementById("show-closed");
    if (closed) closed.checked = !!param("all");
}

document.addEventListener("htmx:afterSwap", (evt) => {
    const target = evt.detail.target;
    if (target.id === "panel" && panelToTop) {
        target.scrollTop = 0;
        panelToTop = false;
    }
    if (target.id === "ops") observeBoards(target);
    if (target.id === "panel") placeNotes(target);
    markActive();
});

// ---- boards: dependency lines ------------------------------------------------

const SVG = "http://www.w3.org/2000/svg";

// One curve per dependency, from the dependency's card to the dependent card.
function drawLinks(board) {
    const svg = board.querySelector("svg.links");
    if (!svg) return;
    // Collapse the old svg first so it doesn't hold scrollWidth open.
    sizeSvg(svg, 0, 0);
    svg.replaceChildren();
    sizeSvg(svg, board.scrollWidth, board.scrollHeight);

    const bend = parseFloat(getComputedStyle(board).getPropertyValue("--link-bend")) || 0;
    const cards = new Map([...board.querySelectorAll(".card")].map((c) => [c.dataset.order, c]));
    for (const card of cards.values()) {
        for (const d of card.dataset.depends.split(",").filter(Boolean)) {
            const from = cards.get(d);
            if (from) svg.appendChild(linkPath(board, from, card, bend));
        }
    }
}

function sizeSvg(svg, width, height) {
    svg.setAttribute("width", width);
    svg.setAttribute("height", height);
}

// A curve from the right edge of card `from` to the left edge of card `to`,
// leaving and entering level, at least `bend` wide.
function linkPath(board, from, to, bend) {
    const a = cardEdges(board, from), b = cardEdges(board, to);
    const dx = Math.max(bend, (b.l - a.r) / 2);
    const path = document.createElementNS(SVG, "path");
    path.setAttribute("d", `M${a.r},${a.y} C${a.r + dx},${a.y} ${b.l - dx},${b.y} ${b.l},${b.y}`);
    path.setAttribute("class", "link" + (from.classList.contains("status-done") ? " done" : ""));
    path.dataset.from = from.dataset.order;
    path.dataset.to = to.dataset.order;
    return path;
}

// A card's left and right edges and middle height, in the board's scroll
// coordinates.
function cardEdges(board, card) {
    const b = board.getBoundingClientRect(), r = card.getBoundingClientRect();
    const left = r.left - b.left + board.scrollLeft;
    return { l: left, r: left + r.width, y: r.top - b.top + board.scrollTop + r.height / 2 };
}

// Boards change size when the window resizes or the panel opens/closes;
// redraw whenever one does. Observing also draws once on swap-in.
const boardObserver = new ResizeObserver((entries) => entries.forEach((e) => drawLinks(e.target)));

function observeBoards(root) {
    boardObserver.disconnect();
    root.querySelectorAll(".board").forEach((b) => boardObserver.observe(b));
}

// Web fonts change card sizes once they load.
document.fonts?.ready.then(() => document.querySelectorAll(".board").forEach(drawLinks));

// Hovering a card highlights it, its lines, and the cards on the other end.
function highlight(card, on) {
    const board = card.closest(".board");
    const n = card.dataset.order;
    board.classList.toggle("focus", on);
    card.classList.toggle("hl", on);
    board.querySelectorAll(`.link[data-from="${n}"], .link[data-to="${n}"]`).forEach((p) => {
        p.classList.toggle("hl", on);
        const other = p.dataset.from === n ? p.dataset.to : p.dataset.from;
        board.querySelector(`.card[data-order="${other}"]`)?.classList.toggle("hl", on);
    });
}

// mouseover and mouseout bubble from the card's children too; only entering
// or leaving the card itself counts.
function hoverCard(on) {
    return (evt) => {
        const card = evt.target.closest(".card");
        if (card && !card.contains(evt.relatedTarget)) highlight(card, on);
    };
}
document.addEventListener("mouseover", hoverCard(true));
document.addEventListener("mouseout", hoverCard(false));

// ---- notes -------------------------------------------------------------------

// Annotate mode: on, a tap on a block of the panel's document opens a note
// form under it. Remembered per browser, like the panel size.
let annotateOn = false;
try { annotateOn = localStorage.getItem("annotate") === "1"; } catch (err) {}

function toggleAnnotate() {
    annotateOn = !annotateOn;
    try { annotateOn ? localStorage.setItem("annotate", "1") : localStorage.removeItem("annotate"); } catch (err) {}
    markActive();
}

// After each panel render, move every open note's card from the hidden
// holder to the block it is about: the first block of the document whose
// first line is the note's line. No such block: the detached tray.
function placeNotes(panel) {
    const article = panel.querySelector("article[data-path]");
    const tray = panel.querySelector(".notes-detached");
    panel.querySelectorAll(".notes-holder .note-card").forEach((card) => {
        const block = article?.querySelector(`[data-line="${card.dataset.noteLine}"]`);
        if (block) placeNote(block, card);
        else tray.appendChild(card);
    });
}

// placeNote puts el, a note card or form, under block, after any notes
// already there: in a row of its own after a table row, inside a list item,
// else right after the block.
function placeNote(block, el) {
    if (block.tagName === "TR") {
        const row = document.createElement("tr");
        row.className = "note-row";
        const cell = row.insertCell();
        cell.colSpan = block.cells.length;
        cell.appendChild(el);
        let after = block;
        while (after.nextElementSibling?.classList.contains("note-row")) after = after.nextElementSibling;
        after.after(row);
    } else if (block.tagName === "LI") {
        block.appendChild(el);
    } else {
        let after = block;
        while (after.nextElementSibling?.matches(".note-card, .note-form")) after = after.nextElementSibling;
        after.after(el);
    }
}

// noteForm renders the note-form template; data is the tapped block's
// target, line and end, or for an edit the note's path and text.
function noteForm(data) {
    const wrap = document.createElement("div");
    wrap.innerHTML = Mustache.render(document.getElementById("note-form").textContent, data);
    return wrap.firstElementChild;
}

// A tap on a block, annotate on: a new-note form under it. One at a time:
// a form holding text stays put, an empty one moves to the new block. Taps
// on links, on notes, or ending a text selection are left alone.
document.addEventListener("click", (evt) => {
    if (!document.body.hasAttribute("data-annotate")) return;
    const article = evt.target.closest("#panel article[data-path]");
    if (!article || evt.target.closest("a, .note-card, .note-form, .note-row")) return;
    if (String(getSelection())) return;
    const block = evt.target.closest("[data-line]");
    if (!block || !article.contains(block)) return;

    const open = article.querySelector(".note-form.new");
    if (open) {
        if (open.querySelector("textarea").value.trim()) return;
        closeNoteForm(open);
    }
    const form = noteForm({ target: article.dataset.path, line: block.dataset.line, end: block.dataset.end });
    form.classList.add("new");
    placeNote(block, form);
    htmx.process(form);
    form.querySelector("textarea").focus();
});

// Edit: the card makes way for a form holding its text; cancel brings it back.
function editNote(button) {
    const card = button.closest(".note-card");
    const form = noteForm({ path: card.dataset.note, text: card.querySelector(".note-text").textContent });
    form.classList.add("edit");
    card.hidden = true;
    card.after(form);
    htmx.process(form);
    form.querySelector("textarea").focus();
}

// closeNoteForm removes a note form (and the table row it sat in, when
// empty), shows the card an edit hid, and lets a held hot reload run.
function closeNoteForm(form) {
    if (form.classList.contains("edit")) form.previousElementSibling.hidden = false;
    const row = form.closest(".note-row");
    form.remove();
    if (row && !row.querySelector(".note-card, .note-form")) row.remove();
    releaseReload();
}

// A saved note closes its form; the server's ping then reloads the panel.
// Once the form is gone htmx fires afterRequest again on a parent, with the
// same detail: only the event on the form itself counts, or the form would
// close twice.
document.addEventListener("htmx:afterRequest", (evt) => {
    const form = evt.detail.elt;
    if (evt.target !== form) return;
    if (evt.detail.successful && form.matches?.(".note-form")) closeNoteForm(form);
});
