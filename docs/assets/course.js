// Stagehand course — navigation, progress tracking and theme.
// Every page is plain HTML with a single <main class="content">; this script
// wraps it with the sidebar, the prev/next pager and per-lesson "done" toggles.
(function () {
  "use strict";

  const COURSE = [
    { group: "Start here", pages: [
      { file: "index.html", title: "Welcome & roadmap" },
      { file: "00-key-terms.html", title: "0 · Key terms in 10 min" },
    ]},
    { group: "Reference", pages: [
      { file: "01-the-domain.html", title: "1 · The domain" },
      { file: "02-ddd-building-blocks.html", title: "2 · DDD building blocks" },
      { file: "03-hexagonal-architecture.html", title: "3 · Hexagonal architecture" },
      { file: "04-events-and-messaging.html", title: "4 · Events & messaging" },
      { file: "05-testing-strategy.html", title: "5 · Testing strategy & TDD" },
      { file: "06-infrastructure.html", title: "6 · Infrastructure & tooling" },
      { file: "07-decisions.html", title: "7 · Decision log (ADRs)" },
    ]},
    { group: "Course", pages: [
      { file: "part-0-orientation.html", title: "Part 0 · Orientation & setup" },
      { file: "part-1-venue-domain.html", title: "Part 1 · Venue domain model" },
      { file: "part-2-venue-application.html", title: "Part 2 · Application layer" },
      { file: "part-3-persistence.html", title: "Part 3 · Persistence adapters" },
      { file: "part-4-driving-adapters.html", title: "Part 4 · HTTP & composition root" },
      { file: "part-5-messaging.html", title: "Part 5 · Outbox & Kafka" },
      { file: "part-6-show-context.html", title: "Part 6 · Show context" },
      { file: "part-7-ticketing-core.html", title: "Part 7 · Ticketing (core)" },
      { file: "part-8-end-to-end.html", title: "Part 8 · End-to-end & hardening" },
      { file: "part-9-stretch.html", title: "Part 9 · Stretch goals" },
    ]},
    { group: "Extension track (optional)", pages: [
      { file: "ext-0-overview.html", title: "Overview: toward production" },
      { file: "ext-1-pricing.html", title: "E1 · Pricing" },
      { file: "ext-2-identity.html", title: "E2 · Identity & access" },
      { file: "ext-3-observability.html", title: "E3 · Observability" },
      { file: "ext-4-audit.html", title: "E4 · Audit trail" },
      { file: "ext-5-payments.html", title: "E5 · Payments" },
      { file: "ext-6-fraud.html", title: "E6 · Fraud screening" },
      { file: "ext-decisions.html", title: "Extension ADRs" },
    ]},
  ];

  const STORE_KEY = "stagehand-course-progress-v1";
  const THEME_KEY = "stagehand-course-theme";

  function load(key, fallback) {
    try { const v = localStorage.getItem(key); return v ? JSON.parse(v) : fallback; }
    catch (_) { return fallback; }
  }
  function save(key, value) {
    try { localStorage.setItem(key, JSON.stringify(value)); } catch (_) { /* storage unavailable */ }
  }

  const flat = COURSE.flatMap(g => g.pages);
  const currentFile = (location.pathname.split("/").pop() || "index.html");
  const idx = Math.max(0, flat.findIndex(p => p.file === currentFile));
  const progress = load(STORE_KEY, {});           // { "part-1-venue-domain.html": ["l1-1", ...] }
  const lessonCounts = load(STORE_KEY + "-counts", {}); // { file: totalLessons }

  const theme = load(THEME_KEY, null);
  if (theme) document.documentElement.dataset.theme = theme;

  function el(tag, attrs, children) {
    const n = document.createElement(tag);
    for (const [k, v] of Object.entries(attrs || {})) {
      if (k === "class") n.className = v; else if (k === "text") n.textContent = v; else n.setAttribute(k, v);
    }
    (children || []).forEach(c => n.append(c));
    return n;
  }

  function buildSidebar() {
    const side = el("nav", { class: "sidebar", "aria-label": "Course navigation" });
    side.append(el("a", { class: "brand", href: "index.html" }, [
      "Stagehand", el("small", { text: "DDD · Hexagonal · TDD in Go" }),
    ]));
    COURSE.forEach(group => {
      side.append(el("h4", { text: group.group }));
      const ol = el("ol");
      group.pages.forEach(p => {
        const a = el("a", { href: p.file }, [el("span", { text: p.title })]);
        if (p.file === currentFile) { a.classList.add("current"); a.setAttribute("aria-current", "page"); }
        const total = lessonCounts[p.file];
        if (total) {
          const done = (progress[p.file] || []).length;
          a.append(el("span", { class: "count" + (done === total ? " complete" : ""), text: done + "/" + total }));
        }
        ol.append(el("li", {}, [a]));
      });
      side.append(ol);
    });
    const themeBtn = el("button", { type: "button", text: "Toggle theme" });
    themeBtn.addEventListener("click", () => {
      const dark = document.documentElement.dataset.theme === "dark" ||
        (!document.documentElement.dataset.theme && matchMedia("(prefers-color-scheme: dark)").matches);
      const next = dark ? "light" : "dark";
      document.documentElement.dataset.theme = next;
      save(THEME_KEY, next);
    });
    const resetBtn = el("button", { type: "button", text: "Reset progress" });
    resetBtn.addEventListener("click", () => {
      if (confirm("Clear all lesson progress?")) { save(STORE_KEY, {}); location.reload(); }
    });
    side.append(el("div", { class: "tools" }, [themeBtn, resetBtn]));
    return side;
  }

  function buildPager() {
    const pager = el("nav", { class: "pager", "aria-label": "Previous and next page" });
    const prev = flat[idx - 1], next = flat[idx + 1];
    pager.append(prev
      ? el("a", { class: "prev", href: prev.file }, [el("small", { text: "← Previous" }), prev.title])
      : el("span", { class: "spacer" }));
    pager.append(next
      ? el("a", { class: "next", href: next.file }, [el("small", { text: "Next →" }), next.title])
      : el("span", { class: "spacer" }));
    return pager;
  }

  function wireLessons(main) {
    const lessons = main.querySelectorAll("section.lesson[id]");
    if (!lessons.length) return;
    lessonCounts[currentFile] = lessons.length;
    save(STORE_KEY + "-counts", lessonCounts);
    const done = new Set(progress[currentFile] || []);
    lessons.forEach(sec => {
      const h3 = sec.querySelector("h3");
      if (!h3) return;
      const box = el("input", { type: "checkbox", "aria-label": "Mark lesson done" });
      box.checked = done.has(sec.id);
      sec.classList.toggle("done", box.checked);
      box.addEventListener("change", () => {
        box.checked ? done.add(sec.id) : done.delete(sec.id);
        sec.classList.toggle("done", box.checked);
        progress[currentFile] = [...done];
        save(STORE_KEY, progress);
        const count = document.querySelector(".sidebar a.current .count");
        if (count) {
          count.textContent = done.size + "/" + lessons.length;
          count.classList.toggle("complete", done.size === lessons.length);
        }
      });
      h3.append(el("label", { class: "done-toggle" }, [box, "done"]));
    });
  }

  function wrapTables(main) {
    main.querySelectorAll("table").forEach(t => {
      if (t.parentElement.classList.contains("table-wrap")) return;
      const w = el("div", { class: "table-wrap" });
      t.replaceWith(w); w.append(t);
    });
  }

  document.addEventListener("DOMContentLoaded", () => {
    const main = document.querySelector("main.content");
    if (!main) return;
    wrapTables(main);
    wireLessons(main);
    const layout = el("div", { class: "layout" });
    const side = buildSidebar();
    const toggle = el("button", { class: "menu-toggle", type: "button", text: "☰ Menu" });
    toggle.addEventListener("click", () => side.classList.toggle("open"));
    main.replaceWith(layout);
    main.append(buildPager());
    layout.append(side, el("div", { style: "flex:1;min-width:0" }, [toggle, main]));
  });
})();
