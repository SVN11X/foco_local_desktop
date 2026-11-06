package windows

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"focolocal-desktop/internal/limit"
)

type Sample struct {
	Window                uintptr
	PID                   uint32
	Browser, Host, Reason string
	Working               bool
	At                    time.Time
}
type App struct {
	store                                                                                                             *limit.Store
	main, overlay, monitor                                                                                            uintptr
	fieldDomain, fieldMinutes, list, info, active, btnPause, chkAuto, chkSafe, overlayTitle, overlayNote, monitorText uintptr
	font, bigFont                                                                                                     uintptr
	done                                                                                                              chan struct{}
	scanDelay                                                                                                         <-chan time.Time
	ticker                                                                                                            *time.Ticker
	mu                                                                                                                sync.Mutex
	sample                                                                                                            Sample
	lastBrowserPID                                                                                                    uint32
	lastBrowserName                                                                                                   string
	lastBrowserHost                                                                                                   string
	lastBrowserAt                                                                                                     time.Time
	suppressUntil                                                                                                     time.Time
	blocked                                                                                                           string
	trayReady                                                                                                         bool
	quit                                                                                                              bool
	mutex                                                                                                             uintptr
	selectedDomain                                                                                                    string
	monitorEnabled                                                                                                    bool
}

var current *App

const (
	idAdd     = 1001
	idRemove  = 1002
	idPause   = 1003
	idAuto    = 1004
	idSafe    = 1005
	idExit    = 1006
	idReset   = 1007
	idExclude = 1008
	idMonitor = 1009
	idNewTab  = 2001
	trayClick = wmApp + 12
	updated   = wmApp + 10
)
const title = "FocoLocal Desktop | Control de sitios web"

func (a *App) now() time.Time { return time.Now() }
func (a *App) publish(s Sample) {
	s.At = time.Now()
	a.mu.Lock()
	a.sample = s
	if s.PID != 0 {
		a.lastBrowserPID = s.PID
		a.lastBrowserName = s.Browser
		if s.Host != "" {
			a.lastBrowserHost = s.Host
			a.lastBrowserAt = time.Now()
		}
	}
	a.mu.Unlock()
	post(a.main, updated, 0, 0)
}
func (a *App) sampleCopy() Sample { a.mu.Lock(); defer a.mu.Unlock(); return a.sample }
func mainWnd(hwnd uintptr, msg uint32, w, l uintptr) uintptr {
	if current == nil {
		return call(pDefWindowProc, hwnd, uintptr(msg), w, l)
	}
	a := current
	switch msg {
	case wmCreate:
		return 0
	case wmClose:
		if a.trayReady {
			show(hwnd, swHide)
		} else {
			a.shutdown()
			call(pPostQuit, 0)
		}
		return 0
	case wmDestroy:
		a.shutdown()
		call(pPostQuit, 0)
		return 0
	case wmTimer:
		if w == 1 {
			a.tick()
		}
		return 0
	case updated:
		a.updateActive()
		a.updateMonitor()
		return 0
	case wmApp + 13:
		a.showHub()
		return 0
	case trayClick:
		if l == 0x202 || l == 0x205 || l == 0x204 {
			a.showHub()
		}
		return 0
	case wmCommand:
		id := int(w & 0xffff)
		switch id {
		case idAdd:
			a.addRule()
		case idRemove:
			a.removeRule()
		case idPause:
			a.togglePause()
		case idAuto:
			a.toggleAuto()
		case idSafe:
			a.toggleSafe()
		case idReset:
			if confirm(a.main, "¿Reiniciar únicamente los contadores de hoy?", "Confirmar reinicio") {
				a.store.ResetTimer(time.Now())
				a.save()
				a.refreshRules()
			}
		case idMonitor:
			a.startMonitor()
		case idExclude:
			a.mu.Lock()
			pid := a.lastBrowserPID
			a.mu.Unlock()
			if pid != 0 {
				a.store.ExcludePID(pid)
				a.refreshRules()
				alert(a.main, fmt.Sprintf("El PID %d está excluido hasta cerrar FocoLocal.", pid))
			} else {
				alert(a.main, "Abre primero el navegador que deseas excluir.")
			}
		case idExit:
			call(pDestroyWindow, a.main)
		}
		return 0
	}
	return call(pDefWindowProc, hwnd, uintptr(msg), w, l)
}

var pDestroyWindow = user32.NewProc("DestroyWindow")
var pCreateMutex = kernel32.NewProc("CreateMutexW")
var pFindWindow = user32.NewProc("FindWindowW")
var pSetForegroundWindow = user32.NewProc("SetForegroundWindow")
var pMessageBox = user32.NewProc("MessageBoxW")

func alert(parent uintptr, s string) { call(pMessageBox, parent, su(s), su("FocoLocal"), 0x40) }
func confirm(parent uintptr, s, caption string) bool {
	return call(pMessageBox, parent, su(s), su(caption), 0x24) == 6
}
func overlayWnd(hwnd uintptr, msg uint32, w, l uintptr) uintptr {
	if current == nil {
		return call(pDefWindowProc, hwnd, uintptr(msg), w, l)
	}
	if msg == wmCommand && int(w&0xffff) == idNewTab {
		current.newTab()
		return 0
	}
	return call(pDefWindowProc, hwnd, uintptr(msg), w, l)
}

func Run() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// A second launch should not create another timer or double-account time.
	h, _, mutexErr := pCreateMutex.Call(0, 0, su("Local\\FocoLocalDesktop.Singleton.v2"))
	if mutexErr == syscall.Errno(183) {
		existing := call(pFindWindow, su("FocoLocal.Main.v2"), 0)
		if existing != 0 {
			post(existing, wmApp+13, 0, 0)
		}
		if h != 0 {
			call(pCloseHandle, h)
		}
		return nil
	}
	call(pSetProcessDPIAware)
	home, e := os.UserConfigDir()
	if e != nil {
		return e
	}
	path := filepath.Join(home, "FocoLocalDesktop", "config.json")
	store, e := limit.Open(path)
	if e != nil {
		return e
	}
	a := &App{store: store, done: make(chan struct{}), mutex: h}
	current = a
	a.initUI()
	if a.main == 0 {
		if h != 0 {
			call(pCloseHandle, h)
		}
		return fmt.Errorf("No se pudo crear la ventana principal")
	}
	a.ticker = time.NewTicker(1300 * time.Millisecond)
	a.scanDelay = a.ticker.C
	go runScanner(a)
	call(pSetTimer, a.main, 1, 1000, 0)
	a.refreshRules()
	a.updateActive()
	if len(os.Args) > 1 && os.Args[1] == "--minimized" && a.trayReady {
		show(a.main, swHide)
	} else {
		show(a.main, swShow)
		call(pUpdateWindow, a.main)
	}
	var m msg
	for call(pGetMessage, uintptr(unsafe.Pointer(&m)), 0, 0, 0) > 0 {
		call(pTranslateMessage, uintptr(unsafe.Pointer(&m)))
		call(pDispatchMessage, uintptr(unsafe.Pointer(&m)))
	}
	a.shutdown()
	return nil
}
func (a *App) shutdown() {
	if a.quit {
		return
	}
	a.quit = true
	close(a.done)
	if a.ticker != nil {
		a.ticker.Stop()
	}
	a.store.Save()
	a.tray(2)
	if a.mutex != 0 {
		call(pCloseHandle, a.mutex)
	}
	if a.font != 0 {
		call(pDeleteObject, a.font)
	}
	if a.bigFont != 0 {
		call(pDeleteObject, a.bigFont)
	}
}
func (a *App) initUI() {
	bg := call(pGetSysColorBrush, 5)
	classRegister("FocoLocal.Main.v2", syscall.NewCallback(mainWnd), bg)
	classRegister("FocoLocal.Overlay.v2", syscall.NewCallback(overlayWnd), bg)
	classRegister("FocoLocal.Monitor.v2", syscall.NewCallback(overlayWnd), bg)
	screenW := int32(call(user32.NewProc("GetSystemMetrics"), 0))
	screenH := int32(call(user32.NewProc("GetSystemMetrics"), 1))
	x := (screenW - 910) / 2
	y := (screenH - 720) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	a.main = create("FocoLocal.Main.v2", title, 0, wsCaption|wsSysmenu|wsMinimize, x, y, 910, 720, 0, 0)
	if a.main == 0 {
		return
	}
	a.font = makeFont(-18, 400)
	a.bigFont = makeFont(-28, 700)
	a.label("FOCOLOCAL", 24, 18, 350, 38, true)
	a.label("Control unificado de navegación, sin extensiones", 24, 60, 790, 24, false)
	a.label("SITIOS CON LÍMITE DIARIO", 24, 107, 630, 28, true)
	a.label("Dominio (ej.: youtube.com)", 24, 147, 330, 24, false)
	a.label("Minutos al día", 420, 147, 180, 24, false)
	a.fieldDomain = a.control("EDIT", "", wsBorder|esAutoHScroll|wsTabstop, 24, 177, 370, 36, 0)
	a.fieldMinutes = a.control("EDIT", "30", wsBorder|esAutoHScroll|wsTabstop, 420, 177, 115, 36, 0)
	a.control("BUTTON", "+ Guardar límite", wsTabstop, 560, 177, 195, 37, idAdd)
	a.list = a.control("LISTBOX", "", wsBorder|wsVScroll|lbsNotify|wsTabstop, 24, 236, 842, 189, 0)
	a.control("BUTTON", "Eliminar seleccionado", 0, 24, 436, 205, 34, idRemove)
	a.control("BUTTON", "Reiniciar contadores de hoy", 0, 249, 436, 246, 34, idReset)
	a.btnPause = a.control("BUTTON", "Pausar control", 0, 689, 436, 177, 34, idPause)
	a.label("ESTADO EN TIEMPO REAL", 24, 493, 400, 28, true)
	a.control("BUTTON", "Proteger último PID detectado", 0, 601, 492, 265, 36, idExclude)
	a.active = a.label("Detección inicializando...", 24, 532, 842, 43, false)
	a.chkSafe = a.control("BUTTON", "Excluir navegadores iniciados por scripts (Python/Node/WebDriver)", bsAutoCheck, 24, 584, 680, 30, idSafe)
	a.chkAuto = a.control("BUTTON", "Iniciar con Windows", bsAutoCheck, 24, 617, 250, 30, idAuto)
	a.control("BUTTON", "Salir por completo", 0, 682, 614, 184, 36, idExit)
	a.control("BUTTON", "Probar con monitor flotante", 0, 330, 614, 330, 36, idMonitor)
	a.info = a.label("Solo cuenta el navegador en primer plano, no vídeos reproduciéndose detrás del hub.", 24, 653, 830, 22, false)
	a.overlay = create("FocoLocal.Overlay.v2", "Límite diario alcanzado", exToolWindow|exTopMost|exNoActivate, wsPopup|wsVisible, 0, 0, 500, 400, 0, 0)
	show(a.overlay, swHide)
	a.overlayTitle = a.overlayLabel("TIEMPO AGOTADO", 220, 145, 760, 58, true)
	a.overlayNote = a.overlayLabel("Has alcanzado el límite diario configurado.", 220, 225, 760, 70, false)
	a.overlayControl("BUTTON", "Abrir una pestaña nueva", 0, 220, 338, 340, 46, idNewTab)
	// Non-activating status window. It lets the user verify counting
	// while the browser stays foreground, without installing a browser extension.
	mx := screenW - 515
	if mx < 0 {
		mx = 0
	}
	a.monitor = create("FocoLocal.Monitor.v2", "Monitor FocoLocal", exToolWindow|exTopMost|exNoActivate, wsPopup, mx, 40, 490, 94, 0, 0)
	if a.monitor != 0 {
		a.monitorText = create("STATIC", "Monitor FocoLocal: esperando navegador...", 0, wsVisible|wsChild|ssLeft, 12, 8, 465, 80, a.monitor, 0)
		setFont(a.monitorText, a.font)
		show(a.monitor, swHide)
	}
	a.tray(0)
}
func (a *App) control(class, text string, style uint32, x, y, w, h int32, id int) uintptr {
	hwnd := create(class, text, 0, wsVisible|wsChild|style, x, y, w, h, a.main, uintptr(id))
	setFont(hwnd, a.font)
	return hwnd
}
func (a *App) label(s string, x, y, w, h int32, big bool) uintptr {
	hwnd := a.control("STATIC", s, ssLeft, x, y, w, h, 0)
	if big {
		setFont(hwnd, a.bigFont)
	}
	return hwnd
}
func (a *App) overlayControl(class, text string, style uint32, x, y, w, h int32, id int) uintptr {
	hwnd := create(class, text, 0, wsVisible|wsChild|style, x, y, w, h, a.overlay, uintptr(id))
	setFont(hwnd, a.font)
	return hwnd
}
func (a *App) overlayLabel(s string, x, y, w, h int32, big bool) uintptr {
	hwnd := a.overlayControl("STATIC", s, ssLeft, x, y, w, h, 0)
	if big {
		setFont(hwnd, a.bigFont)
	}
	return hwnd
}
func (a *App) addRule() {
	domain := getText(a.fieldDomain)
	minutes, e := strconv.Atoi(strings.TrimSpace(getText(a.fieldMinutes)))
	if e != nil {
		alert(a.main, "Ingresa un número de minutos válido.")
		return
	}
	if e = a.store.Add(domain, minutes); e != nil {
		alert(a.main, e.Error())
		return
	}
	text(a.fieldDomain, "")
	a.save()
	a.refreshRules()
}
func (a *App) selected() string {
	idx := int(send(a.list, lbGetCurSel, 0, 0))
	cfg, _ := a.store.Snapshot(time.Now())
	if idx < 0 || idx >= len(cfg.Rules) {
		return ""
	}
	return cfg.Rules[idx].Domain
}
func (a *App) removeRule() {
	domain := a.selected()
	if domain == "" {
		alert(a.main, "Selecciona primero un sitio de la lista.")
		return
	}
	a.store.Remove(domain)
	a.save()
	a.refreshRules()
}
func (a *App) togglePause() {
	cfg, _ := a.store.Snapshot(time.Now())
	a.store.SetPause(!cfg.Paused)
	a.save()
	a.refreshRules()
}
func (a *App) toggleAuto() {
	checked := send(a.chkAuto, bmGetCheck, 0, 0) != 0
	if err := a.setAutoStart(checked); err != nil {
		alert(a.main, "No se pudo configurar el inicio automático: "+err.Error())
		send(a.chkAuto, bmSetCheck, 0, 0)
		return
	}
	a.store.SetAutoStart(checked)
	a.save()
}
func (a *App) toggleSafe() {
	a.store.SetSafeAutomation(send(a.chkSafe, bmGetCheck, 0, 0) != 0)
	a.save()
}
func (a *App) save() {
	if e := a.store.Save(); e != nil {
		alert(a.main, "No se pudo guardar: "+e.Error())
	}
}
func (a *App) refreshRules() {
	cfg, used := a.store.Snapshot(time.Now())
	prev := a.selected()
	send(a.list, lbResetContent, 0, 0)
	idxToSelect := -1
	for i, r := range cfg.Rules {
		total := int64(r.Minutes) * 60000
		spent := used[r.Domain]
		remain := (total - spent + 59999) / 60000
		if remain < 0 {
			remain = 0
		}
		state := "ACTIVO"
		if !r.Enabled {
			state = "PAUSADO"
		}
		line := fmt.Sprintf("%-35s  %4d min/día  |  usados: %02d:%02d  |  quedan: %d min  | %s", r.Domain, r.Minutes, spent/60000, (spent/1000)%60, remain, state)
		send(a.list, lbAddString, 0, su(line))
		if r.Domain == prev {
			idxToSelect = i
		}
	}
	if idxToSelect >= 0 {
		send(a.list, 0x0186, uintptr(idxToSelect), 0)
	} // LB_SETCURSEL
	if cfg.Paused {
		text(a.btnPause, "Reanudar control")
	} else {
		text(a.btnPause, "Pausar control")
	}
	if cfg.AutoStart {
		send(a.chkAuto, bmSetCheck, 1, 0)
	} else {
		send(a.chkAuto, bmSetCheck, 0, 0)
	}
	if cfg.SafeAutomation {
		send(a.chkSafe, bmSetCheck, 1, 0)
	} else {
		send(a.chkSafe, bmSetCheck, 0, 0)
	}
}
func (a *App) updateActive() {
	s := a.sampleCopy()
	if s.Host != "" {
		text(a.active, fmt.Sprintf("Detectado: %s | %s | PID %d", s.Browser, s.Host, s.PID))
		return
	}
	if s.PID != 0 {
		text(a.active, fmt.Sprintf("%s | PID %d | %s", s.Browser, s.PID, s.Reason))
		return
	}
	if strings.HasPrefix(s.Reason, "Hub en primer plano") {
		a.mu.Lock()
		name, host, pid, at := a.lastBrowserName, a.lastBrowserHost, a.lastBrowserPID, a.lastBrowserAt
		a.mu.Unlock()
		if host != "" && time.Since(at) < time.Hour {
			text(a.active, fmt.Sprintf("Hub en primer plano: contador detenido. Usa el monitor flotante.\r\nÚltima detección: %s | %s | PID %d", name, host, pid))
		} else {
			text(a.active, "Hub en primer plano: contador detenido. Pulsa 'Probar con monitor flotante'.")
		}
		return
	}
	text(a.active, s.Reason)
}

func (a *App) showHub() {
	a.monitorEnabled = false
	if a.monitor != 0 {
		show(a.monitor, swHide)
	}
	show(a.main, swRestore)
	call(pSetForegroundWindow, a.main)
}

// The floating monitor is deliberately non-activating and read-only.
// Hiding the hub returns focus to the browser; the monitor itself does not count.
func (a *App) startMonitor() {
	if a.monitor == 0 {
		alert(a.main, "No se pudo crear el monitor flotante")
		return
	}
	a.monitorEnabled = true
	a.updateMonitor()
	show(a.monitor, swShowNA)
	if a.trayReady {
		show(a.main, swHide)
	} else {
		show(a.main, swMinimized)
	}
}

func formatDuration(ms int64) string {
	if ms < 0 {
		ms = 0
	}
	totalSeconds := ms / 1000
	return fmt.Sprintf("%02d:%02d", totalSeconds/60, totalSeconds%60)
}

func (a *App) updateMonitor() {
	if !a.monitorEnabled || a.monitorText == 0 {
		return
	}
	s := a.sampleCopy()
	if s.Host == "" || s.Window == 0 || fg() != s.Window || time.Since(s.At) > 3300*time.Millisecond {
		reason := s.Reason
		if reason == "" {
			reason = "No hay navegador activo"
		}
		text(a.monitorText, "FOCOLOCAL | MONITOR EN VIVO\r\n"+reason+"\r\nAbre el hub desde el icono de bandeja")
		return
	}
	cfg, used := a.store.Snapshot(time.Now())
	var domain string
	var max int64
	for _, r := range cfg.Rules {
		if r.Enabled && limit.Matches(s.Host, r.Domain) && len(r.Domain) > len(domain) {
			domain = r.Domain
			max = int64(r.Minutes) * 60 * 1000
		}
	}
	if cfg.Paused {
		text(a.monitorText, "FOCOLOCAL | CONTROL PAUSADO\r\n"+s.Browser+" | "+s.Host+"\r\nReanuda desde el panel principal")
		return
	}
	if domain == "" {
		text(a.monitorText, "FOCOLOCAL | SIN REGLA PARA ESTE SITIO\r\n"+s.Browser+" | "+s.Host+"\r\nNo se descuenta tiempo")
		return
	}
	spent := used[domain]
	remaining := max - spent
	text(a.monitorText, fmt.Sprintf("FOCOLOCAL | %s | %s\r\nUsados: %s de %s | Restan: %s\r\nEl contador solo avanza con el navegador activo", s.Browser, s.Host, formatDuration(spent), formatDuration(max), formatDuration(remaining)))
}

func (a *App) tick() {
	s := a.sampleCopy()
	now := time.Now()
	active := fg()
	host := s.Host
	// Never charge old pages after focus moved away or the scanner hasn't updated in time.
	if active != s.Window || time.Since(s.At) > 3300*time.Millisecond {
		host = ""
	}
	if s.Window == 0 {
		host = ""
	}
	id := ""
	if host != "" {
		id = fmt.Sprintf("%d:%d:%s", s.Window, s.PID, host)
	}
	blocked, _, _ := a.store.Observe(host, id, now)
	if blocked != "" && host != "" && active == s.Window && now.After(a.suppressUntil) {
		a.blocked = blocked
		a.showOverlay(s, blocked)
	} else {
		a.blocked = ""
		show(a.overlay, swHide)
	}
	a.updateMonitor()
	if now.Unix()%4 == 0 {
		a.refreshRules()
	}
	if now.Unix()%12 == 0 {
		_ = a.store.Save()
	}
}
func (a *App) showOverlay(s Sample, rule string) {
	if s.Window == 0 {
		return
	}
	r := position(s.Window)
	w := r.Right - r.Left
	h := r.Bottom - r.Top
	if w < 320 || h < 240 {
		return
	}
	// Covers only this interactive window. Does not alter network traffic or other apps.
	label := fmt.Sprintf("Límite diario de %s agotado", rule)
	text(a.overlayTitle, "TIEMPO AGOTADO")
	text(a.overlayNote, label+"\r\nEl contador se reinicia al comenzar el siguiente día.")
	// Center controls relative to window size; keeps other monitors / moving windows in sync.
	x := (w - 760) / 2
	if x < 25 {
		x = 25
	}
	y := (h - 300) / 2
	if y < 25 {
		y = 25
	}
	child := func(hwnd uintptr, xx, yy, ww, hh int32) {
		call(pSetWindowPos, hwnd, 0, uintptr(xx), uintptr(yy), uintptr(ww), uintptr(hh), swpNoZOrder|swpNoActivate)
	}
	child(a.overlayTitle, x, y, 760, 58)
	child(a.overlayNote, x, y+88, 760, 88)
	// locate button by id (GetDlgItem)
	btn := call(user32.NewProc("GetDlgItem"), a.overlay, idNewTab)
	child(btn, x, y+205, 340, 48)
	call(pSetWindowPos, a.overlay, hwndTopMost, uintptr(r.Left), uintptr(r.Top), uintptr(w), uintptr(h), swpNoActivate|swpShowWindow)
}
func (a *App) newTab() {
	target := a.sampleCopy().Window
	a.suppressUntil = time.Now().Add(3 * time.Second)
	show(a.overlay, swHide)
	if target == 0 {
		return
	}
	call(pSetForegroundWindow, target)
	// Keyboard shortcut Ctrl+T: does not close tabs or modify running processes.
	call(pKeybdEvent, 0x11, 0, 0, 0)
	call(pKeybdEvent, 0x54, 0, 0, 0)
	call(pKeybdEvent, 0x54, 0, 2, 0)
	call(pKeybdEvent, 0x11, 0, 2, 0)
}

// Native Win32 notification area icon: the app remains running when the hub is closed.
type notification struct {
	Size                uint32
	Hwnd                uintptr
	ID, Flags, Callback uint32
	Icon                uintptr
	Tip                 [128]uint16
	State, StateMask    uint32
	Info                [256]uint16
	Timeout             uint32
	InfoTitle           [64]uint16
	InfoFlags           uint32
	Guid                guid
	BalloonIcon         uintptr
}

func (a *App) tray(action uintptr) {
	var nid notification
	nid.Size = uint32(unsafe.Sizeof(nid))
	nid.Hwnd = a.main
	nid.ID = 1
	nid.Flags = 1 | 2 | 4
	nid.Callback = trayClick
	nid.Icon = call(pLoadIcon, 0, 32512)
	copy(nid.Tip[:], syscall.StringToUTF16("FocoLocal - doble clic o clic para abrir"))
	if call(pShellNotify, action, uintptr(unsafe.Pointer(&nid))) != 0 && action == 0 {
		a.trayReady = true
	}
}
func (a *App) setAutoStart(enabled bool) error {
	const hkcu = 0x80000001
	const keyWrite = 0x20006
	regPath := "Software\\Microsoft\\Windows\\CurrentVersion\\Run"
	var h uintptr
	var disposition uint32
	result := call(pRegCreateKey, hkcu, su(regPath), 0, 0, 0, keyWrite, 0, uintptr(unsafe.Pointer(&h)), uintptr(unsafe.Pointer(&disposition)))
	if result != 0 {
		return fmt.Errorf("código de Registro %d", result)
	}
	defer call(pRegCloseKey, h)
	name := su("FocoLocalDesktop")
	if !enabled {
		result = call(pRegDeleteValue, h, name)
		if result != 0 && result != 2 {
			return fmt.Errorf("código %d", result)
		}
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	data := syscall.StringToUTF16("\"" + exe + "\" --minimized")
	result = call(pRegSetValue, h, name, 0, 1, uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)*2))
	if result != 0 {
		return fmt.Errorf("código %d", result)
	}
	return nil
}
