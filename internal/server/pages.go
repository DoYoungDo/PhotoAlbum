package server

import (
	"html/template"
	"net/http"
)

// ── 主应用页面（需登录）──────────────────────────────
const appTemplate = `<!doctype html>
<html lang="zh-CN" data-theme="">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>PhotoAlbum</title>
  <link rel="stylesheet" href="/static/app.css">
</head>
<body>
  <div id="app"></div>
  <script src="/static/app.js"></script>
</body>
</html>`

// ── 登录页 ───────────────────────────────────────────
const loginTemplate = `<!doctype html>
<html lang="zh-CN" data-theme="">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>登录 - PhotoAlbum</title>
  <link rel="stylesheet" href="/static/app.css">
</head>
<body>
<script>
  (function(){
    var t = localStorage.getItem('theme') || (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
    document.documentElement.dataset.theme = t;
  })();
</script>
<div class="login-wrap">
  <div class="login-card">
    <div class="login-logo">
      <svg width="28" height="28" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
        <rect x="3" y="3" width="18" height="18" rx="2"/>
        <circle cx="8.5" cy="8.5" r="1.5"/>
        <polyline points="21 15 16 10 5 21"/>
      </svg>
      PhotoAlbum
    </div>
    <div class="form-group">
      <label class="form-label" for="username">用户名</label>
      <input class="input" id="username" type="text" autocomplete="username" placeholder="请输入用户名">
    </div>
    <div class="form-group">
      <label class="form-label" for="password">密码</label>
      <input class="input" id="password" type="password" autocomplete="current-password" placeholder="请输入密码">
    </div>
    <button class="btn btn-primary" style="width:100%;justify-content:center" id="login-btn">登录</button>
    <div class="login-error" id="login-error"></div>
  </div>
</div>
<script>
(function() {
  var btn = document.getElementById('login-btn');
  var errEl = document.getElementById('login-error');
  async function doLogin() {
    var u = document.getElementById('username').value.trim();
    var p = document.getElementById('password').value;
    if (!u || !p) { errEl.textContent = '请输入用户名和密码'; return; }
    btn.disabled = true; btn.textContent = '登录中…';
    try {
      var r = await fetch('/api/auth/login', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({username: u, password: p})
      });
      if (r.ok) { location.href = '/'; return; }
      var d = await r.json();
      errEl.textContent = d.error || '登录失败';
    } catch(e) { errEl.textContent = '网络错误'; }
    btn.disabled = false; btn.textContent = '登录';
  }
  btn.addEventListener('click', doLogin);
  document.addEventListener('keydown', function(e){ if(e.key === 'Enter') doLogin(); });
})();
</script>
</body>
</html>`

var (
	appTmpl   = template.Must(template.New("app").Parse(appTemplate))
	loginTmpl = template.Must(template.New("login").Parse(loginTemplate))
)

func serveHTML(w http.ResponseWriter, t *template.Template) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.Execute(w, nil); err != nil {
		// 模板是编译期内联常量，执行失败属于严重错误
		http.Error(w, "内部错误", http.StatusInternalServerError)
	}
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	serveHTML(w, appTmpl)
}
func (s *Server) handleAlbumsPage(w http.ResponseWriter, r *http.Request) {
	serveHTML(w, appTmpl)
}
func (s *Server) handleAlbumDetailPage(w http.ResponseWriter, r *http.Request) {
	serveHTML(w, appTmpl)
}
func (s *Server) handleTrashPage(w http.ResponseWriter, r *http.Request) {
	serveHTML(w, appTmpl)
}
func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	serveHTML(w, loginTmpl)
}
