/**
 * mustache-tmpl — a small htmx extension for a JSON-in / Mustache-out workflow:
 * the server answers JSON only and the page owns all markup.
 *
 * REQUESTS
 *   Any element using this extension sends its parameters as a JSON body
 *   instead of form-urlencoded.
 *
 * RESPONSES
 *   If the requesting element has a `mustache-tmpl="<id>"` attribute, the JSON
 *   response is parsed and rendered through <script type="text/x-template"
 *   id="<id>"> using Mustache. Every such script tag on the page is registered
 *   as a Mustache partial keyed by its own id.
 *
 *   opsview addition: every render also sees a `markdown` lambda, so a
 *   template turns a raw markdown field into HTML with
 *   {{#markdown}}{{{body}}}{{/markdown}}, and an `annotated` lambda that
 *   does the same and marks which file lines each block came from.
 */

// The markdown lambda: render the section (the raw body), then parse it.
const markdown = () => (text, render) => marked.parse(render(text));

// The annotated lambda: like markdown, but every block a note can sit on
// carries data-line and data-end, its first and last line in the source
// file. Lines count from the rendered data's bodyLine, the file line the
// body starts at, so the server can quote exactly those lines.
const annotated = function () {
    const bodyLine = this.bodyLine || 1;
    return (text, render) => {
        const tokens = annotator.lexer(render(text));
        tagLines(tokens, bodyLine);
        return annotator.parser(tokens);
    };
};

// tagLines gives each block token its first and last file line, starting at
// line: the tokens' raw texts follow one another, newlines included. Nested
// blocks (list items, what a list item or blockquote holds) count from
// their parent's first line the same way.
function tagLines(tokens, line) {
    for (const t of tokens) {
        t.line = line;
        t.end = line + t.raw.replace(/\n+$/, "").split("\n").length - 1;
        if (t.items) tagLines(t.items, line);
        else if (t.tokens && (t.type === "list_item" || t.type === "blockquote")) tagLines(t.tokens, line);
        line += t.raw.split("\n").length - 1;
    }
}

// lineAttrs puts data-line and data-end on the first tag of html.
function lineAttrs(html, token) {
    if (token.line === undefined) return html;
    return html.replace(/^(\s*<[a-z0-9]+)/i, `$1 data-line="${token.line}" data-end="${token.end}"`);
}

// withLines wraps marked's default renderer method name: it renders as
// marked does, then adds the line attributes.
function withLines(name) {
    return function (token) { return lineAttrs(marked.Renderer.prototype[name].call(this, token), token); };
}

// A marked of its own, so the docs page's `marked` stays untouched. A table
// row has no token of its own: the header row is the table's first line,
// body row i is line + 2 + i (the separator line sits between).
const annotator = new marked.Marked({
    renderer: {
        heading: withLines("heading"),
        paragraph: withLines("paragraph"),
        listitem: withLines("listitem"),
        code: withLines("code"),
        blockquote: withLines("blockquote"),
        table(token) {
            let row = 0;
            return marked.Renderer.prototype.table.call(this, token).replace(/<tr>/g, () => {
                const line = row === 0 ? token.line : token.line + 1 + row;
                row++;
                return `<tr data-line="${line}" data-end="${line}">`;
            });
        },
    },
});

htmx.defineExtension("mustache-tmpl", {
    onEvent: function(name, evt) {
        if (name === "htmx:configRequest") {
            evt.detail.headers["Content-Type"] = "application/json";
        }
        return true;
    },

    encodeParameters: function(xhr, parameters) {
        xhr.overrideMimeType("text/json");
        // htmx 2 passes a FormData, which JSON.stringify turns into "{}".
        return JSON.stringify(Object.fromEntries(parameters));
    },

    transformResponse: function(text, xhr, elt) {
        const templateId = elt.getAttribute("mustache-tmpl");
        if (!templateId) return text;

        const templateEl = document.getElementById(templateId);
        if (!templateEl) {
            console.error(`mustache-tmpl: no template found with id="${templateId}"`);
            return text;
        }

        let data;
        try {
            data = text ? JSON.parse(text) : {};
        } catch (err) {
            console.error(`mustache-tmpl: response for #${templateId} was not JSON`, err);
            return text;
        }
        return Mustache.render(templateEl.textContent, Object.assign({ markdown, annotated }, data), templatePartials());
    },
});

// Every template on the page, keyed by id, so any template can include any
// other as {{> id}}.
function templatePartials() {
    const partials = {};
    document.querySelectorAll('script[type="text/x-template"][id]').forEach((t) => (partials[t.id] = t.textContent));
    return partials;
}

// ---- shared by every page ----------------------------------------------

// Theme: "light" or "dark" on <html data-theme>, read by app.css. A choice
// made with the toggle is remembered per browser; without one the theme
// follows the system setting, live. This runs from <head>, so the first
// paint already has the right colours.
const darkQuery = matchMedia("(prefers-color-scheme: dark)");
let themeChoice = "";
try { themeChoice = localStorage.getItem("theme") || ""; } catch (err) {}

function applyTheme() {
    document.documentElement.dataset.theme = themeChoice || (darkQuery.matches ? "dark" : "light");
}
function toggleTheme() {
    themeChoice = document.documentElement.dataset.theme === "dark" ? "light" : "dark";
    try { localStorage.setItem("theme", themeChoice); } catch (err) {}
    applyTheme();
}
darkQuery.addEventListener("change", applyTheme);
applyTheme();
// Page scripts (docs.js, ops.js) load after this one and add their own
// click, popstate and swap handlers.

// Page state lives in the URL; mount points read it on every load/reload.
const param = (name) => new URLSearchParams(location.search).get(name) || "";

// Fire the "reload" trigger on a mount point, if this page has it.
function reload(selector) {
    const el = document.querySelector(selector);
    if (el) htmx.trigger(el, "reload");
}

// A plain left click, the kind that may switch content in place; any modifier
// keeps the browser's own behaviour (new tab, new window).
function plainClick(evt) {
    return evt.button === 0 && !evt.metaKey && !evt.ctrlKey && !evt.shiftKey;
}

// Toggle the "active" class on every element matching selector: on where
// test(el) holds, off elsewhere.
function markEach(selector, test) {
    document.querySelectorAll(selector).forEach((el) => el.classList.toggle("active", test(el)));
}

// Links and images inside a rendered doc are relative to the doc's own file.
function rewriteLinks(article, docPath) {
    const base = new URL(docPath, "http://doc/");
    article.querySelectorAll("a[href]").forEach((a) => {
        const ref = a.getAttribute("href");
        if (isExternal(ref)) return;
        const { path, hash } = resolveRef(ref, base);
        const { href, attr } = linkTarget(path, hash);
        a.href = href;
        if (attr) a.setAttribute(attr, "");
    });
    article.querySelectorAll("img[src]").forEach((img) => {
        const ref = img.getAttribute("src");
        if (!isExternal(ref)) img.src = "/files/" + resolveRef(ref, base).path;
    });
}

// A reference that doesn't point at a file next to the doc: a URL, an
// anchor or an absolute path.
function isExternal(ref) {
    return !ref || /^[a-z][a-z0-9+.-]*:/i.test(ref) || ref.startsWith("#") || ref.startsWith("/");
}

// The served path and hash of ref, read relative to the doc's URL base.
function resolveRef(ref, base) {
    const u = new URL(ref, base);
    return { path: decodeURIComponent(u.pathname.slice(1)), hash: u.hash };
}

// Where a doc's link to path should go: orders and plans open in the panel
// when the page has one, other markdown in the docs page, anything else is
// served from /files/. attr marks links the page switches in place.
function linkTarget(path, hash) {
    const kind = document.getElementById("panel") && panelKind(path);
    if (kind) return { href: `/ops.html?${kind}=${encodeURIComponent(path)}`, attr: "data-panel" };
    if (path.endsWith(".md")) return { href: "/?path=" + encodeURIComponent(path) + hash, attr: "data-doc" };
    return { href: "/files/" + path };
}

// Wrap each rendered table in a scroll container (styled in app.css).
function wrapTables(article) {
    article.querySelectorAll("table").forEach((t) => {
        const wrap = document.createElement("div");
        wrap.className = "table-scroll";
        t.replaceWith(wrap);
        wrap.appendChild(t);
    });
}

// A RISKS section gets a box of its own (.risks in app.css), so risks stand
// out in any doc.
function boxRisks(article) {
    article.querySelectorAll(":scope > h1").forEach((h) => {
        if (h.textContent.trim() === "RISKS") boxSection(h, "risks");
    });
}

// Move heading h and everything after it, up to the next h1, into a
// <section class="<cls>">.
function boxSection(h, cls) {
    const box = document.createElement("section");
    box.className = cls;
    h.before(box);
    for (let el = h; el && (el === h || el.tagName !== "H1"); ) {
        const next = el.nextElementSibling;
        box.appendChild(el);
        el = next;
    }
}

// "order" or "plan" for a path the ops panel can show, else "".
function panelKind(path) {
    if (/(^|\/)\.saved\/ops\/[^/]+\/orders\/[^/]+\.md$/.test(path)) return "order";
    if (/(^|\/)\.saved\/ops\/[^/]+\/plan\.md$/.test(path)) return "plan";
    return "";
}

// ---- swaps, errors, hot reload ---------------------------------------------

// Any swapped-in markdown article gets its links fixed, tables wrapped and
// risks boxed.
document.addEventListener("htmx:afterSwap", (evt) => {
    const article = evt.detail.target.querySelector("article[data-path]");
    if (article) {
        rewriteLinks(article, article.dataset.path);
        wrapTables(article);
        boxRisks(article);
    }
});

// A missing doc or order answers 404 with {state: "failed", error}; htmx
// doesn't swap errors, so show the message instead of the previous content.
// The message goes in the request's target, not the element that sent it:
// for a note form that is its status line, so the typed text stays.
document.addEventListener("htmx:responseError", (evt) => {
    if (!evt.detail.elt.hasAttribute("mustache-tmpl")) return;
    let msg = evt.detail.xhr.statusText;
    try { msg = JSON.parse(evt.detail.xhr.responseText).error || msg; } catch (err) {}
    const p = document.createElement("p");
    p.className = "error";
    p.textContent = msg;
    evt.detail.target.replaceChildren(p);
});

// ---- confirm dialog --------------------------------------------------------

// An element with the confirm-dialog attribute asks first: its request is
// held, #confirm-modal shows confirm-modal-tmpl rendered with the element's
// data-* values, and only its confirm button sends the request. htmx:confirm
// fires before every request, so the attribute is the gate.
document.addEventListener("htmx:confirm", (evt) => {
    const elt = evt.detail.elt;
    const dialog = document.getElementById("confirm-modal");
    const tmpl = document.getElementById("confirm-modal-tmpl");
    if (!elt.hasAttribute("confirm-dialog") || !dialog || !tmpl) return;
    evt.preventDefault();
    dialog.innerHTML = Mustache.render(tmpl.textContent, { ...elt.dataset });
    dialog.querySelector("[data-confirm]").addEventListener("click", () => {
        dialog.close();
        evt.detail.issueRequest(true); // skip htmx's own confirm, just send
    });
    dialog.querySelector("[data-close]").addEventListener("click", () => dialog.close());
    dialog.showModal();
});

// Backdrop click closes (native <dialog> doesn't). Esc already works.
document.addEventListener("click", (evt) => {
    const dialog = evt.target;
    if (dialog.id !== "confirm-modal") return;
    const r = dialog.getBoundingClientRect();
    const inside = evt.clientX >= r.left && evt.clientX <= r.right && evt.clientY >= r.top && evt.clientY <= r.bottom;
    if (!inside) dialog.close();
});

// Hot reload: the server pings on any change to a matched file, and every
// mount point on the page re-fetches. A reconnect means the server restarted
// (air rebuild), so the page itself may be stale. While #panel holds a note
// form with unsent text, its reload is held instead, so the draft survives
// an officer writing a report meanwhile; releaseReload() reloads it once the
// form is gone. Only a hot reload waits: navigating replaces the panel.
const events = new EventSource("/api/events");
let connected = false;
events.onopen = () => {
    if (connected) location.reload();
    connected = true;
};
let panelHeld = false;

function holdsDraft(el) {
    return [...el.querySelectorAll(".note-form textarea")].some((t) => t.value.trim());
}

function releaseReload() {
    const panel = document.getElementById("panel");
    if (panelHeld && panel && !holdsDraft(panel)) {
        panelHeld = false;
        htmx.trigger(panel, "reload");
    }
}

events.onmessage = () => {
    document.querySelectorAll('[hx-trigger*="reload"]').forEach((el) => {
        if (el.id === "panel" && holdsDraft(el)) panelHeld = true;
        else htmx.trigger(el, "reload");
    });
};
