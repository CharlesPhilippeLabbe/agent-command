// Docs page (index.html): the file tree and one rendered doc, selected by
// /?path=<doc>. Loaded after app.js.

// Sidebar: "collapsed" on <html data-nav>, read by app.css and remembered per
// browser. This runs from <head>, so the first paint is already right.
let navCollapsed = false;
try { navCollapsed = localStorage.getItem("navCollapsed") === "1"; } catch (err) {}

function applyNav() {
    if (navCollapsed) document.documentElement.dataset.nav = "collapsed";
    else delete document.documentElement.dataset.nav;
}
function toggleSidebar() {
    navCollapsed = !navCollapsed;
    try { navCollapsed ? localStorage.setItem("navCollapsed", "1") : localStorage.removeItem("navCollapsed"); } catch (err) {}
    applyNav();
}
applyNav();

function currentPath() { return param("path"); }

function docURL() { return "/api/doc?path=" + encodeURIComponent(currentPath()); }

// data-doc links switch the doc in place: push the URL, reload #doc.
document.addEventListener("click", (evt) => {
    const a = evt.target.closest("a[data-doc]");
    if (!a || !plainClick(evt)) return;
    evt.preventDefault();
    history.pushState(null, "", a.href);
    reload("#doc");
    window.scrollTo(0, 0);
    markActive();
});
window.addEventListener("popstate", () => {
    reload("#doc");
    markActive();
});

function markActive() {
    markEach("#nav a[data-doc]", (a) => (new URL(a.href).searchParams.get("path") || "") === currentPath());
}

// Keep the sidebar's scroll position across hot-reload re-renders.
let navScroll = 0;
document.addEventListener("htmx:beforeSwap", (evt) => {
    if (evt.detail.target.id === "nav") navScroll = evt.detail.target.scrollTop;
});

document.addEventListener("htmx:afterSwap", (evt) => {
    const target = evt.detail.target;
    if (target.id === "nav") target.scrollTop = navScroll;
    if (target.id === "doc") document.title = (currentPath() || "index") + " · opsview";
    markActive();
});
