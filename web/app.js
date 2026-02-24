const api = {
  async create(parent_id, text) {
    return fetch("/comments", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ parent_id, text })
    });
  },

  async getTree(parent, page, limit, sort) {
    const u = new URL("/comments", location.origin);
    u.searchParams.set("parent", String(parent));
    u.searchParams.set("page", String(page));
    u.searchParams.set("limit", String(limit));
    u.searchParams.set("sort", sort);
    return fetch(u.toString());
  },

  async del(id) {
    return fetch(`/comments/${id}`, { method: "DELETE" });
  },

  async search(q, page, limit, sort) {
    const u = new URL("/comments/search", location.origin);
    u.searchParams.set("q", q);
    u.searchParams.set("page", String(page));
    u.searchParams.set("limit", String(limit));
    u.searchParams.set("sort", sort);
    return fetch(u.toString());
  },

  async path(id) {
    const u = new URL("/comments/path", location.origin);
    u.searchParams.set("id", String(id));
    return fetch(u.toString());
  },

  async subtree(id, sort) {
    const u = new URL("/comments/subtree", location.origin);
    u.searchParams.set("id", String(id));
    u.searchParams.set("sort", sort);
    return fetch(u.toString());
  }
};

const els = {
  status: document.getElementById("status"),
  tree: document.getElementById("tree"),
  results: document.getElementById("results"),

  searchInput: document.getElementById("searchInput"),
  searchBtn: document.getElementById("searchBtn"),
  resetBtn: document.getElementById("resetBtn"),

  newText: document.getElementById("newText"),
  createRootBtn: document.getElementById("createRootBtn"),

  sortSelect: document.getElementById("sortSelect"),
  reloadTreeBtn: document.getElementById("reloadTreeBtn")
};

let state = {
  sort: els.sortSelect.value,
  highlightId: null,
  currentRootId: null
};

function setStatus(msg, kind = "info") {
  if (!msg) { els.status.textContent = ""; return; }
  if (kind === "ok") els.status.style.color = "var(--ok)";
  else if (kind === "err") els.status.style.color = "var(--danger)";
  else els.status.style.color = "var(--muted)";
  els.status.textContent = msg;
}

function fmtDate(iso) {
  try { return new Date(iso).toLocaleString(); } catch { return iso; }
}

function escapeHtml(s) {
  return String(s)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#039;");
}

function renderTree(nodes) {
  els.tree.innerHTML = "";
  if (!nodes || nodes.length === 0) {
    els.tree.innerHTML = `<div class="results empty">Пока нет комментариев.</div>`;
    return;
  }
  nodes.forEach(n => els.tree.appendChild(renderNode(n)));
  if (state.highlightId) {
    scrollToHighlight();
  }
}

function renderNode(node) {
  const wrap = document.createElement("div");
  wrap.className = "node";
  wrap.dataset.id = node.id;

  const isHighlighted = state.highlightId === node.id;

  wrap.innerHTML = `
    <div class="${isHighlighted ? "highlight" : ""}">
      <div class="nodeHead">
        <div class="nodeText">
          <div>${escapeHtml(node.text)}</div>
          <div class="nodeMeta">id=${node.id} · parent=${node.parent_id} · ${fmtDate(node.created_at)}</div>
        </div>
        <div class="nodeActions">
          <button class="secondary replyToggle">Ответить</button>
          <button class="danger deleteBtn">Удалить</button>
        </div>
      </div>

      <div class="replyRow" style="display:none;">
        <input class="replyInput" type="text" placeholder="Ответ..." />
        <button class="replyBtn">Ок</button>
      </div>
    </div>
  `;

  wrap.querySelector(".replyToggle").addEventListener("click", () => {
    const row = wrap.querySelector(".replyRow");
    row.style.display = (row.style.display === "none") ? "flex" : "none";
  });

  wrap.querySelector(".replyBtn").addEventListener("click", async () => {
    const input = wrap.querySelector(".replyInput");
    const text = input.value.trim();
    if (!text) return setStatus("Текст ответа пустой", "err");

    const resp = await api.create(node.id, text);
    if (!resp.ok) {
      const e = await safeJson(resp);
      return setStatus(`Ошибка создания: ${e?.error || resp.status}`, "err");
    }
    input.value = "";
    setStatus("Ответ добавлен", "ok");
    await reloadCurrentTree();
  });

  wrap.querySelector(".deleteBtn").addEventListener("click", async () => {
    if (!confirm("Удалить комментарий и всё поддерево?")) return;

    const resp = await api.del(node.id);
    if (!resp.ok) {
      const e = await safeJson(resp);
      return setStatus(`Ошибка удаления: ${e?.error || resp.status}`, "err");
    }
    const data = await resp.json();
    setStatus(`Удалено: ${data.deleted}`, "ok");

    // если удалили корень текущей ветки — возвращаемся к корням
    if (state.currentRootId === node.id) {
      state.currentRootId = null;
      state.highlightId = null;
      await loadRootTree();
    } else {
      await reloadCurrentTree();
    }
  });

  // children
  if (node.children && node.children.length) {
    node.children.forEach(ch => wrap.appendChild(renderNode(ch)));
  }

  return wrap;
}

async function safeJson(resp) {
  try { return await resp.json(); } catch { return null; }
}

function scrollToHighlight() {
  const el = document.querySelector(`[data-id="${state.highlightId}"] .highlight`);
  if (el) el.scrollIntoView({ behavior: "smooth", block: "center" });
}

async function loadRootTree() {
  state.currentRootId = null;
  state.highlightId = null;
  setStatus("Загружаю корневые комментарии…");
  const resp = await api.getTree(0, 1, 50, state.sort);
  if (!resp.ok) {
    const e = await safeJson(resp);
    setStatus(`Ошибка загрузки: ${e?.error || resp.status}`, "err");
    return;
  }
  const data = await resp.json();
  renderTree(data.items);
  setStatus("");
}

async function loadSubtree(rootId, highlightId) {
  state.currentRootId = rootId;
  state.highlightId = highlightId ?? null;

  setStatus(`Открываю ветку root=${rootId}…`);
  const resp = await api.subtree(rootId, state.sort);
  if (!resp.ok) {
    const e = await safeJson(resp);
    setStatus(`Ошибка ветки: ${e?.error || resp.status}`, "err");
    return;
  }
  const node = await resp.json();
  renderTree([node]);
  setStatus("");
}

async function reloadCurrentTree() {
  if (state.currentRootId) {
    await loadSubtree(state.currentRootId, state.highlightId);
  } else {
    await loadRootTree();
  }
}

async function onSearch() {
  const q = els.searchInput.value.trim();
  if (!q) {
    els.results.className = "results empty";
    els.results.textContent = "Введите запрос.";
    return;
  }

  setStatus("Ищу…");
  const resp = await api.search(q, 1, 20, "rank_desc");
  if (!resp.ok) {
    const e = await safeJson(resp);
    setStatus(`Ошибка поиска: ${e?.error || resp.status}`, "err");
    return;
  }
  const data = await resp.json();
  renderResults(data.items);
  setStatus(`Найдено: ${data.total}`, "ok");
}

function renderResults(items) {
  if (!items || items.length === 0) {
    els.results.className = "results empty";
    els.results.textContent = "Ничего не найдено.";
    return;
  }

  els.results.className = "results";
  els.results.innerHTML = "";

  items.forEach(it => {
    const div = document.createElement("div");
    div.className = "resultItem";
    div.innerHTML = `
      <div class="snippet">${it.snippet}</div>
      <div class="meta">id=${it.id} · parent=${it.parent_id} · rank=${Number(it.rank).toFixed(3)} · ${fmtDate(it.created_at)}</div>
      <button class="openBtn secondary">Открыть в дереве</button>
    `;
    div.querySelector(".openBtn").addEventListener("click", () => openInTree(it.id));
    els.results.appendChild(div);
  });
}

async function openInTree(id) {
  setStatus("Строю путь…");
  const resp = await api.path(id);
  if (!resp.ok) {
    const e = await safeJson(resp);
    setStatus(`Ошибка path: ${e?.error || resp.status}`, "err");
    return;
  }
  const data = await resp.json();
  const path = data.items;
  if (!path || path.length === 0) {
    setStatus("Путь не найден", "err");
    return;
  }
  const rootId = path[0].id;
  await loadSubtree(rootId, id);
}

// handlers
els.searchBtn.addEventListener("click", onSearch);
els.searchInput.addEventListener("keydown", (e) => { if (e.key === "Enter") onSearch(); });

els.resetBtn.addEventListener("click", async () => {
  els.searchInput.value = "";
  els.results.className = "results empty";
  els.results.textContent = "Введите запрос и нажмите “Найти”.";
  await loadRootTree();
});

els.sortSelect.addEventListener("change", async () => {
  state.sort = els.sortSelect.value;
  await reloadCurrentTree();
});

els.reloadTreeBtn.addEventListener("click", reloadCurrentTree);

els.createRootBtn.addEventListener("click", async () => {
  const text = els.newText.value.trim();
  if (!text) return setStatus("Текст пустой", "err");

  const resp = await api.create(0, text);
  if (!resp.ok) {
    const e = await safeJson(resp);
    return setStatus(`Ошибка создания: ${e?.error || resp.status}`, "err");
  }
  els.newText.value = "";
  setStatus("Комментарий добавлен", "ok");
  await loadRootTree();
});

// init
loadRootTree();