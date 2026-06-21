// theme.js — light/dark toggle with persistence
(function () {
    var root = document.documentElement;
    var btn = document.getElementById("theme-toggle");

    function syncIcon() {
        if (!btn) return;
        var icon = btn.querySelector("i");
        var dark = root.getAttribute("data-theme") === "dark";
        if (icon) icon.className = dark ? "fa fa-sun" : "fa fa-moon";
    }

    function toggle() {
        var next = root.getAttribute("data-theme") === "dark" ? "light" : "dark";
        root.setAttribute("data-theme", next);
        localStorage.setItem("theme", next);
        syncIcon();
    }

    syncIcon();
    if (btn) btn.addEventListener("click", toggle);
})();
