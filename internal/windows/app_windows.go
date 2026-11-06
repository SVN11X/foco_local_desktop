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

	"focolocal-desktop/internal/catalog"
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
	store                                                                                                                      *limit.Store
	main, overlay, monitor                                                                                                     uintptr
	fieldDomain, fieldMinutes, list, info, active, btnPause, chkAuto, chkSafe, chkStop, overlayTitle, overlayNote, monitorText uintptr
	statRules, statUsed, statMode                                                                                              uintptr
	font, bigFont, smallFont, sectionFont, heroFont                                                                            uintptr
	brushes                                                                                                                    map[uint32]uintptr
	labelStyles                                                                                                                map[uintptr]labelStyle
	buttonStyles                                                                                                               map[uintptr]buttonStyle
	done                                                                                                                       chan struct{}
	scanDelay                                                                                                                  <-chan time.Time
	ticker                                                                                                                     *time.Ticker
	mu                                                                                                                         sync.Mutex
	sample                                                                                                                     Sample
	lastBrowserPID                                                                                                             uint32
	lastBrowserName, lastBrowserHost                                                                                           string
	lastBrowserAt, suppressUntil                                                                                               time.Time
	blocked                                                                                                                    string
	trayReady, quit                                                                                                            bool
	mutex                                                                                                                      uintptr
	selectedDomain                                                                                                             string
	monitorEnabled                                                                                                             bool
	lastRedirectKey                                                                                                            string
	lastRedirectAt                                                                                                             time.Time
	lastSeenBrowser                                                                                                            string
	seenConsecutive                                                                                                            int
}

var current *App

const (
	idAdd         = 1001
	idRemove      = 1002
	idPause       = 1003
	idAuto        = 1004
	idSafe        = 1005
	idExit        = 1006
	idReset       = 1007
	idExclude     = 1008
	idMonitor     = 1009
	idStop        = 1010
	idGitHub      = 1011
	idPresetStart = 1200
	idNewTab      = 2001
	trayClick     = wmApp + 12
	updated       = wmApp + 10
)
const title = "FocoLocal Desktop  ·  Tu tiempo, tus reglas"
const repositoryURL = "https://github.com/SVN11X/foco_local_desktop"

func (a *App) now() time.Time { return time.Now() }
func (a *App) publish(s Sample) {
	s.At = time.Now()
	a.mu.Lock()
	a.sample = s
	if s.Host != "" && s.Window != 0 {
		key := fmt.Sprintf("%d:%d:%s", s.Window, s.PID, s.Host)
		if a.lastSeenBrowser == key {
			a.seenConsecutive++
		} else {
			a.lastSeenBrowser, a.seenConsecutive = key, 1
		}
	} else {
		a.lastSeenBrowser, a.seenConsecutive = "", 0
	}
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
	case wmEraseBackground:
		return 1
	case wmPaint:
		a.renderBackground(hwnd)
		return 0
	case wmCtlColorStatic, wmCtlColorBtn:
		return a.ctlColor(w, l)
	case wmDrawItem:
		return a.paintButton(l)
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
		case idStop:
			a.toggleStop()
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
		case idGitHub:
			a.openRepository()
		case idExit:
			call(pDestroyWindow, a.main)
		}
		if id >= idPresetStart && id < idPresetStart+len(catalog.Sites) {
			a.addPreset(catalog.Sites[id-idPresetStart])
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
	for _, font := range []uintptr{a.bigFont, a.smallFont, a.sectionFont, a.heroFont} {
		if font != 0 {
			call(pDeleteObject, font)
		}
	}
	for _, brush := range a.brushes {
		if brush != 0 {
			call(pDeleteObject, brush)
		}
	}
}
func (a *App) initUI() {
	bg := call(pGetSysColorBrush, 5)
	classRegister("FocoLocal.Main.v2", syscall.NewCallback(mainWnd), bg)
	classRegister("FocoLocal.Overlay.v2", syscall.NewCallback(overlayWnd), bg)
	classRegister("FocoLocal.Monitor.v2", syscall.NewCallback(overlayWnd), bg)
	screenW := int32(call(user32.NewProc("GetSystemMetrics"), 0))
	screenH := int32(call(user32.NewProc("GetSystemMetrics"), 1))
	x, y := (screenW-1100)/2, (screenH-748)/2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	a.main = create("FocoLocal.Main.v2", title, 0, wsCaption|wsSysmenu|wsMinimize, x, y, 1100, 748, 0, 0)
	if a.main == 0 {
		return
	}
	a.font = makeFont(-17, 400)
	a.smallFont = makeFont(-14, 400)
	a.bigFont = makeFont(-28, 700)
	a.sectionFont = makeFont(-22, 700)
	a.heroFont = makeFont(-31, 700)

	a.styledLabel("◉  FOCOLOCAL", 25, 28, 190, 36, a.sectionFont, colorWhite, colorSidebar)
	a.styledLabel("Tu tiempo, tus reglas.", 28, 70, 182, 21, a.smallFont, colorSubtle, colorSidebar)
	a.styledLabel("ENFOQUE DIARIO", 34, 136, 165, 20, a.smallFont, colorSubtle, colorSidebarSoft)
	a.styledLabel("Menos distracciones.", 34, 165, 165, 24, a.font, colorWhite, colorSidebarSoft)
	a.styledLabel("Más tiempo para ti.", 34, 190, 165, 24, a.font, colorWhite, colorSidebarSoft)
	a.styledLabel("AJUSTES", 27, 235, 170, 20, a.smallFont, colorSubtle, colorSidebar)
	a.chkSafe = a.control("BUTTON", "Proteger scripts", bsAutoCheck|wsTabstop, 25, 264, 190, 30, idSafe)
	a.labelColors(a.chkSafe, colorWhite, colorSidebar)
	a.styledLabel("Python, Node y WebDriver", 47, 296, 160, 19, a.smallFont, colorSubtle, colorSidebar)
	a.chkStop = a.control("BUTTON", "Detener vídeo y audio", bsAutoCheck|wsTabstop, 25, 329, 190, 30, idStop)
	a.labelColors(a.chkStop, colorWhite, colorSidebar)
	a.styledLabel("Sustituye la pestaña", 47, 361, 160, 19, a.smallFont, colorSubtle, colorSidebar)
	a.chkAuto = a.control("BUTTON", "Iniciar con Windows", bsAutoCheck|wsTabstop, 25, 393, 193, 30, idAuto)
	a.labelColors(a.chkAuto, colorWhite, colorSidebar)
	a.btnPause = a.styledButton("Ⅱ  Pausar control", 25, 454, 184, 40, idPause, colorSidebarSoft, colorSidebar, colorWhite, colorSidebarSoft)
	a.styledButton("↗  Ver repositorio", 25, 525, 184, 40, idGitHub, colorSidebarSoft, colorSidebar, colorWhite, colorSidebarSoft)
	a.styledButton("Salir de la app", 25, 581, 184, 38, idExit, colorSidebar, colorSidebarSoft, colorSubtle, colorSidebarSoft)
	a.styledLabel("v0.2.3  ·  LOCAL", 28, 665, 188, 18, a.smallFont, colorSubtle, colorSidebar)

	a.styledLabel("Tu espacio de enfoque", 252, 24, 522, 38, a.heroFont, colorText, colorCanvas)
	a.styledLabel("Controla tus sitios sin extensiones ni bloqueos de red.", 253, 68, 580, 24, a.font, colorMuted, colorCanvas)
	a.styledButton("↗  GitHub", 872, 32, 172, 40, idGitHub, colorWhite, colorPaleBlue, colorBlue, colorBorder)

	a.styledLabel("SITIOS CON LÍMITE", 268, 112, 220, 19, a.smallFont, colorMuted, colorWhite)
	a.statRules = a.styledLabel("0 sitios", 268, 136, 230, 35, a.heroFont, colorText, colorWhite)
	a.styledLabel("TIEMPO USADO HOY", 542, 112, 211, 19, a.smallFont, colorMuted, colorWhite)
	a.statUsed = a.styledLabel("00:00", 542, 136, 224, 35, a.heroFont, colorText, colorWhite)
	a.styledLabel("ESTADO DEL CONTROL", 817, 112, 213, 19, a.smallFont, colorMuted, colorWhite)
	a.statMode = a.styledLabel("● Activo", 817, 138, 225, 32, a.sectionFont, colorGreen, colorWhite)

	a.styledLabel("Añade un sitio", 287, 205, 432, 31, a.sectionFont, colorText, colorWhite)
	a.styledLabel("Dominio web", 275, 241, 330, 20, a.smallFont, colorMuted, colorWhite)
	a.styledLabel("Minutos al día", 648, 241, 130, 20, a.smallFont, colorMuted, colorWhite)
	a.fieldDomain = a.control("EDIT", "", wsBorder|esAutoHScroll|wsTabstop, 275, 267, 352, 34, 0)
	a.fieldMinutes = a.control("EDIT", "30", wsBorder|esAutoHScroll|wsTabstop, 648, 267, 116, 34, 0)
	a.styledButton("＋  Guardar límite", 790, 267, 245, 36, idAdd, colorBlue, colorBlueDark, colorWhite, colorBlue)
	a.styledLabel("SITIOS PREDEFINIDOS · haz clic para añadir", 275, 311, 590, 20, a.smallFont, colorMuted, colorWhite)
	for i, p := range catalog.Sites {
		col, row := i%4, i/4
		px, py := int32(275+col*191), int32(337+row*35)
		a.styledButton("＋  "+p.Name, px, py, 176, 29, idPresetStart+i, colorPaleBlue, colorBorder, colorBlue, colorPaleBlue)
	}

	a.styledLabel("Tus límites diarios", 273, 423, 630, 30, a.sectionFont, colorText, colorWhite)
	a.styledLabel("Tiempo compartido entre navegadores; actualizado cada segundo.", 274, 454, 670, 19, a.smallFont, colorMuted, colorWhite)
	a.list = a.control("LISTBOX", "", wsBorder|wsVScroll|lbsNotify|wsTabstop, 275, 478, 760, 63, 0)
	a.styledButton("Eliminar seleccionado", 275, 548, 217, 33, idRemove, colorWhite, colorPaleBlue, colorError, colorBorder)
	a.styledButton("Reiniciar contadores", 502, 548, 227, 33, idReset, colorWhite, colorPaleBlue, colorText, colorBorder)
	a.styledButton("Proteger último PID", 800, 548, 235, 33, idExclude, colorWhite, colorPaleBlue, colorText, colorBorder)

	a.styledLabel("ACTIVIDAD EN DIRECTO", 274, 609, 470, 21, a.smallFont, colorMuted, colorWhite)
	a.active = a.styledLabel("Esperando navegador activo...", 274, 637, 537, 43, a.font, colorText, colorWhite)
	a.styledButton("◷  Monitor flotante", 836, 641, 198, 39, idMonitor, colorPaleGreen, colorBorder, colorGreen, colorPaleGreen)
	a.info = a.styledLabel("Solo cuenta mientras el navegador está en primer plano; no cuenta pestañas ocultas.", 274, 682, 773, 18, a.smallFont, colorMuted, colorWhite)

	a.overlay = create("FocoLocal.Overlay.v2", "Límite diario alcanzado", exToolWindow|exTopMost|exNoActivate, wsPopup|wsVisible, 0, 0, 500, 400, 0, 0)
	show(a.overlay, swHide)
	a.overlayTitle = a.overlayLabel("Tu tiempo terminó por hoy", 220, 145, 760, 58, true)
	a.overlayNote = a.overlayLabel("Puedes continuar con otra actividad.", 220, 225, 760, 100, false)
	a.overlayControl("BUTTON", "Abrir una pestaña nueva", 0, 220, 350, 340, 46, idNewTab)

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
func (a *App) openRepository() {
	if result := call(pShellExecute, a.main, su("open"), su(repositoryURL), 0, 0, 1); result <= 32 {
		alert(a.main, "No se pudo abrir GitHub. Visita: "+repositoryURL)
	}
}
func (a *App) addPreset(p catalog.Preset) {
	cfg, _ := a.store.Snapshot(time.Now())
	for _, r := range cfg.Rules {
		if r.Domain == p.Domain {
			text(a.fieldDomain, p.Domain)
			text(a.fieldMinutes, strconv.Itoa(r.Minutes))
			text(a.info, "Este sitio ya está en tu lista. Modifica los minutos arriba y pulsa Guardar límite.")
			return
		}
	}
	if err := a.store.Add(p.Domain, p.Minutes); err != nil {
		alert(a.main, err.Error())
		return
	}
	a.save()
	a.refreshRules()
	text(a.info, fmt.Sprintf("%s añadido con %d minutos diarios. Puedes cambiar su límite en cualquier momento.", p.Name, p.Minutes))
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
	text(a.info, "Límite guardado. Solo se descuenta tiempo con el navegador activo.")
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
	if !confirm(a.main, "¿Eliminar el límite de "+domain+"?", "Eliminar sitio") {
		return
	}
	a.store.Remove(domain)
	a.save()
	a.refreshRules()
	text(a.info, "Se eliminó el límite de "+domain+".")
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
func (a *App) toggleStop() {
	mode := "overlay"
	if send(a.chkStop, bmGetCheck, 0, 0) != 0 {
		mode = "navigate"
	}
	_ = a.store.SetBlockMode(mode)
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
	send(a.list, wmSetRedraw, 0, 0)
	send(a.list, lbResetContent, 0, 0)
	indexToSelect := -1
	var totalUsed int64
	activeCount := 0
	for i, r := range cfg.Rules {
		spent := used[r.Domain]
		totalUsed += spent
		budget := int64(r.Minutes) * 60000
		remaining := budget - spent
		if remaining < 0 {
			remaining = 0
		}
		state := "● Activo"
		if !r.Enabled {
			state = "Ⅱ Pausado"
		}
		if spent >= budget {
			state = "✓ Agotado"
		}
		if r.Enabled {
			activeCount++
		}
		line := fmt.Sprintf("%-25s  %4d min/día    ·    usado %s    ·    queda %s    ·    %s", r.Domain, r.Minutes, formatDuration(spent), formatDuration(remaining), state)
		send(a.list, lbAddString, 0, su(line))
		if r.Domain == prev {
			indexToSelect = i
		}
	}
	if indexToSelect >= 0 {
		send(a.list, 0x0186, uintptr(indexToSelect), 0)
	}
	send(a.list, wmSetRedraw, 1, 0)
	call(pInvalidateRect, a.list, 0, 0)
	text(a.statRules, fmt.Sprintf("%d sitios", activeCount))
	text(a.statUsed, formatDuration(totalUsed))
	if cfg.Paused {
		text(a.btnPause, "▶  Reanudar control")
		text(a.statMode, "Ⅱ Pausado")
		a.labelColors(a.statMode, colorError, colorWhite)
	} else {
		text(a.btnPause, "Ⅱ  Pausar control")
		text(a.statMode, "● Activo")
		a.labelColors(a.statMode, colorGreen, colorWhite)
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
	if cfg.BlockMode == "navigate" {
		send(a.chkStop, bmSetCheck, 1, 0)
	} else {
		send(a.chkStop, bmSetCheck, 0, 0)
	}
}
func (a *App) updateActive() {
	s := a.sampleCopy()
	if s.Host != "" && s.Window != 0 && fg() == s.Window && time.Since(s.At) <= 3300*time.Millisecond {
		text(a.active, fmt.Sprintf("● Contando: %s · %s\r\nActualización cada segundo", s.Browser, s.Host))
		return
	}
	if s.PID != 0 {
		text(a.active, fmt.Sprintf("En espera: %s · PID %d\r\n%s", s.Browser, s.PID, s.Reason))
		return
	}
	if strings.HasPrefix(s.Reason, "Hub en primer plano") {
		text(a.active, "Hub abierto: contador detenido.\r\nUtiliza el monitor flotante mientras navegas.")
		return
	}
	text(a.active, "Sin navegación activa: no se descuenta tiempo.\r\n"+s.Reason)
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
		cfg, _ := a.store.Snapshot(now)
		if cfg.BlockMode == "navigate" && a.stopBlockedTab(s, blocked, now) {
			show(a.overlay, swHide)
		} else {
			a.showOverlay(s, blocked) // safe fallback if navigation is unavailable
		}
	} else {
		a.blocked = ""
		show(a.overlay, swHide)
	}
	a.updateMonitor()
	// The user-visible time moves at 1 Hz, independently of the UIA scan rate.
	// Store.Observe has already gated accounting on a fresh foreground sample.
	a.refreshRules()
	a.updateActive()
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
	text(a.overlayTitle, "Un momento para ti")
	text(a.overlayNote, label+"\r\nPrueba estirarte, beber agua o avanzar en otro proyecto. Mañana tendrás un nuevo límite.")
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
