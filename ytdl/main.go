package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// --------------------------------------------------------------------------
// SSE broker: fans out log lines to all connected browser clients
// --------------------------------------------------------------------------

type broker struct {
	mu      sync.Mutex
	clients map[chan string]struct{}
}

func newBroker() *broker { return &broker{clients: make(map[chan string]struct{})} }

func (b *broker) subscribe() chan string {
	ch := make(chan string, 64)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *broker) unsubscribe(ch chan string) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
}

func (b *broker) publish(msg string) {
	b.mu.Lock()
	for ch := range b.clients {
		select {
		case ch <- msg:
		default:
		}
	}
	b.mu.Unlock()
}

// --------------------------------------------------------------------------
// Globals
// --------------------------------------------------------------------------

var (
	events      = newBroker()
	downloading sync.Mutex // only one download at a time
)

// --------------------------------------------------------------------------
// HTML page (embedded as a string literal)
// --------------------------------------------------------------------------

const page = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>YT Playlist Downloader</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: 'Segoe UI', system-ui, sans-serif;
    background: #0f0f13;
    color: #e0e0e0;
    display: flex;
    justify-content: center;
    padding: 40px 16px;
    min-height: 100vh;
  }
  .card {
    background: #1a1a24;
    border: 1px solid #2e2e3e;
    border-radius: 14px;
    padding: 32px 36px;
    width: 100%;
    max-width: 620px;
    height: fit-content;
  }
  h1 { font-size: 1.5rem; margin-bottom: 24px; color: #fff; }
  h1 span { color: #e74c3c; }
  label { display: block; font-size: .85rem; color: #aaa; margin-bottom: 6px; }
  input[type=text], input[type=url] {
    width: 100%;
    background: #0f0f13;
    border: 1px solid #3a3a4e;
    border-radius: 8px;
    padding: 10px 14px;
    color: #fff;
    font-size: .95rem;
    outline: none;
    transition: border .2s;
  }
  input:focus { border-color: #e74c3c; }
  .row { display: flex; gap: 12px; margin-top: 20px; align-items: flex-end; }
  .field { flex: 1; }
  .fmt-group {
    display: flex;
    gap: 10px;
    margin-top: 20px;
  }
  .fmt-btn {
    flex: 1;
    padding: 12px;
    border-radius: 8px;
    border: 2px solid #3a3a4e;
    background: #0f0f13;
    color: #aaa;
    font-size: 1rem;
    cursor: pointer;
    transition: all .15s;
    text-align: center;
    user-select: none;
  }
  .fmt-btn.active { border-color: #e74c3c; color: #fff; background: #2a0e0e; }
  .dir-row {
    display: flex;
    gap: 8px;
    margin-top: 20px;
    align-items: center;
  }
  .dir-row input { flex: 1; }
  .pick-btn {
    background: #2e2e3e;
    border: 1px solid #3a3a4e;
    color: #ccc;
    border-radius: 8px;
    padding: 10px 14px;
    cursor: pointer;
    white-space: nowrap;
    font-size: .9rem;
  }
  .pick-btn:hover { background: #3a3a4e; }
  .dl-btn {
    width: 100%;
    margin-top: 24px;
    padding: 14px;
    background: #e74c3c;
    color: #fff;
    border: none;
    border-radius: 10px;
    font-size: 1.05rem;
    font-weight: 600;
    cursor: pointer;
    transition: background .15s, opacity .15s;
  }
  .dl-btn:hover { background: #c0392b; }
  .dl-btn:disabled { opacity: .5; cursor: not-allowed; }
  .progress-wrap {
    margin-top: 16px;
    background: #0f0f13;
    border-radius: 6px;
    height: 8px;
    overflow: hidden;
  }
  #bar {
    height: 100%;
    background: #e74c3c;
    width: 0%;
    transition: width .4s;
  }
  #status {
    margin-top: 8px;
    font-size: .82rem;
    color: #aaa;
    min-height: 1.2em;
  }
  #log {
    margin-top: 16px;
    background: #0a0a0f;
    border: 1px solid #2e2e3e;
    border-radius: 8px;
    padding: 12px 14px;
    height: 180px;
    overflow-y: auto;
    font-family: monospace;
    font-size: .8rem;
    color: #9be;
    white-space: pre-wrap;
  }
  .done-msg { color: #2ecc71 !important; font-weight: bold; }
  .err-msg  { color: #e74c3c !important; }
</style>
</head>
<body>
<div class="card">
  <h1>YT <span>Playlist</span> Downloader</h1>

  <label>Playlist or Video URL</label>
  <input type="url" id="url" placeholder="https://www.youtube.com/playlist?list=..." />

  <div style="margin-top:20px">
    <label>Format</label>
    <div class="fmt-group">
      <div class="fmt-btn active" id="btn-mp4" onclick="setFmt('mp4')">🎬 MP4 (video)</div>
      <div class="fmt-btn"        id="btn-mp3" onclick="setFmt('mp3')">🎵 MP3 (audio only)</div>
    </div>
  </div>

  <div style="margin-top:20px">
    <label>Save to folder</label>
    <div class="dir-row">
      <input type="text" id="outdir" value="" />
      <button class="pick-btn" onclick="pickDir()">Browse…</button>
    </div>
  </div>

  <button class="dl-btn" id="dlbtn" onclick="startDownload()">Download</button>

  <div class="progress-wrap"><div id="bar"></div></div>
  <div id="status">Ready.</div>
  <div id="log"></div>
</div>

<script>
let fmt = 'mp4';

function setFmt(f) {
  fmt = f;
  document.getElementById('btn-mp4').className = 'fmt-btn' + (f==='mp4' ? ' active' : '');
  document.getElementById('btn-mp3').className = 'fmt-btn' + (f==='mp3' ? ' active' : '');
}

async function pickDir() {
  const res = await fetch('/pickdir');
  const data = await res.json();
  if (data.dir) document.getElementById('outdir').value = data.dir;
}

function log(msg, cls) {
  const el = document.getElementById('log');
  const line = document.createElement('div');
  if (cls) line.className = cls;
  line.textContent = msg;
  el.appendChild(line);
  el.scrollTop = el.scrollHeight;
}

function setStatus(msg) { document.getElementById('status').textContent = msg; }
function setBar(pct)    { document.getElementById('bar').style.width = pct + '%'; }

async function startDownload() {
  const url = document.getElementById('url').value.trim();
  const dir = document.getElementById('outdir').value.trim();
  if (!url) { alert('Please enter a URL.'); return; }
  if (!dir) { alert('Please choose a save folder.'); return; }

  document.getElementById('dlbtn').disabled = true;
  document.getElementById('log').innerHTML = '';
  setBar(0);
  setStatus('Starting…');

  const es = new EventSource('/progress');
  es.onmessage = e => {
    const d = JSON.parse(e.data);
    if (d.type === 'log')    log(d.msg);
    if (d.type === 'status') setStatus(d.msg);
    if (d.type === 'bar')    setBar(d.pct);
    if (d.type === 'done')   { log(d.msg, 'done-msg'); setStatus(d.msg); setBar(100); es.close(); document.getElementById('dlbtn').disabled = false; }
    if (d.type === 'error')  { log(d.msg, 'err-msg');  setStatus('Error — check log.'); es.close(); document.getElementById('dlbtn').disabled = false; }
  };

  fetch('/download', {
    method: 'POST',
    headers: {'Content-Type':'application/json'},
    body: JSON.stringify({url, fmt, dir})
  });
}

// set default Downloads folder on load
fetch('/defaultdir').then(r=>r.json()).then(d=>{ document.getElementById('outdir').value = d.dir; });
</script>
</body>
</html>`

// --------------------------------------------------------------------------
// HTTP handlers
// --------------------------------------------------------------------------

func handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, page)
}

func handleDefaultDir(w http.ResponseWriter, r *http.Request) {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, "Downloads")
	os.MkdirAll(dir, 0755)
	json.NewEncoder(w).Encode(map[string]string{"dir": dir})
}

func handlePickDir(w http.ResponseWriter, r *http.Request) {
	// Use zenity if available, fall back to kdialog, then xdg-open approach
	dir := ""
	for _, tool := range []string{"zenity", "kdialog"} {
		path, err := exec.LookPath(tool)
		if err != nil {
			continue
		}
		var out []byte
		if tool == "zenity" {
			out, err = exec.Command(path, "--file-selection", "--directory", "--title=Choose save folder").Output()
		} else {
			out, err = exec.Command(path, "--getexistingdirectory").Output()
		}
		if err == nil {
			dir = strings.TrimSpace(string(out))
			break
		}
	}
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, "Downloads")
	}
	json.NewEncoder(w).Encode(map[string]string{"dir": dir})
}

func handleProgress(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", 500)
		return
	}

	ch := events.subscribe()
	defer events.unsubscribe(ch)

	// keep-alive ticker
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg, open := <-ch:
			if !open {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

type downloadReq struct {
	URL string `json:"url"`
	Fmt string `json:"fmt"`
	Dir string `json:"dir"`
}

func emit(t string, extra map[string]interface{}) {
	m := map[string]interface{}{"type": t}
	for k, v := range extra {
		m[k] = v
	}
	b, _ := json.Marshal(m)
	events.publish(string(b))
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	var req downloadReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", 400)
		return
	}
	if req.URL == "" || req.Dir == "" {
		http.Error(w, "missing fields", 400)
		return
	}
	if req.Fmt != "mp3" && req.Fmt != "mp4" {
		req.Fmt = "mp4"
	}

	go runDownload(req)
	w.WriteHeader(http.StatusAccepted)
}

func runDownload(req downloadReq) {
	downloading.Lock()
	defer downloading.Unlock()

	os.MkdirAll(req.Dir, 0755)

	// Locate python3 that has yt_dlp installed
	python, err := findPython()
	if err != nil {
		emit("error", map[string]interface{}{"msg": "Cannot find yt-dlp. Install via: pip install yt-dlp"})
		return
	}

	outtmpl := filepath.Join(req.Dir, "%(playlist_index)s - %(title)s.%(ext)s")

	var args []string
	if req.Fmt == "mp3" {
		args = []string{"-m", "yt_dlp",
			"-f", "bestaudio/best",
			"-x", "--audio-format", "mp3", "--audio-quality", "192K",
			"--newline",
			"-o", outtmpl,
			"--ignore-errors",
			req.URL,
		}
	} else {
		args = []string{"-m", "yt_dlp",
			"-f", "bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best",
			"--merge-output-format", "mp4",
			"--newline",
			"-o", outtmpl,
			"--ignore-errors",
			req.URL,
		}
	}

	emit("status", map[string]interface{}{"msg": fmt.Sprintf("Starting %s download…", strings.ToUpper(req.Fmt))})
	emit("log", map[string]interface{}{"msg": fmt.Sprintf("▶  yt-dlp [%s]  →  %s", strings.ToUpper(req.Fmt), req.Dir)})

	cmd := exec.Command(python, args...)
	stdout, _ := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout // merge stderr into stdout pipe

	if err := cmd.Start(); err != nil {
		emit("error", map[string]interface{}{"msg": "Failed to start yt-dlp: " + err.Error()})
		return
	}

	total, done := 0, 0
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()

		// count total items from playlist lines
		if strings.Contains(line, "Downloading item") {
			// "[download] Downloading item X of Y"
			var x, y int
			if n, _ := fmt.Sscanf(line, "[download] Downloading item %d of %d", &x, &y); n == 2 {
				total = y
				emit("status", map[string]interface{}{"msg": fmt.Sprintf("Item %d of %d", x, y)})
			}
		}

		if strings.Contains(line, "[download]") && strings.Contains(line, "%") {
			// progress percentage line
			var pct float64
			fmt.Sscanf(extractAfter(line, "[download]"), "%f%%", &pct)
			if total > 0 && pct > 0 {
				overall := (float64(done)/float64(total))*100 + pct/float64(total)
				emit("bar", map[string]interface{}{"pct": int(overall)})
			}
			emit("status", map[string]interface{}{"msg": strings.TrimSpace(line)})
			continue // don't spam log with every % line
		}

		if strings.HasPrefix(line, "[download] Destination:") ||
			strings.HasPrefix(line, "[ExtractAudio]") ||
			strings.HasPrefix(line, "[Merger]") ||
			strings.HasPrefix(line, "[ffmpeg]") ||
			strings.HasPrefix(line, "ERROR") ||
			strings.HasPrefix(line, "WARNING") {
			emit("log", map[string]interface{}{"msg": line})
		}

		if strings.Contains(line, "has already been downloaded") || strings.Contains(line, "Destination:") {
			done++
			if total > 0 {
				emit("bar", map[string]interface{}{"pct": int(float64(done) / float64(total) * 100)})
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		emit("log", map[string]interface{}{"msg": "yt-dlp exited: " + err.Error()})
	}

	emit("done", map[string]interface{}{"msg": fmt.Sprintf("Done! Files saved to: %s", req.Dir)})
}

func extractAfter(s, prefix string) string {
	idx := strings.Index(s, prefix)
	if idx < 0 {
		return s
	}
	return strings.TrimSpace(s[idx+len(prefix):])
}

// findPython returns the first python3 binary that can import yt_dlp.
func findPython() (string, error) {
	candidates := []string{
		"/usr/local/bin/python3",
		"/usr/bin/python3",
		"python3",
		"python",
	}
	for _, p := range candidates {
		out, err := exec.Command(p, "-c", "import yt_dlp").CombinedOutput()
		if err == nil && len(out) == 0 {
			return p, nil
		}
	}
	return "", fmt.Errorf("yt_dlp not found in any python3 installation")
}

// openBrowser opens the given URL in the default system browser.
func openBrowser(url string) {
	time.Sleep(200 * time.Millisecond)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	if cmd != nil {
		cmd.Start()
	}
}

// --------------------------------------------------------------------------
// main
// --------------------------------------------------------------------------

func main() {
	// Pick a free port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to bind:", err)
		os.Exit(1)
	}
	addr := fmt.Sprintf("http://127.0.0.1:%d", ln.Addr().(*net.TCPAddr).Port)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/defaultdir", handleDefaultDir)
	mux.HandleFunc("/pickdir", handlePickDir)
	mux.HandleFunc("/progress", handleProgress)
	mux.HandleFunc("/download", handleDownload)

	fmt.Printf("YT Playlist Downloader running at %s\n", addr)
	go openBrowser(addr)

	if err := http.Serve(ln, mux); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
