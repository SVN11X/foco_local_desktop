package windows

import (
	"fmt"
	"focolocal-desktop/internal/limit"
	"html"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unsafe"
)

// No browser extensions, DNS rules, proxy or process termination.
// Once a *foreground, freshly scanned, non-automated* browser tab reaches its
// daily limit, replace its contents with a local explanatory page. Navigating
// the current tab stops its media without muting unrelated browser tabs.
// WARNING: This can discard unsaved text on the blocked page. The user can
// switch to overlay-only mode in the desktop hub.
var pSendInput = user32.NewProc("SendInput")

const (
	inputKeyboard = 1
	keyUp         = 0x0002
	keyUnicode    = 0x0004
	vkControl     = 0x11
	vkL           = 0x4c
	vkReturn      = 0x0d
)

// INPUT is a 40-byte Win64 struct. Its union is 32 bytes (MOUSEINPUT),
// even though KEYBDINPUT is only 24 bytes. The final eight bytes are padding.
type winKeyboardInput struct {
	Kind  uint32
	_     uint32
	VK    uint16
	Scan  uint16
	Flags uint32
	Time  uint32
	Extra uintptr
	_     [8]byte
}

func vkInput(key uint16, released bool) winKeyboardInput {
	input := winKeyboardInput{Kind: inputKeyboard, VK: key}
	if released {
		input.Flags = keyUp
	}
	return input
}

func unicodeInput(ch uint16, released bool) winKeyboardInput {
	input := winKeyboardInput{Kind: inputKeyboard, Scan: ch, Flags: keyUnicode}
	if released {
		input.Flags |= keyUp
	}
	return input
}

// SendInput is an atomic batch to the foreground window's keyboard queue.
// We never ask to elevate privileges, and refuse to act when the user has
// switched to another window between the last scan and this tick.
func sendKeyboardBatch(target uintptr, pid uint32, events []winKeyboardInput) bool {
	if len(events) == 0 || fg() != target || windowPID(target) != pid {
		return false
	}
	sent := call(pSendInput, uintptr(len(events)), uintptr(unsafe.Pointer(&events[0])), unsafe.Sizeof(events[0]))
	runtime.KeepAlive(events)
	return sent == uintptr(len(events))
}

// Browsers apply Ctrl+L asynchronously. Sending the shortcut and first URI
// character in one SendInput batch occasionally loses the leading 'f' of
// file:///... and turns the destination into a web search. Synchronize phases.
// No clipboard changes, extensions, global DNS rules or process termination.
func navigateForegroundTab(target, destination uintptr, urlString string) bool {
	pid := uint32(destination)
	if target == 0 || fg() != target || pid == 0 || urlString == "" {
		return false
	}
	if unsafe.Sizeof(winKeyboardInput{}) != 40 {
		return false
	}
	focus := []winKeyboardInput{vkInput(vkControl, false), vkInput(vkL, false), vkInput(vkL, true), vkInput(vkControl, true)}
	if !sendKeyboardBatch(target, pid, focus) {
		return false
	}
	time.Sleep(170 * time.Millisecond)
	events := make([]winKeyboardInput, 0, len(urlString)*2)
	for _, ch := range urlString {
		if ch > 0xffff {
			return false
		}
		events = append(events, unicodeInput(uint16(ch), false), unicodeInput(uint16(ch), true))
	}
	if !sendKeyboardBatch(target, pid, events) {
		return false
	}
	time.Sleep(100 * time.Millisecond)
	return sendKeyboardBatch(target, pid, []winKeyboardInput{vkInput(vkReturn, false), vkInput(vkReturn, true)})
}

func blockedPagePath(configPath, domain string) (string, error) {
	return blockedPagePathWithMessage(configPath, domain, "Este descanso se renueva según la regla que configuraste.")
}
func blockedPagePathWithMessage(configPath, domain, message string) (string, error) {
	// All domains here originate from validated limit rules. A second check
	// prevents unexpected path traversal if the caller ever changes.
	if domain == "" || limit.NormalizeRuleTarget(domain) != domain {
		return "", fmt.Errorf("dominio inválido para pantalla de bloqueo")
	}
	dir := filepath.Dir(configPath)
	if e := os.MkdirAll(dir, 0700); e != nil {
		return "", e
	}
	path := filepath.Join(dir, "bloqueado-"+strings.ReplaceAll(domain, "/", "-")+".html")
	content := blockedPageHTMLMessage(limit.RuleLabel(domain), message)
	if e := os.WriteFile(path, []byte(content), 0600); e != nil {
		return "", e
	}
	return path, nil
}

// This self-contained, read-only HTML is never served over the network. It has
// no JavaScript, remote fonts, images, analytics or dynamic permissions.
func blockedPageHTML(domain string) string {
	return blockedPageHTMLMessage(domain, "Tu descanso se renueva según la regla que configuraste.")
}
func blockedPageHTMLMessage(domain, message string) string {
	const page = `<!doctype html>
<html lang="es"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'; object-src 'none'">
<meta name="color-scheme" content="light dark"><title>Tu tiempo de descanso | FocoLocal</title>
<style>
:root{color-scheme:light;--bg:#eaf0fa;--surface:#fff;--text:#172b49;--muted:#5e7090;--line:#d9e3f2;--accent:#386af7;--soft:#eef3ff;--mint:#136f5b}
*{box-sizing:border-box}html,body{height:100%;height:100dvh;width:100%;overflow:hidden;margin:0}
body{font-family:'Segoe UI Variable','Segoe UI',system-ui,sans-serif;color:var(--text);background:radial-gradient(circle at 86% 9%,#cfe1ff,transparent 38%),radial-gradient(circle at 8% 87%,#d2eee8,transparent 39%),var(--bg);display:grid;place-items:center;padding:clamp(8px,2.3vh,24px)}
main{width:min(900px,100%);max-height:calc(100dvh - 16px);background:var(--surface);border:1px solid var(--line);border-radius:clamp(16px,3vh,30px);box-shadow:0 24px 70px #12284818;padding:clamp(18px,3.4vh,42px) clamp(20px,4.6vw,52px);display:flex;flex-direction:column;justify-content:center;gap:clamp(8px,1.4vh,18px);overflow:hidden}
.head{display:flex;align-items:center;justify-content:space-between;gap:14px}.brand{font-size:14px;letter-spacing:.09em;font-weight:850;display:flex;align-items:center;gap:10px}.logo{display:grid;place-items:center;background:var(--soft);color:var(--accent);border-radius:12px;width:36px;height:36px;font-size:22px}.chip{background:#e6f7ee;color:var(--mint);padding:8px 13px;font-size:12px;font-weight:750;border-radius:100px;white-space:nowrap}
.hero{display:flex;align-items:center;gap:18px}.sun{display:grid;place-items:center;font-size:34px;width:65px;height:65px;flex:none;border-radius:22px;background:var(--soft);color:var(--accent)}h1{font-size:clamp(29px,4vw,48px);letter-spacing:-.035em;line-height:1.09;margin:0}
p{font-size:clamp(14px,1.5vw,16px);line-height:1.52;color:var(--muted);margin:0}.domain{color:var(--accent);font-weight:800;overflow-wrap:anywhere}.renew{font-size:14px;font-weight:800;color:var(--text);background:var(--soft);border-radius:12px;padding:11px 15px}
h2{font-size:clamp(15px,2vw,18px);margin:0}.ideas{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:12px}.idea{padding:clamp(12px,1.9vh,18px);border:1px solid var(--line);border-radius:16px;background:var(--soft)}.icon{font-size:22px;display:block;margin-bottom:6px}.idea strong{display:block;font-size:14px;margin-bottom:3px}.idea span:last-child{font-size:13px;line-height:1.35;color:var(--muted)}
footer{display:flex;justify-content:space-between;align-items:center;gap:14px;border-top:1px solid var(--line);padding-top:12px}footer p{font-size:12px;max-width:555px}.action{font-size:13px;font-weight:700;color:var(--accent);text-decoration:none;white-space:nowrap;border:1px solid var(--line);padding:11px 15px;border-radius:12px;background:var(--soft)}.action:focus-visible{outline:3px solid var(--accent);outline-offset:2px}
@media(prefers-color-scheme:dark){:root{color-scheme:dark;--bg:#0e1624;--surface:#1b293d;--text:#f0f6ff;--muted:#bfcee2;--line:#314660;--accent:#9bbaff;--soft:#253953;--mint:#88eac8}body{background:radial-gradient(circle at 86% 9%,#172c58,transparent 43%),radial-gradient(circle at 8% 85%,#123c41,transparent 37%),var(--bg)}.chip{background:#1b463d}}
@media(max-height:780px){main{gap:9px;padding:15px 28px}.sun{width:50px;height:50px;font-size:26px}.idea{padding:10px 13px}.idea .icon{font-size:18px;margin-bottom:3px}.idea strong{margin-bottom:2px}.hero{gap:12px}}
@media(max-height:610px){.sun{display:none}.ideas{gap:7px}.idea span:last-child{display:none}.idea{padding:8px}.hero h1{font-size:clamp(25px,4vw,37px)}.chip{display:none}main{gap:7px;padding:12px 20px}.renew{padding:7px 11px}footer{padding-top:7px}}
@media(max-height:470px){.ideas,h2{display:none}main{padding:9px 16px;gap:6px}.head .logo{width:28px;height:28px}.renew{padding:5px 9px}}
@media(max-width:620px){.ideas{grid-template-columns:1fr}.idea{min-height:0}.idea:nth-child(n+2){display:none}footer{align-items:flex-start;flex-direction:column;gap:6px}.action{padding:8px 10px}main{padding:clamp(10px,2vh,20px)}.chip{display:none}}
@media(prefers-reduced-motion:reduce){*{transition:none!important;animation:none!important}}
</style></head><body><main>
<header class="head"><div class="brand"><span class="logo" aria-hidden="true">◷</span>FOCOLOCAL</div><span class="chip">Pausa digital activada</span></header>
<div class="hero"><span class="sun" aria-hidden="true">☀</span><h1>Tu tiempo también merece espacio.</h1></div>
<p>Alcanzaste el límite que elegiste para <strong class="domain">{{DOMAIN}}</strong>. Detuvimos esta pestaña y su reproducción sin cerrar el navegador ni afectar tus demás aplicaciones.</p>
<div class="renew">{{MESSAGE}}</div>
<h2>Mientras tanto, puedes...</h2>
<div class="ideas">
<div class="idea"><span class="icon" aria-hidden="true">☕</span><strong>Hacer una pausa</strong><span>Respira, camina o descansa la vista unos minutos.</span></div>
<div class="idea"><span class="icon" aria-hidden="true">✦</span><strong>Retomar un proyecto</strong><span>Avanza una tarea pendiente, aunque sea un paso.</span></div>
<div class="idea"><span class="icon" aria-hidden="true">◎</span><strong>Elegir otra actividad</strong><span>Puedes seguir usando sitios sin límite agotado.</span></div>
</div>
<footer><p>¿Necesitas ajustar la regla? Abre FocoLocal desde la bandeja de Windows. Tus automatizaciones permanecen independientes.</p><a class="action" href="about:blank" target="_blank" rel="noopener noreferrer">Abrir pestaña en blanco ↗</a></footer>
</main></body></html>`
	result := strings.ReplaceAll(page, "{{DOMAIN}}", html.EscapeString(domain))
	return strings.ReplaceAll(result, "{{MESSAGE}}", html.EscapeString(message))
}

func pathToFileURL(path string) string {
	p := filepath.ToSlash(path)
	if len(p) >= 2 && p[1] == ':' { // C:/Users/... -> /C:/Users/...
		p = "/" + p
	}
	return (&url.URL{Scheme: "file", Path: p}).String()
}

func (a *App) stopBlockedTab(s Sample, rule string, now time.Time) bool {
	key := fmt.Sprintf("%d:%d:%s", s.Window, s.PID, s.Site)
	if a.lastRedirectKey == key && now.Sub(a.lastRedirectAt) < 5*time.Second {
		return true // scanner can briefly retain the old URL; do not inject twice
	}
	a.mu.Lock()
	confirmed := a.lastSeenBrowser == key && a.seenConsecutive >= 2
	a.mu.Unlock()
	if !confirmed || s.Window == 0 || s.PID == 0 || now.Sub(s.At) > 1800*time.Millisecond || fg() != s.Window || windowPID(s.Window) != s.PID {
		return false
	}
	cfg, _ := a.store.Snapshot(now)
	if cfg.SafeAutomation && isAutomated(s.PID) { // defense in depth
		return false
	}
	message := "Puedes volver a utilizar este sitio cuando tu regla lo permita."
	cfg, _ = a.store.Snapshot(now)
	for _, r := range cfg.Rules {
		if r.Domain != rule {
			continue
		}
		switch r.Period {
		case "day", "":
			message = "El contador se reinicia mañana."
		case "week":
			message = "Tu presupuesto se renueva el próximo lunes."
		case "month":
			message = "Tu presupuesto se renueva el primer día del próximo mes."
		case "blocked":
			message = "Este sitio permanecerá bloqueado hasta que cambies su regla."
		case "schedule":
			message = "Horario permitido: " + r.From + " a " + r.To + " (si aún queda tiempo disponible)."
		case "interval":
			message = fmt.Sprintf("Tómate una pausa de %d minutos por cada %d minutos de uso, si queda presupuesto diario.", r.BreakMinutes, r.SessionMinutes)
		}
		break
	}
	path, err := blockedPagePathWithMessage(a.store.Path, rule, message)
	if err != nil {
		return false
	}
	if !navigateForegroundTab(s.Window, uintptr(s.PID), pathToFileURL(path)) {
		return false
	}
	a.lastRedirectKey, a.lastRedirectAt = key, now
	return true
}
