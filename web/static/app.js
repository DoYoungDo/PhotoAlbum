/* ===== PhotoAlbum 前端 ===== */
'use strict';

// ── 工具函数 ──────────────────────────────────────────
const $ = (sel, ctx = document) => ctx.querySelector(sel);
const $$ = (sel, ctx = document) => [...ctx.querySelectorAll(sel)];
const el = (tag, cls, html = '') => {
  const e = document.createElement(tag);
  if (cls) e.className = cls;
  if (html) e.innerHTML = html;
  return e;
};
const api = {
  async get(url) {
    const r = await fetch(url);
    if (!r.ok) throw await r.json();
    return r.json();
  },
  async post(url, data) {
    const r = await fetch(url, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(data) });
    if (!r.ok) throw await r.json();
    return r.json();
  },
  async put(url, data) {
    const r = await fetch(url, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(data) });
    if (!r.ok) throw await r.json();
    return r.json();
  },
  async del(url) {
    const r = await fetch(url, { method: 'DELETE' });
    if (!r.ok) throw await r.json();
    return r.json();
  },
};
function formatDate(iso) {
  const d = new Date(iso);
  return d.toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' });
}
function formatSize(bytes) {
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1048576) return (bytes / 1024).toFixed(1) + ' KB';
  return (bytes / 1048576).toFixed(1) + ' MB';
}
function groupByDate(photos) {
  const groups = {};
  for (const p of photos) {
    const key = formatDate(p.taken_at);
    if (!groups[key]) groups[key] = [];
    groups[key].push(p);
  }
  return groups;
}
let _uid = 0;
function uid() { return 'u' + (++_uid); }

// c-3: 长按检测（移动端操作菜单）
function addLongPress(el, callback, delay = 500) {
  let timer = null;
  let moved = false;
  el.addEventListener('touchstart', e => {
    moved = false;
    timer = setTimeout(() => {
      if (!moved) { e.preventDefault(); callback(e); }
    }, delay);
  }, { passive: false });
  el.addEventListener('touchmove',  () => { moved = true; clearTimeout(timer); });
  el.addEventListener('touchend',   () => clearTimeout(timer));
  el.addEventListener('touchcancel',() => clearTimeout(timer));
}

// ── SVG 图标 ──────────────────────────────────────────
const icons = {
  timeline: `<svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/></svg>`,
  album:    `<svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M3 7h18M3 12h18M3 17h18"/></svg>`,
  trash:    `<svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><polyline points="3 6 5 6 21 6"/><path d="M19 6l-1 14H6L5 6"/><path d="M10 11v6M14 11v6"/><path d="M9 6V4h6v2"/></svg>`,
  upload:   `<svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><polyline points="16 16 12 12 8 16"/><line x1="12" y1="12" x2="12" y2="21"/><path d="M20.39 18.39A5 5 0 0018 9h-1.26A8 8 0 103 16.3"/></svg>`,
  sun:      `<svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>`,
  moon:     `<svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg>`,
  close:    `<svg width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>`,
  prev:     `<svg width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><polyline points="15 18 9 12 15 6"/></svg>`,
  next:     `<svg width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><polyline points="9 18 15 12 9 6"/></svg>`,
  share:    `<svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><line x1="8.59" y1="13.51" x2="15.42" y2="17.49"/><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"/></svg>`,
  check:    `<svg width="12" height="12" fill="none" stroke="#fff" stroke-width="2.5" viewBox="0 0 24 24"><polyline points="20 6 9 17 4 12"/></svg>`,
  photo:    `<svg width="48" height="48" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24"><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><polyline points="21 15 16 10 5 21"/></svg>`,
  logout:   `<svg width="18" height="18" fill="none" stroke="currentColor" stroke-width="1.8" viewBox="0 0 24 24"><path d="M9 21H5a2 2 0 01-2-2V5a2 2 0 012-2h4"/><polyline points="16 17 21 12 16 7"/><line x1="21" y1="12" x2="9" y2="12"/></svg>`,
  plus:     `<svg width="16" height="16" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>`,
  shareSmall: `<svg width="11" height="11" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><line x1="8.59" y1="13.51" x2="15.42" y2="17.49"/><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"/></svg>`,
};

// ── 主题 ──────────────────────────────────────────────
function initTheme() {
  const saved = localStorage.getItem('theme') || (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
  document.documentElement.dataset.theme = saved;
}
function toggleTheme() {
  const t = document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark';
  document.documentElement.dataset.theme = t;
  localStorage.setItem('theme', t);
  updateThemeBtn();
}
function updateThemeBtn() {
  const btn = $('#theme-btn');
  if (!btn) return;
  const iconEl = btn.querySelector('.theme-icon');
  if (iconEl) iconEl.innerHTML = document.documentElement.dataset.theme === 'dark' ? icons.sun : icons.moon;
}

// ── 状态 ──────────────────────────────────────────────
const state = {
  view: 'timeline',
  photos: [],
  timelineCursor: '',
  timelineHasMore: true,
  timelineLoading: false,
  trashPhotos: [],
  trashCursor: '',
  trashHasMore: true,
  trashLoading: false,
  albums: [],
  currentAlbum: null,
  albumPhotos: [],
  albumCursor: '',
  albumHasMore: true,
  albumLoading: false,
  selected: new Set(),
  lightboxPhotos: [],
  lightboxIndex: 0,
  // b-2: 当前用户的分享链接，key=`${type}:${targetId}`
  shareMap: {},
  // d-2: 上传队列状态
  uploadJobs: [],
  uploadRunning: false,
  // h-1: 用于刷新后恢复相册详情页
  currentAlbumID: null,
};

// ── 分享状态加载 ─────────────────────────────────────
async function loadShareMap() {
  try {
    const links = await api.get('/api/shares');
    state.shareMap = {};
    (links || []).forEach(l => {
      const key = `${l.type}:${l.target_id}`;
      if (!state.shareMap[key]) state.shareMap[key] = [];
      state.shareMap[key].push(l);
    });
  } catch(e) { /* 非关键，忽略 */ }
}

// ── 右键菜单 ──────────────────────────────────────────
let _ctxMenu = null;
function showContextMenu(x, y, items) {
  closeContextMenu();
  const menu = el('div');
  menu.style.cssText = `position:fixed;left:${x}px;top:${y}px;z-index:2000;
    background:var(--card);border:1px solid var(--border);border-radius:8px;
    box-shadow:0 4px 16px rgba(0,0,0,.15);padding:4px 0;min-width:160px;`;
  items.forEach(item => {
    if (item === '-') {
      const sep = el('div');
      sep.style.cssText = 'height:1px;background:var(--border);margin:4px 0;';
      menu.appendChild(sep); return;
    }
    const btn = el('button');
    btn.textContent = item.label;
    if (item.danger) btn.style.color = 'var(--danger)';
    btn.style.cssText += `display:block;width:100%;padding:8px 14px;background:none;border:none;
      text-align:left;font-size:.88rem;cursor:pointer;`;
    btn.addEventListener('mouseenter', () => btn.style.background = 'var(--bg2)');
    btn.addEventListener('mouseleave', () => btn.style.background = 'none');
    btn.addEventListener('click', () => { closeContextMenu(); item.action(); });
    menu.appendChild(btn);
  });
  document.body.appendChild(menu);
  _ctxMenu = menu;
  const rect = menu.getBoundingClientRect();
  if (rect.right > window.innerWidth)   menu.style.left = (x - rect.width) + 'px';
  if (rect.bottom > window.innerHeight) menu.style.top  = (y - rect.height) + 'px';
}
function closeContextMenu() {
  if (_ctxMenu) { _ctxMenu.remove(); _ctxMenu = null; }
}

// ── 渲染框架 ──────────────────────────────────────────
function renderApp() {
  const isDark = document.documentElement.dataset.theme === 'dark';
  document.body.innerHTML = `
<div class="drawer-overlay" id="drawer-overlay"></div>
<div id="app">
  <nav class="nav" id="main-nav">
    <div class="nav-logo">${icons.photo} PhotoAlbum</div>
    <a class="nav-item${state.view === 'timeline' ? ' active' : ''}" href="#" data-view="timeline">${icons.timeline} 时间线</a>
    <a class="nav-item${state.view === 'albums' || state.view === 'album-detail' ? ' active' : ''}" href="#" data-view="albums">${icons.album} 相册</a>
    <a class="nav-item${state.view === 'trash' ? ' active' : ''}" href="#" data-view="trash">${icons.trash} 回收站</a>
    <div class="nav-spacer"></div>
    <div class="nav-bottom">
      <a class="nav-item" href="#" id="theme-btn"><span class="theme-icon">${isDark ? icons.sun : icons.moon}</span> 切换主题</a>
      <a class="nav-item" href="#" id="logout-btn">${icons.logout} 退出登录</a>
    </div>
  </nav>
  <div class="main">
    <div class="topbar">
      <button class="hamburger" id="hamburger-btn" aria-label="菜单">
        <span></span><span></span><span></span>
      </button>
      <span class="topbar-title" id="topbar-title"></span>
      <div id="topbar-actions"></div>
    </div>
    <div class="content" id="content"></div>
  </div>
</div>
${renderLightbox()}
${renderUploadModal()}
${renderCreateAlbumModal()}
${renderShareModal()}
${renderAlbumPickerModal()}
${renderShareListModal()}`;

  bindNav();
  bindGlobal();
  renderView();
}

function bindNav() {
  $$('.nav-item[data-view]').forEach(a => {
    a.addEventListener('click', e => { e.preventDefault(); closeDrawer(); switchView(a.dataset.view); });
  });
  $('#theme-btn').addEventListener('click', e => { e.preventDefault(); toggleTheme(); });
  $('#logout-btn').addEventListener('click', e => { e.preventDefault(); logout(); });

  // 汉堡按钮 / 抽屉 (b-6)
  const hamburger = $('#hamburger-btn');
  const overlay   = $('#drawer-overlay');
  if (hamburger) hamburger.addEventListener('click', toggleDrawer);
  if (overlay)   overlay.addEventListener('click', closeDrawer);
}

function toggleDrawer() {
  const nav     = $('#main-nav');
  const overlay = $('#drawer-overlay');
  const open    = nav && nav.classList.toggle('open');
  if (overlay) overlay.classList.toggle('open', open);
}
function closeDrawer() {
  const nav     = $('#main-nav');
  const overlay = $('#drawer-overlay');
  if (nav)     nav.classList.remove('open');
  if (overlay) overlay.classList.remove('open');
}

function switchView(view) {
  state.view = view;
  state.selected.clear();
  if (view !== 'album-detail') {
    state.currentAlbumID = null;
    setHashView(view); // c-2: 同步到 hash
  }
  $$('.nav-item[data-view]').forEach(a => a.classList.toggle('active', a.dataset.view === view));
  renderView();
}
function renderView() {
  switch (state.view) {
    case 'timeline':     renderTimeline();    break;
    case 'albums':       renderAlbums();      break;
    case 'album-detail': renderAlbumDetail(); break;
    case 'trash':        renderTrash();       break;
  }
}

// ── 时间线视图 ─────────────────────────────────────────
async function renderTimeline() {
  $('#topbar-title').textContent = '时间线';
  $('#topbar-actions').innerHTML = `<button class="btn btn-primary btn-sm" id="upload-btn">${icons.upload} 上传</button>`;
  $('#upload-btn').addEventListener('click', openUploadModal);

  $('#content').innerHTML = `
<div class="toolbar">
  <span id="sel-bar" class="selected-bar">
    <span class="selected-count" id="sel-count">0</span> 张已选
    <button class="btn btn-sm" style="background:rgba(255,255,255,.2);border-color:transparent;color:#fff" id="add-to-album-btn">${icons.album} 添加到相册</button>
    <button class="btn btn-sm" style="background:rgba(255,255,255,.2);border-color:transparent;color:#fff" id="delete-sel-btn">${icons.trash} 删除</button>
    <button class="btn-icon" style="color:#fff" id="clear-sel-btn">${icons.close}</button>
  </span>
</div>
<div id="timeline-groups"></div>
<div class="load-more" id="load-more"><div class="spinner"></div>加载中…</div>`;

  $('#clear-sel-btn').addEventListener('click', clearSelection);
  $('#delete-sel-btn').addEventListener('click', deleteSelected);
  $('#add-to-album-btn').addEventListener('click', () => openAlbumPickerModal(null));

  state.photos = [];
  state.timelineCursor = '';
  state.timelineHasMore = true;
  await loadShareMap();   // b-2: 加载分享状态
  await loadMoreTimeline();
  observeLoadMore('load-more', loadMoreTimeline, () => state.timelineHasMore && !state.timelineLoading);
}

async function loadMoreTimeline() {
  if (state.timelineLoading || !state.timelineHasMore) return;
  state.timelineLoading = true;
  try {
    const url = '/api/photos' + (state.timelineCursor ? `?cursor=${encodeURIComponent(state.timelineCursor)}` : '');
    const page = await api.get(url);
    state.photos.push(...(page.photos || []));
    state.timelineCursor = page.next_cursor || '';
    state.timelineHasMore = page.has_more || false;
    renderTimelineGroups(page.photos || [], state.photos.length - (page.photos || []).length);
  } catch (e) { console.error(e); }
  finally { state.timelineLoading = false; updateLoadMoreUI('load-more', state.timelineHasMore); }
}

function renderTimelineGroups(newPhotos, offset) {
  const container = $('#timeline-groups');
  if (!container) return;
  if (offset === 0 && newPhotos.length === 0) {
    container.innerHTML = `<div class="empty">${icons.photo}<p>还没有照片，点击右上角上传吧</p></div>`;
    return;
  }
  const groups = groupByDate(newPhotos);
  for (const [date, photos] of Object.entries(groups)) {
    let group = container.querySelector(`[data-date="${CSS.escape(date)}"]`);
    if (!group) {
      group = el('div', 'date-group');
      group.dataset.date = date;
      group.innerHTML = `<div class="date-label"><span class="date-text">${date}</span><button class="date-select-all" type="button">全选本日</button></div><div class="photo-grid"></div>`;
      group.querySelector('.date-select-all').addEventListener('click', () => selectAllInGroup(group));
      container.appendChild(group);
    }
    const grid = group.querySelector('.photo-grid');
    photos.forEach(p => grid.appendChild(makePhotoThumb(p, state.photos)));
  }
}

// ── 缩略图 ────────────────────────────────────────────
function makePhotoThumb(photo, listRef, opts = {}) {
  const div = el('div', 'photo-thumb');
  div.dataset.id = photo.id;

  const isShared = !!state.shareMap[`photo:${photo.id}`];
  const shareBadge = isShared
    ? `<span class="share-badge">${icons.shareSmall}</span>` : '';

  div.innerHTML = `<span class="check">${icons.check}</span><img loading="lazy" src="/media/thumbnails/${photo.uuid}" alt="${photo.original_name}">${shareBadge}`;

  // b-1: 点击 .check 区域直接进入/切换选择模式
  const checkEl = div.querySelector('.check');
  checkEl.addEventListener('click', e => {
    e.stopPropagation();
    toggleSelect(photo.id, div);
  });

  // 图片主体点击
  div.addEventListener('click', () => {
    if (opts.trashMode) {
      openLightbox(listRef, listRef.indexOf(photo));
      return;
    }
    if (state.selected.size > 0) {
      toggleSelect(photo.id, div);
    } else {
      openLightbox(listRef, listRef.indexOf(photo));
    }
  });

  // PC 右键菜单
  div.addEventListener('contextmenu', e => {
    e.preventDefault();
    if (opts.trashMode) showTrashContextMenu(e.clientX, e.clientY, photo);
    else showPhotoContextMenu(e.clientX, e.clientY, photo, div, listRef);
  });

  // c-3: 长按触发操作菜单（移动端）
  addLongPress(div, e => {
    const touch = e.changedTouches[0];
    if (opts.trashMode) showTrashContextMenu(touch.clientX, touch.clientY, photo);
    else showPhotoContextMenu(touch.clientX, touch.clientY, photo, div, listRef);
  });

  if (state.selected.has(photo.id)) div.classList.add('selected');
  return div;
}

// 时间线图片右键菜单
function showPhotoContextMenu(x, y, photo, thumbEl, listRef) {
  const isSelected = state.selected.has(photo.id);
  const isShared   = !!state.shareMap[`photo:${photo.id}`];
  showContextMenu(x, y, [
    { label: isSelected ? '取消选择' : '选择（点击勾选图标可快速选择）', action: () => toggleSelect(photo.id, thumbEl) },
    { label: '查看', action: () => openLightbox(listRef, listRef.indexOf(photo)) },
    { label: '下载', action: () => triggerDownload(`/api/photos/${photo.id}/download`) },
    '-',
    { label: '添加到相册…', action: () => openAlbumPickerModal([photo.id]) },
    { label: isShared ? '管理分享…' : '分享…', action: () => isShared ? openShareListModal('photo', photo.id) : openShareModal('photo', photo.id) },
    '-',
    { label: '删除', danger: true, action: () => deleteSinglePhoto(photo.id) },
  ]);
}

// 回收站图片右键菜单 (b-4)
function showTrashContextMenu(x, y, photo) {
  showContextMenu(x, y, [
    { label: '恢复到时间线', action: () => restorePhoto(photo.id) },
    { label: '永久删除', danger: true, action: () => hardDeleteSinglePhoto(photo.id) },
  ]);
}

async function addSinglePhotoToAlbum(photoId) {
  openAlbumPickerModal([photoId]);
}

async function deleteSinglePhoto(photoId) {
  if (!confirm('确定要将这张照片移入回收站吗？')) return;
  try { await api.del(`/api/photos/${photoId}`); switchView('timeline'); }
  catch(e) { alert('删除失败: ' + (e.error || e)); }
}

async function hardDeleteSinglePhoto(photoId) {
  if (!confirm('确定要永久删除这张照片吗？此操作不可恢复。')) return;
  try { await api.del(`/api/photos/${photoId}`); switchView('trash'); }
  catch(e) { alert('删除失败: ' + (e.error || e)); }
}

// ── 选择 ─────────────────────────────────────────────
function toggleSelect(id, thumbEl) {
  if (state.selected.has(id)) { state.selected.delete(id); thumbEl.classList.remove('selected'); }
  else { state.selected.add(id); thumbEl.classList.add('selected'); }
  updateSelectionModeUI();
  updateSelectionBar();
}
function clearSelection() {
  state.selected.clear();
  $$('.photo-thumb.selected').forEach(t => t.classList.remove('selected'));
  updateSelectionModeUI();
  updateSelectionBar();
  updateTrashSelBar();
}
function updateSelectionModeUI() {
  document.body.classList.toggle('selection-mode', state.selected.size > 0);
}
function updateSelectionBar() {
  const bar = $('#sel-bar');
  if (!bar) return;
  bar.classList.toggle('visible', state.selected.size > 0);
  const cnt = $('#sel-count');
  if (cnt) cnt.textContent = state.selected.size;
}
function selectAllInGroup(groupEl) {
  const thumbs = $$('.photo-thumb', groupEl);
  thumbs.forEach(thumb => {
    const id = Number(thumb.dataset.id);
    if (!state.selected.has(id)) {
      state.selected.add(id);
      thumb.classList.add('selected');
    }
  });
  updateSelectionModeUI();
  updateSelectionBar();
  updateTrashSelBar();
}
async function deleteSelected() {
  if (!state.selected.size) return;
  if (!confirm(`确定要删除选中的 ${state.selected.size} 张照片吗？`)) return;
  for (const id of state.selected) {
    try { await api.del(`/api/photos/${id}`); } catch (e) { console.error(e); }
  }
  clearSelection();
  switchView('timeline');
}

// ── 相册列表 ──────────────────────────────────────────
async function renderAlbums() {
  $('#topbar-title').textContent = '相册';
  $('#topbar-actions').innerHTML = `<button class="btn btn-primary btn-sm" id="new-album-btn">${icons.plus} 新建相册</button>`;
  $('#new-album-btn').addEventListener('click', openCreateAlbumModal);

  $('#content').innerHTML = `<div id="album-grid-wrap"></div>`;
  try {
    state.albums = await api.get('/api/albums');
    renderAlbumGrid();
  } catch(e) { $('#content').innerHTML = `<p style="color:var(--danger)">加载失败</p>`; }
}

function renderAlbumGrid() {
  const wrap = $('#album-grid-wrap');
  if (!wrap) return;
  if (!state.albums || !state.albums.length) {
    wrap.innerHTML = `<div class="empty">${icons.album}<p>还没有相册，点击右上角新建</p></div>`;
    return;
  }
  const grid = el('div', 'album-grid');
  state.albums.forEach(a => grid.appendChild(makeAlbumCard(a)));
  wrap.innerHTML = '';
  wrap.appendChild(grid);
}

function makeAlbumCard(album) {
  const card = el('div', 'album-card');
  // c-1: 用 cover_uuid 显示封面缩略图
  const coverHtml = album.cover_uuid
    ? `<img loading="lazy" src="/media/thumbnails/${album.cover_uuid}" alt="${album.name}">`
    : `<div class="album-cover-empty">${icons.photo}</div>`;
  card.innerHTML = `
<div class="album-cover">${coverHtml}</div>
<div class="album-info">
  <div class="album-name">${album.name}</div>
  <div class="album-count">${album.photo_count || 0} 张</div>
</div>`;
  card.addEventListener('click', () => openAlbumDetail(album));
  return card;
}

// ── 相册详情 ──────────────────────────────────────────
async function openAlbumDetail(album) {
  state.currentAlbum = album;
  state.currentAlbumID = album.id;
  state.albumPhotos = [];
  state.albumCursor = '';
  state.albumHasMore = true;
  state.view = 'album-detail';
  setHashView('album-detail', album.id);
  $$('.nav-item[data-view]').forEach(a => a.classList.toggle('active', a.dataset.view === 'albums'));
  renderAlbumDetail();
}
async function renderAlbumDetail() {
  let album = state.currentAlbum;
  if (!album && state.currentAlbumID) {
    try {
      album = await api.get(`/api/albums/${state.currentAlbumID}`);
      state.currentAlbum = album;
    } catch (e) {
      // 相册不存在或加载失败时回退到相册列表
      state.currentAlbum = null;
      state.currentAlbumID = null;
      switchView('albums');
      return;
    }
  }
  if (!album) {
    switchView('albums');
    return;
  }
  $('#topbar-title').textContent = album.name;
  $('#topbar-actions').innerHTML = `<button class="btn btn-danger btn-sm" id="delete-album-btn">删除相册</button><button class="btn btn-sm" id="back-albums-btn">← 返回相册</button>`;
  $('#delete-album-btn').addEventListener('click', async () => {
    if (!confirm(`确定要删除相册「${album.name}」吗？图片本身不会被删除。`)) return;
    try {
      await api.del(`/api/albums/${album.id}`);
      state.currentAlbum = null;
      switchView('albums');
    } catch (e) {
      alert('删除相册失败: ' + (e.error || e));
    }
  });
  $('#back-albums-btn').addEventListener('click', () => switchView('albums'));

  $('#content').innerHTML = `<div id="album-groups"></div><div class="load-more" id="load-more"><div class="spinner"></div>加载中…</div>`;
  state.albumPhotos = []; state.albumCursor = ''; state.albumHasMore = true;
  await loadMoreAlbumPhotos();
  observeLoadMore('load-more', loadMoreAlbumPhotos, () => state.albumHasMore && !state.albumLoading);
}
async function loadMoreAlbumPhotos() {
  if (state.albumLoading || !state.albumHasMore || !state.currentAlbum) return;
  state.albumLoading = true;
  try {
    const id = state.currentAlbum.id;
    const url = `/api/albums/${id}/photos` + (state.albumCursor ? `?cursor=${encodeURIComponent(state.albumCursor)}` : '');
    const page = await api.get(url);
    state.albumPhotos.push(...(page.photos || []));
    state.albumCursor = page.next_cursor || '';
    state.albumHasMore = page.has_more || false;
    renderAlbumGroups(page.photos || []);
  } catch(e) { console.error(e); }
  finally { state.albumLoading = false; updateLoadMoreUI('load-more', state.albumHasMore); }
}
function renderAlbumGroups(newPhotos) {
  const container = $('#album-groups');
  if (!container) return;
  if (state.albumPhotos.length === 0 && newPhotos.length === 0) {
    container.innerHTML = `<div class="empty">${icons.photo}<p>相册还没有照片</p></div>`;
    return;
  }
  const groups = groupByDate(newPhotos);
  for (const [date, photos] of Object.entries(groups)) {
    let group = container.querySelector(`[data-date="${CSS.escape(date)}"]`);
    if (!group) {
      group = el('div', 'date-group');
      group.dataset.date = date;
      group.innerHTML = `<div class="date-label"><span class="date-text">${date}</span><button class="date-select-all" type="button">全选本日</button></div><div class="photo-grid"></div>`;
      group.querySelector('.date-select-all').addEventListener('click', () => selectAllInGroup(group));
      container.appendChild(group);
    }
    const grid = group.querySelector('.photo-grid');
    photos.forEach(p => grid.appendChild(makePhotoThumb(p, state.albumPhotos)));
  }
}

// ── 回收站 (b-4 修复) ─────────────────────────────────
async function renderTrash() {
  $('#topbar-title').textContent = '回收站';
  $('#topbar-actions').innerHTML = `<button class="btn btn-danger btn-sm" id="empty-trash-btn">${icons.trash} 清空回收站</button>`;
  $('#empty-trash-btn').addEventListener('click', emptyTrash);

  // c-5: 加入批量恢复工具栏
  $('#content').innerHTML = `
<div class="toolbar">
  <span id="trash-sel-bar" class="selected-bar">
    <span class="selected-count" id="trash-sel-count">0</span> 张已选
    <button class="btn btn-sm" style="background:rgba(255,255,255,.2);border-color:transparent;color:#fff" id="restore-sel-btn">${icons.prev} 批量恢复</button>
    <button class="btn-icon" style="color:#fff" id="trash-clear-sel-btn">${icons.close}</button>
  </span>
</div>
<p style="font-size:.82rem;color:var(--text2);margin-bottom:8px">左键预览，右键或长按恢复/删除，点击勾选图标批量操作</p>
<div id="trash-groups"></div>
<div class="load-more" id="load-more"><div class="spinner"></div>加载中…</div>`;

  $('#trash-clear-sel-btn').addEventListener('click', () => { clearSelection(); updateTrashSelBar(); });
  $('#restore-sel-btn').addEventListener('click', restoreSelected);

  state.trashPhotos = []; state.trashCursor = ''; state.trashHasMore = true;
  await loadMoreTrash();
  observeLoadMore('load-more', loadMoreTrash, () => state.trashHasMore && !state.trashLoading);
}
async function loadMoreTrash() {
  if (state.trashLoading || !state.trashHasMore) return;
  state.trashLoading = true;
  try {
    const url = '/api/trash' + (state.trashCursor ? `?cursor=${encodeURIComponent(state.trashCursor)}` : '');
    const page = await api.get(url);
    state.trashPhotos.push(...(page.photos || []));
    state.trashCursor = page.next_cursor || '';
    state.trashHasMore = page.has_more || false;
    renderTrashGroups(page.photos || []);
  } catch(e) { console.error(e); }
  finally { state.trashLoading = false; updateLoadMoreUI('load-more', state.trashHasMore); }
}
function renderTrashGroups(newPhotos) {
  const container = $('#trash-groups');
  if (!container) return;
  if (state.trashPhotos.length === 0 && newPhotos.length === 0) {
    container.innerHTML = `<div class="empty">${icons.trash}<p>回收站是空的</p></div>`;
    return;
  }
  const groups = groupByDate(newPhotos);
  for (const [date, photos] of Object.entries(groups)) {
    let group = container.querySelector(`[data-date="${CSS.escape(date)}"]`);
    if (!group) {
      group = el('div', 'date-group');
      group.dataset.date = date;
      group.innerHTML = `<div class="date-label"><span class="date-text">${date}</span><button class="date-select-all" type="button">全选本日</button></div><div class="photo-grid"></div>`;
      group.querySelector('.date-select-all').addEventListener('click', () => selectAllInGroup(group));
      container.appendChild(group);
    }
    const grid = group.querySelector('.photo-grid');
    // b-4 + c-5: trashMode 支持勾选批量恢复
    photos.forEach(p => {
      const thumb = makePhotoThumb(p, state.trashPhotos, { trashMode: true });
      // 覆盖 check 的点击，同时更新回收站选择栏
      const ck = thumb.querySelector('.check');
      ck.addEventListener('click', e => {
        e.stopPropagation();
        toggleSelect(p.id, thumb);
        updateTrashSelBar();
      }, { capture: true });
      grid.appendChild(thumb);
    });
  }
}
async function emptyTrash() {
  if (!confirm('确定要永久删除回收站中所有照片吗？此操作不可恢复。')) return;
  try { await api.del('/api/trash'); switchView('trash'); }
  catch(e) { alert('操作失败: ' + (e.error || e)); }
}
async function restorePhoto(id) {
  try { await api.post(`/api/photos/${id}/restore`, {}); switchView('trash'); }
  catch(e) { alert('恢复失败: ' + (e.error || e)); }
}

// c-5: 更新回收站批量操作栏
function updateTrashSelBar() {
  const bar = $('#trash-sel-bar');
  if (!bar) return;
  bar.classList.toggle('visible', state.selected.size > 0);
  const cnt = $('#trash-sel-count');
  if (cnt) cnt.textContent = state.selected.size;
  updateSelectionModeUI();
}

// c-5: 批量恢复选中图片
async function restoreSelected() {
  if (!state.selected.size) return;
  if (!confirm(`确定要恢复选中的 ${state.selected.size} 张照片吗？`)) return;
  const ids = [...state.selected];
  clearSelection();
  for (const id of ids) {
    try { await api.post(`/api/photos/${id}/restore`, {}); }
    catch(e) { console.error('恢复失败:', id, e); }
  }
  switchView('trash');
}

// ── 无限滚动 ──────────────────────────────────────────
function observeLoadMore(id, loadFn, canLoad) {
  const sentinel = document.getElementById(id);
  if (!sentinel) return;
  const obs = new IntersectionObserver(entries => {
    if (entries[0].isIntersecting && canLoad()) loadFn();
  }, { rootMargin: '200px' });
  obs.observe(sentinel);
}
function updateLoadMoreUI(id, hasMore) {
  const e = document.getElementById(id);
  if (!e) return;
  e.style.display = hasMore ? '' : 'none';
}

// ── 灯箱 ──────────────────────────────────────────────
function renderLightbox() {
  return `<div class="lightbox" id="lightbox">
  <div class="lightbox-header">
    <span class="lb-title" id="lb-title"></span>
    <button class="btn-icon" style="color:#ccc" id="lb-download">下载</button>
    <button class="btn-icon" style="color:#ccc" id="lb-share">${icons.share}</button>
    <button class="btn-icon" style="color:#ccc" id="lb-close">${icons.close}</button>
  </div>
  <div class="lightbox-body">
    <img class="lightbox-img" id="lb-img" src="" alt="">
    <button class="lb-nav lb-prev" id="lb-prev">${icons.prev}</button>
    <button class="lb-nav lb-next" id="lb-next">${icons.next}</button>
  </div>
  <div class="lightbox-info" id="lb-info"></div>
</div>`;
}
function bindGlobal() {
  document.addEventListener('click', () => closeContextMenu());
  document.addEventListener('keydown', e => {
    if (e.key === 'Escape') closeContextMenu();
    if (!$('#lightbox').classList.contains('open')) return;
    if (e.key === 'Escape')     closeLightbox();
    if (e.key === 'ArrowLeft')  lbNav(-1);
    if (e.key === 'ArrowRight') lbNav(1);
  });
  document.addEventListener('click', e => {
    if (e.target.closest('#lb-close')) closeLightbox();
    if (e.target.closest('#lb-prev'))  lbNav(-1);
    if (e.target.closest('#lb-next'))  lbNav(1);
    if (e.target.closest('#lb-download')) downloadCurrentPhoto();
    if (e.target.closest('#lb-share')) lbShare();
  });
}
function openLightbox(photos, index) {
  state.lightboxPhotos = photos;
  state.lightboxIndex  = Math.max(0, index);
  $('#lightbox').classList.add('open');
  lbRender();
}
function closeLightbox() { $('#lightbox').classList.remove('open'); }
function lbNav(dir) {
  const n = state.lightboxIndex + dir;
  if (n < 0 || n >= state.lightboxPhotos.length) return;
  state.lightboxIndex = n;
  lbRender();
}
function lbRender() {
  const p = state.lightboxPhotos[state.lightboxIndex];
  if (!p) return;
  $('#lb-img').src = `/media/photos/${p.uuid}`;
  $('#lb-title').textContent = p.original_name;
  $('#lb-prev').classList.toggle('hidden', state.lightboxIndex === 0);
  $('#lb-next').classList.toggle('hidden', state.lightboxIndex === state.lightboxPhotos.length - 1);
  const items = [
    ['拍摄时间', formatDate(p.taken_at)],
    ['尺寸', p.width && p.height ? `${p.width} × ${p.height}` : '—'],
    ['大小', formatSize(p.size)],
    ['文件名', p.original_name],
  ];
  $('#lb-info').innerHTML = items.map(([k, v]) => `<div class="lb-info-item"><span>${k}</span><span>${v}</span></div>`).join('');
}
async function lbShare() {
  const p = state.lightboxPhotos[state.lightboxIndex];
  if (!p) return;
  closeLightbox();
  const isShared = !!state.shareMap[`photo:${p.id}`];
  if (isShared) openShareListModal('photo', p.id);
  else openShareModal('photo', p.id);
}

function triggerDownload(url) {
	const a = document.createElement('a');
	a.href = url;
	a.style.display = 'none';
	document.body.appendChild(a);
	a.click();
	a.remove();
}

function downloadCurrentPhoto() {
	const p = state.lightboxPhotos[state.lightboxIndex];
	if (!p) return;
	triggerDownload(`/api/photos/${p.id}/download`);
}

// ── 上传模态框 ────────────────────────────────────────
function renderUploadModal() {
  return `<div class="modal-overlay" id="upload-modal">
  <div class="modal" style="width:520px">
    <div class="modal-title">${icons.upload} 上传照片</div>
    <div class="upload-zone" id="drop-zone">
      ${icons.upload}
      <div style="margin-top:8px">拖拽照片到这里，或点击选择文件</div>
      <div style="font-size:.8rem;margin-top:4px">支持 JPG、PNG、GIF、WebP</div>
      <input type="file" id="file-input" accept="image/*" multiple aria-hidden="true">
    </div>
    <div class="upload-queue" id="upload-queue"></div>
    <div class="modal-footer">
      <button class="btn" id="retry-failed-btn" style="display:none">重传失败项</button>
      <button class="btn" id="upload-close-btn">关闭</button>
    </div>
  </div>
</div>`;
}
let _uploadZoneBound = false;
function openUploadModal() {
  $('#upload-modal').classList.add('open');
  $('#upload-queue').innerHTML = '';
  $('#retry-failed-btn').style.display = 'none';
  state.uploadJobs = [];
  state.uploadRunning = false;
  const input = $('#file-input');
  if (input) input.value = '';
  if (!_uploadZoneBound) { _uploadZoneBound = true; bindUploadZone(); }
}
function closeUploadModal() { $('#upload-modal').classList.remove('open'); }
function bindUploadZone() {
  const zone = $('#drop-zone');
  const input = $('#file-input');
  $('#upload-close-btn').addEventListener('click', () => { closeUploadModal(); switchView('timeline'); });
  $('#retry-failed-btn').addEventListener('click', retryFailedUploads);
  zone.addEventListener('click', e => {
    if (e.target === input) return;
    // 某些桌面浏览器虽然实现了 showPicker，但对当前隐藏 input 调用会失败。
    // 这里优先尝试 showPicker，失败时立即回退到 input.click()。
    try {
      if (typeof input.showPicker === 'function') {
        input.showPicker();
        return;
      }
    } catch (_) {
      // ignore and fallback
    }
    input.click();
  });
  zone.addEventListener('dragover', e => { e.preventDefault(); zone.classList.add('drag-over'); });
  zone.addEventListener('dragleave', () => zone.classList.remove('drag-over'));
  zone.addEventListener('drop', e => { e.preventDefault(); zone.classList.remove('drag-over'); handleFiles(e.dataTransfer.files); });
  input.addEventListener('change', () => { if (input.files.length) handleFiles(input.files); });
}
async function handleFiles(fileList) {
  const files = [...fileList].filter(f => f.type.startsWith('image/'));
  if (!files.length) return;
  const queue = $('#upload-queue');
  for (const file of files) {
    const id = uid();
    const job = { id, file, status: 'queued' };
    state.uploadJobs.push(job);
    const row = el('div', 'upload-item');
    row.id = `upload-row-${id}`;
    row.innerHTML = `<span class="up-name">${file.name}</span><div style="flex:1"><div class="progress-bar"><div class="progress-fill" style="width:0%" id="prog-${id}"></div></div></div><span class="up-status" id="stat-${id}">等待中</span><button class="btn btn-sm" id="retry-${id}" style="display:none">重传</button>`;
    queue.appendChild(row);
    const retryBtn = row.querySelector(`#retry-${id}`);
    retryBtn.addEventListener('click', () => retrySingleUpload(id));
  }

  runUploadQueue();
}

async function runUploadQueue() {
  if (state.uploadRunning) return;
  state.uploadRunning = true;
  try {
    for (;;) {
      const job = state.uploadJobs.find(j => j.status === 'queued');
      if (!job) break;
      await uploadFile(job.file, job.id, job);
    }
  } finally {
    state.uploadRunning = false;
    updateRetryFailedButton();
  }
}

async function uploadFile(file, id, job) {
  const prog = $(`#prog-${id}`);
  const stat = $(`#stat-${id}`);
  const retryBtn = $(`#retry-${id}`);
  if (job) job.status = 'uploading';
  if (retryBtn) retryBtn.style.display = 'none';
  if (stat) { stat.textContent = '上传中'; stat.className = 'up-status'; }
  const fd = new FormData();
  fd.append('photo', file);
  // 传递浏览器 File 对象的本地最后修改时间，供后端在无 EXIF 时作为回退时间。
  if (file.lastModified) {
    fd.append('client_last_modified_ms', String(file.lastModified));
  }
  try {
    await new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      xhr.open('POST', '/api/photos/upload');
      xhr.upload.onprogress = e => { if (prog && e.lengthComputable) prog.style.width = (e.loaded / e.total * 100) + '%'; };
      xhr.onload = () => { if (xhr.status === 201) resolve(); else { try { reject(JSON.parse(xhr.responseText)); } catch { reject({ error: xhr.statusText }); } } };
      xhr.onerror = () => reject({ error: '网络错误' });
      xhr.send(fd);
    });
    if (prog) prog.style.width = '100%';
    if (job) job.status = 'done';
    if (stat) { stat.textContent = '完成'; stat.className = 'up-status done'; }
  } catch(e) {
    if (job) job.status = 'failed';
    if (stat) { stat.textContent = e.error || '失败'; stat.className = 'up-status error'; }
    if (retryBtn) retryBtn.style.display = '';
  }
  updateRetryFailedButton();
}

function updateRetryFailedButton() {
  const btn = $('#retry-failed-btn');
  if (!btn) return;
  const failedCount = state.uploadJobs.filter(j => j.status === 'failed').length;
  btn.style.display = failedCount > 0 ? '' : 'none';
  btn.textContent = failedCount > 0 ? `重传失败项（${failedCount}）` : '重传失败项';
}

function retrySingleUpload(id) {
  const job = state.uploadJobs.find(j => j.id === id);
  if (!job) return;
  job.status = 'queued';
  const prog = $(`#prog-${id}`);
  const stat = $(`#stat-${id}`);
  const retryBtn = $(`#retry-${id}`);
  if (prog) prog.style.width = '0%';
  if (stat) { stat.textContent = '等待重传'; stat.className = 'up-status'; }
  if (retryBtn) retryBtn.style.display = 'none';
  runUploadQueue();
}

function retryFailedUploads() {
  let found = false;
  state.uploadJobs.forEach(job => {
    if (job.status === 'failed') {
      found = true;
      job.status = 'queued';
      const prog = $(`#prog-${job.id}`);
      const stat = $(`#stat-${job.id}`);
      const retryBtn = $(`#retry-${job.id}`);
      if (prog) prog.style.width = '0%';
      if (stat) { stat.textContent = '等待重传'; stat.className = 'up-status'; }
      if (retryBtn) retryBtn.style.display = 'none';
    }
  });
  if (found) runUploadQueue();
}

// ── 新建相册模态框 ────────────────────────────────────
function renderCreateAlbumModal() {
  return `<div class="modal-overlay" id="create-album-modal">
  <div class="modal">
    <div class="modal-title">新建相册</div>
    <div class="form-group"><label class="form-label">相册名称</label><input class="input" id="album-name-input" placeholder="输入相册名称" maxlength="50"></div>
    <div class="form-group"><label class="form-label">描述（可选）</label><input class="input" id="album-desc-input" placeholder="输入描述"></div>
    <div class="modal-footer">
      <button class="btn" id="cancel-album-btn">取消</button>
      <button class="btn btn-primary" id="confirm-album-btn">创建</button>
    </div>
  </div>
</div>`;
}
function openCreateAlbumModal() {
  $('#create-album-modal').classList.add('open');
  $('#album-name-input').value = '';
  $('#album-desc-input').value = '';
  $('#cancel-album-btn').onclick = () => $('#create-album-modal').classList.remove('open');
  $('#confirm-album-btn').onclick = createAlbum;
}
async function createAlbum() {
  const name = $('#album-name-input').value.trim();
  if (!name) { alert('请输入相册名称'); return; }
  try {
    await api.post('/api/albums', { name, description: $('#album-desc-input').value.trim() });
    $('#create-album-modal').classList.remove('open');
    // 通过 switchView 而不是直接 render，确保菜单高亮和 hash 保持一致。
    switchView('albums');
  } catch(e) { alert('创建失败: ' + (e.error || e)); }
}

// ── 相册选择弹窗 (b-3) ───────────────────────────────
function renderAlbumPickerModal() {
  return `<div class="modal-overlay" id="album-picker-modal">
  <div class="modal" style="width:480px">
    <div class="modal-title">${icons.album} 选择相册</div>
    <div class="album-picker-grid" id="album-picker-grid"></div>
    <div style="font-size:.8rem;color:var(--text2);margin-top:10px" id="album-picker-hint"></div>
    <div class="modal-footer">
      <button class="btn" id="album-picker-cancel">取消</button>
      <button class="btn btn-primary" id="album-picker-confirm">添加</button>
    </div>
  </div>
</div>`;
}

let _pickerPhotoIds = null;
let _pickerSelected = null;

async function openAlbumPickerModal(photoIds) {
  // photoIds: null=用已选集合, 数组=指定图片
  _pickerPhotoIds = photoIds;
  _pickerSelected = null;
  const modal = $('#album-picker-modal');
  const cancelBtn = $('#album-picker-cancel');
  const confirmBtn = $('#album-picker-confirm');
  modal.classList.add('open');
  $('#album-picker-hint').textContent = '';

  // 无论后续加载结果如何，都先绑定好按钮，避免“无相册时只能刷新”的死状态。
  cancelBtn.disabled = false;
  cancelBtn.onclick = () => modal.classList.remove('open');
  confirmBtn.disabled = false;
  confirmBtn.textContent = '添加';
  confirmBtn.onclick = confirmAlbumPicker;

  const grid = $('#album-picker-grid');
  grid.innerHTML = '<div style="padding:16px;color:var(--text2)">加载中…</div>';

  try {
    const albums = await api.get('/api/albums');
    if (!albums || !albums.length) {
      grid.innerHTML = `<div style="padding:16px;color:var(--text2)">还没有相册，请先新建相册</div>`;
      $('#album-picker-hint').textContent = '你可以直接在当前弹窗里去创建相册。';
      confirmBtn.textContent = '去创建相册';
      confirmBtn.onclick = () => {
        modal.classList.remove('open');
        openCreateAlbumModal();
      };
      return;
    }
    grid.innerHTML = '';
    albums.forEach(a => {
      const item = el('div', 'album-picker-item');
      item.innerHTML = `<div class="album-picker-cover">${icons.photo}</div><div class="album-picker-name">${a.name} (${a.photo_count||0})</div>`;
      item.addEventListener('click', () => {
        $$('.album-picker-item.picked').forEach(i => i.classList.remove('picked'));
        item.classList.add('picked');
        _pickerSelected = a;
      });
      grid.appendChild(item);
    });
  } catch(e) { grid.innerHTML = `<div style="color:var(--danger)">加载失败</div>`; }

}

async function confirmAlbumPicker() {
  if (!_pickerSelected) { $('#album-picker-hint').textContent = '请先选择一个相册'; return; }
  const album = _pickerSelected;
  const ids = _pickerPhotoIds || [...state.selected];
  if (!ids.length) { $('#album-picker-hint').textContent = '没有选中的图片'; return; }

  $('#album-picker-confirm').disabled = true;
  let ok = 0, fail = 0;
  for (const id of ids) {
    try { await api.post(`/api/albums/${album.id}/photos`, { photo_id: id }); ok++; }
    catch(e) { fail++; }
  }
  $('#album-picker-confirm').disabled = false;
  $('#album-picker-modal').classList.remove('open');
  alert(`已添加 ${ok} 张到「${album.name}」${fail ? `，${fail} 张失败` : ''}`);
  if (_pickerPhotoIds === null) clearSelection();
}

// ── 分享模态框 ────────────────────────────────────────
function renderShareModal() {
  return `<div class="modal-overlay" id="share-modal">
  <div class="modal">
    <div class="modal-title">${icons.share} 创建分享链接</div>
    <div class="form-group">
      <label class="form-label">过期时间</label>
      <select class="input" id="share-expires">
        <option value="0">永不过期</option>
        <option value="7">7 天</option>
        <option value="30">30 天</option>
        <option value="90">90 天</option>
      </select>
    </div>
    <div id="share-result" style="margin-top:12px;display:none">
      <label class="form-label">分享链接</label>
      <input class="input" id="share-link-input" readonly style="cursor:pointer">
      <p style="font-size:.8rem;color:var(--text2);margin-top:4px">点击链接复制</p>
    </div>
    <div class="modal-footer">
      <button class="btn" id="share-cancel-btn">关闭</button>
      <button class="btn btn-primary" id="share-confirm-btn">生成链接</button>
    </div>
  </div>
</div>`;
}
let _shareTarget = null;
function openShareModal(type, targetId) {
  _shareTarget = { type, targetId };
  $('#share-modal').classList.add('open');
  $('#share-result').style.display = 'none';
  $('#share-cancel-btn').onclick = () => { $('#share-modal').classList.remove('open'); };
  $('#share-confirm-btn').onclick = generateShareLink;
}
async function generateShareLink() {
  if (!_shareTarget) return;
  const days = parseInt($('#share-expires').value);
  const body = { type: _shareTarget.type, target_id: _shareTarget.targetId };
  if (days > 0) body.expires_in_days = days;
  try {
    const link = await api.post('/api/shares', body);
    // 更新 shareMap
    const key = `${link.type}:${link.target_id}`;
    if (!state.shareMap[key]) state.shareMap[key] = [];
    state.shareMap[key].push(link);

    const url = `${location.origin}/s/${link.token}`;
    const input = $('#share-link-input');
    input.value = url;
    $('#share-result').style.display = '';
    const newInput = input.cloneNode(true);
    input.parentNode.replaceChild(newInput, input);
    newInput.addEventListener('click', () => navigator.clipboard.writeText(url).then(() => alert('已复制链接')));
  } catch(e) { alert('生成失败: ' + (e.error || e)); }
}

// ── 分享列表弹窗 (b-2) ───────────────────────────────
function renderShareListModal() {
  return `<div class="modal-overlay" id="share-list-modal">
  <div class="modal" style="width:480px">
    <div class="modal-title">${icons.share} 管理分享链接</div>
    <div id="share-list-content"></div>
    <div class="modal-footer">
      <button class="btn" id="share-list-close">关闭</button>
      <button class="btn btn-primary" id="share-list-add">新建分享…</button>
    </div>
  </div>
</div>`;
}

let _shareListTarget = null;
function openShareListModal(type, targetId) {
  _shareListTarget = { type, targetId };
  const modal = $('#share-list-modal');
  modal.classList.add('open');
  renderShareList();
  $('#share-list-close').onclick = () => modal.classList.remove('open');
  $('#share-list-add').onclick = () => {
    modal.classList.remove('open');
    openShareModal(type, targetId);
  };
}

function renderShareList() {
  const key   = `${_shareListTarget.type}:${_shareListTarget.targetId}`;
  const links = state.shareMap[key] || [];
  const wrap  = $('#share-list-content');
  if (!links.length) {
    wrap.innerHTML = `<p style="color:var(--text2);padding:12px 0">暂无分享链接</p>`;
    return;
  }
  wrap.innerHTML = '';
  links.forEach(l => {
    const url = `${location.origin}/s/${l.token}`;
    const exp = l.expires_at ? `过期：${formatDate(l.expires_at)}` : '永不过期';
    const row = el('div');
    row.style.cssText = 'display:flex;align-items:center;gap:8px;padding:8px 0;border-bottom:1px solid var(--border);font-size:.85rem;';
    row.innerHTML = `
      <div style="flex:1;overflow:hidden">
        <div style="font-weight:600;white-space:nowrap;overflow:hidden;text-overflow:ellipsis">${url}</div>
        <div style="color:var(--text2);font-size:.75rem;margin-top:2px">${exp}</div>
      </div>
      <button class="btn btn-sm" data-copy="${url}">复制</button>
      <button class="btn btn-sm btn-danger" data-del="${l.id}">删除</button>`;
    row.querySelector('[data-copy]').addEventListener('click', e => {
      navigator.clipboard.writeText(e.target.dataset.copy).then(() => alert('已复制'));
    });
    row.querySelector('[data-del]').addEventListener('click', async e => {
      const id = parseInt(e.target.dataset.del);
      try {
        await api.del(`/api/shares/${id}`);
        // 从 shareMap 移除
        const key2 = `${_shareListTarget.type}:${_shareListTarget.targetId}`;
        state.shareMap[key2] = (state.shareMap[key2] || []).filter(x => x.id !== id);
        renderShareList();
      } catch(ex) { alert('删除失败'); }
    });
    wrap.appendChild(row);
  });
}

// ── 登出 ──────────────────────────────────────────────
async function logout() {
  await fetch('/api/auth/logout', { method: 'POST' });
  location.reload();
}

// ── 启动 ──────────────────────────────────────────────
initTheme();

// c-2: 从 hash 恢复视图，支持刷新后保持页面
function getHashView() {
  const h = location.hash.replace('#', '');
  if (['timeline','albums','trash'].includes(h)) {
    return { view: h, albumID: null };
  }
  if (h.startsWith('album/')) {
    const id = Number(h.split('/')[1]);
    if (Number.isFinite(id) && id > 0) {
      return { view: 'album-detail', albumID: id };
    }
  }
  return { view: 'timeline', albumID: null };
}
function setHashView(view, albumID = null) {
  if (view === 'album-detail' && albumID) {
    history.replaceState(null, '', `#album/${albumID}`);
    return;
  }
  if (['timeline','albums','trash'].includes(view)) {
    history.replaceState(null, '', '#' + view);
  }
}
const initialRoute = getHashView();
state.view = initialRoute.view;
state.currentAlbumID = initialRoute.albumID;
renderApp();
