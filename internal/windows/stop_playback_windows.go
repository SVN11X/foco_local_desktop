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
func navigateForegroundTab(target, destination uintptr, urlString string) bool {
	if target == 0 || fg() != target || destination == 0 || urlString == "" {
		return false
	}
	if unsafe.Sizeof(winKeyboardInput{}) != 40 { // fails closed if compiled for a different ABI
		return false
	}
	events := make([]winKeyboardInput, 0, 6+len(urlString)*2)
	// Ctrl+L focuses browser address bar; do not touch page content via JS.
	events = append(events, vkInput(vkControl, false), vkInput(vkL, false), vkInput(vkL, true), vkInput(vkControl, true))
	for _, ch := range urlString { // Unicode input allows spaces/non-ASCII in file: URLs.
		if ch > 0xffff { // file: URLs produced by net/url are ASCII escaped
			return false
		}
		events = append(events, unicodeInput(uint16(ch), false), unicodeInput(uint16(ch), true))
	}
	events = append(events, vkInput(vkReturn, false), vkInput(vkReturn, true))
	if fg() != target || windowPID(target) != uint32(destination) {
		return false
	}
	sent := call(pSendInput, uintptr(len(events)), uintptr(unsafe.Pointer(&events[0])), unsafe.Sizeof(events[0]))
	runtime.KeepAlive(events)
	return sent == uintptr(len(events))
}

func blockedPagePath(configPath, domain string) (string, error) {
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
	content := fmt.Sprintf(`<!doctype html>
<html lang="es"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'">
<title>Tiempo agotado - FocoLocal</title>
<style>
:root{color-scheme:dark;font-family:Segoe UI,system-ui,sans-serif}*{box-sizing:border-box}
body{margin:0;min-height:100vh;display:grid;place-items:center;background:radial-gradient(circle at 25%% 18%%,#173058,#0b1221 65%%);color:#f3f6ff;padding:24px}
main{width:min(590px,100%%);border:1px solid #39445b;border-radius:24px;padding:48px;background:#141d30;box-shadow:0 18px 50px #050a16}
small{color:#94baff;letter-spacing:.16em;font-weight:700}h1{font-size:clamp(2rem,6vw,3.5rem);line-height:1.1;margin:18px 0}p{font-size:1.1rem;line-height:1.65;color:#c2ccdd}strong{color:#fff}
hr{border:0;border-top:1px solid #35415c;margin:26px 0}.hint{font-size:.91rem;color:#9ba9c2}
</style></head><body><main role="main"><small>FOCOLOCAL · CONTROL DE TIEMPO</small>
<h1>Tiempo agotado</h1><p>Has alcanzado tu límite diario para <strong>%s</strong>.</p>
<p>Esta pestaña se ha sustituido para detener también el vídeo y el audio. Puedes seguir utilizando otras pestañas, aplicaciones y tus automatizaciones.</p>
<hr><p class="hint">El contador se restablece mañana. Para cambiar la regla o pausar el control, abre FocoLocal desde el área de notificación de Windows. Si tenías cambios sin guardar en esta pestaña, podrían haberse perdido.</p>
</main></body></html>`, html.EscapeString(domain))
	if e := os.WriteFile(path, []byte(content), 0600); e != nil {
		return "", e
	}
	return path, nil
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
	path, err := blockedPagePath(a.store.Path, rule)
	if err != nil {
		return false
	}
	if !navigateForegroundTab(s.Window, uintptr(s.PID), pathToFileURL(path)) {
		return false
	}
	a.lastRedirectKey, a.lastRedirectAt = key, now
	return true
}
