package windows

import (
	"fmt"
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
	if domain == "" || strings.ContainsAny(domain, `/\\:`) || strings.Contains(domain, "..") {
		return "", fmt.Errorf("dominio inválido para pantalla de bloqueo")
	}
	dir := filepath.Dir(configPath)
	if e := os.MkdirAll(dir, 0700); e != nil {
		return "", e
	}
	path := filepath.Join(dir, "bloqueado-"+domain+".html")
	content := blockedPageHTMLMessage(domain, message)
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
<meta name="color-scheme" content="light">
<title>Tiempo de descanso · FocoLocal</title>
<style>
:root{color-scheme:light;--ink:#142744;--sub:#56657d;--line:#e5ebf5;--blue:#2763df;--mint:#0a8069}
*{box-sizing:border-box}body{margin:0;padding:32px 20px;min-height:100vh;display:grid;place-items:center;background:radial-gradient(circle at 75% 10%,#dbe9ff 0%,transparent 35%),linear-gradient(135deg,#f4f7ff,#eaf7f4);color:var(--ink);font-family:'Segoe UI Variable','Segoe UI',system-ui,sans-serif}
main{width:min(780px,100%);border:1px solid #e5ebf5;border-radius:28px;background:#fff;box-shadow:0 18px 60px #20396918;padding:clamp(26px,6vw,52px)}
.brand{display:flex;align-items:center;justify-content:space-between;gap:12px;font-weight:800;letter-spacing:.04em;font-size:15px}.brandMark{display:inline-grid;place-items:center;width:35px;height:35px;border-radius:12px;background:#e9f1ff;color:var(--blue);margin-right:10px}
.badge{font-size:12px;font-weight:700;border-radius:20px;padding:9px 14px;background:#e8f8f2;color:var(--mint)}
.art{display:grid;place-items:center;margin:36px 0 18px;width:74px;height:74px;border-radius:24px;background:#e9f1ff;color:var(--blue);font-size:36px}
h1{font-size:clamp(2rem,5vw,3.3rem);letter-spacing:-.035em;line-height:1.12;margin:0 0 15px}p{color:var(--sub);line-height:1.7;font-size:16px;margin:0 0 14px}
.domain{color:var(--blue);overflow-wrap:anywhere}h2{font-size:18px;margin:28px 0 16px}.grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:14px}.idea{border:1px solid var(--line);background:#f9fbff;border-radius:16px;padding:19px 17px;min-height:135px}.idea b{display:block;font-size:15px;color:var(--ink);margin:8px 0}.idea span{display:block;color:var(--sub);line-height:1.5;font-size:13px}.glyph{font-size:24px}
.foot{border-top:1px solid var(--line);padding-top:22px;margin-top:28px;display:flex;align-items:center;justify-content:space-between;gap:22px;flex-wrap:wrap}.foot small{line-height:1.6;color:var(--sub);max-width:460px}.action{border:1px solid #bdd1ff;padding:12px 17px;border-radius:12px;color:var(--blue);text-decoration:none;font-size:14px;font-weight:700;background:#eef4ff}.action:hover,.action:focus-visible{outline:2px solid var(--blue);outline-offset:3px}
@media(max-width:620px){.grid{grid-template-columns:1fr}.idea{min-height:0}.badge{display:none}.foot{align-items:stretch}.action{text-align:center}}@media(prefers-reduced-motion:reduce){*{animation:none!important;scroll-behavior:auto!important}}
</style></head><body><main>
<div class="brand"><div><span class="brandMark" aria-hidden="true">◷</span>FOCOLOCAL</div><span class="badge">Un descanso digital</span></div>
<div class="art" aria-hidden="true">☀</div>
<h1>Un poco de espacio para ti.</h1>
<p>Has llegado al límite que elegiste para <strong class="domain">{{DOMAIN}}</strong>. FocoLocal ha detenido el contenido de esta pestaña, incluido el vídeo y el audio, sin afectar a tus otras aplicaciones.</p>
<p><strong>{{MESSAGE}}</strong></p>
<h2>¿Qué puedes hacer ahora?</h2>
<div class="grid" role="list">
<div class="idea" role="listitem"><div class="glyph" aria-hidden="true">☕</div><b>Haz una pausa</b><span>Camina, toma agua o descansa la vista dos minutos.</span></div>
<div class="idea" role="listitem"><div class="glyph" aria-hidden="true">✦</div><b>Continúa algo importante</b><span>Retoma un proyecto, estudio o una tarea pendiente.</span></div>
<div class="idea" role="listitem"><div class="glyph" aria-hidden="true">◎</div><b>Elige otra actividad</b><span>Puedes seguir usando otros sitios que no tengan su límite agotado.</span></div>
</div>
<div class="foot"><small>Para ajustar tu límite, abre FocoLocal desde el icono de la bandeja de Windows. No bloqueamos Internet ni los procesos automáticos.</small><a class="action" href="about:blank" target="_blank" rel="noopener noreferrer">Abrir pestaña en blanco ↗</a></div>
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
	key := fmt.Sprintf("%d:%d:%s", s.Window, s.PID, s.Host)
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
