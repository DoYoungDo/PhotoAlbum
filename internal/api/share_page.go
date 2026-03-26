package api

import "github.com/gin-gonic/gin"

const sharePageHTML = `<!doctype html>
<html lang="zh-CN" data-theme="">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>分享 - PhotoAlbum</title>
  <link rel="stylesheet" href="/static/app.css">
</head>
<body>
<script>
  (function(){
    var t = localStorage.getItem('theme') || (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
    document.documentElement.dataset.theme = t;
  })();
</script>
<div class="share-wrap">
  <div class="share-header">
    <svg width="32" height="32" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24" style="margin:0 auto 8px">
      <rect x="3" y="3" width="18" height="18" rx="2"/>
      <circle cx="8.5" cy="8.5" r="1.5"/>
      <polyline points="21 15 16 10 5 21"/>
    </svg>
    <strong id="share-title">加载中…</strong>
    <span id="share-sub"></span>
  </div>
  <div id="share-content" style="width:100%;max-width:960px"></div>
</div>
<script>
(function() {
  var token = location.pathname.split('/').pop();
  function escapeHtml(value) {
    return String(value || '')
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  }
  async function load() {
    try {
      var r = await fetch('/api/s/' + token);
      if (!r.ok) { document.getElementById('share-title').textContent = '链接无效或已过期'; return; }
      var link = await r.json();
      if (link.type === 'photo') {
        document.getElementById('share-title').textContent = link.target_media_kind === 'video' ? '分享的视频' : '分享的照片';
        var mediaUrl = '/media/s/' + token + '/' + (link.target_uuid || link.target_id);
        var mediaHtml = link.target_media_kind === 'video'
          ? '<video controls playsinline preload="metadata" src="' + mediaUrl + '" style="max-width:100%;border-radius:12px;box-shadow:0 4px 24px rgba(0,0,0,.15)"></video>'
          : '<img src="' + mediaUrl + '" alt="' + escapeHtml(link.target_original_name || 'shared-media') + '" style="max-width:100%;border-radius:12px;box-shadow:0 4px 24px rgba(0,0,0,.15)">';
        document.getElementById('share-content').innerHTML =
          '<div style="margin-bottom:12px"><a class="btn btn-primary" href="/s/' + token + '/download">下载原文件</a></div>' + mediaHtml;
      } else if (link.type === 'album') {
        document.getElementById('share-title').textContent = '分享的相册';
        var pr = await fetch('/api/s/' + token + '/photos');
        if (pr.ok) {
          var pg = await pr.json();
          var grid = document.createElement('div');
          grid.className = 'photo-grid';
          (pg.photos || []).forEach(function(p) {
            var d = document.createElement('div');
            d.className = 'photo-thumb';
            if (p.media_kind === 'video') {
              d.innerHTML = '<video muted playsinline preload="metadata" src="/media/s/' + token + '/' + p.uuid + '"></video>';
            } else {
              d.innerHTML = '<img loading="lazy" src="/media/s/' + token + '/' + p.uuid + '" alt="' + escapeHtml(p.original_name) + '">';
            }
            grid.appendChild(d);
          });
          document.getElementById('share-content').appendChild(grid);
        }
      }
    } catch(e) { document.getElementById('share-title').textContent = '加载失败'; }
  }
  load();
})();
</script>
</body>
</html>`

func handleSharePage() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Data(200, "text/html; charset=utf-8", []byte(sharePageHTML))
	}
}
