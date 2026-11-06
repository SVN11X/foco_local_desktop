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
	Pinned                bool
	At                    time.Time
}
type App struct {
	scaleX, scaleY                                                                                                                                                                                                           float64
	store                                                                                                                                                                                                                    *limit.Store
	main, overlay, monitor                                                                                                                                                                                                   uintptr
	fieldDomain, fieldMinutes, fieldFrom, fieldTo, fieldSession, fieldBreak, fieldMonths, comboPeriod, comboUnit, comboDays, list, info, active, btnPause, chkAuto, chkSafe, chkStop, overlayTitle, overlayNote, monitorText uintptr
	ruleRows                                                                                                                                                                                                                 []ruleRow
	statRules, statUsed, statMode, statUsedExplain                                                                                                                                                                           uintptr
	themeButtons                                                                                                                                                                                                             [3]uintptr
	themeMode                                                                                                                                                                                                                string
	isDark                                                                                                                                                                                                                   bool
	font, bigFont, smallFont, sectionFont, heroFont                                                                                                                                                                          uintptr
	brushes                                                                                                                                                                                                                  map[uint32]uintptr
	labelStyles                                                                                                                                                                                                              map[uintptr]labelStyle
	buttonStyles                                                                                                                                                                                                             map[uintptr]buttonStyle
	done                                                                                                                                                                                                                     chan struct{}
	scanDelay                                                                                                                                                                                                                <-chan time.Time
	ticker                                                                                                                                                                                                                   *time.Ticker
	mu                                                                                                                                                                                                                       sync.Mutex
	sample                                                                                                                                                                                                                   Sample
	lastBrowserPID                                                                                                                                                                                                           uint32
	lastBrowserWindow, pinnedWindow                                                                                                                                                                                          uintptr
	pinnedPID                                                                                                                                                                                                                uint32
	pages                                                                                                                                                                                                                    [4][]uintptr
	navButtons                                                                                                                                                                                                               [4]uintptr
	buildPage, page, category                                                                                                                                                                                                int
	categoryButtons                                                                                                                                                                                                          []uintptr
	siteButtons                                                                                                                                                                                                              []uintptr
	lastBrowserName, lastBrowserHost                                                                                                                                                                                         string
	lastBrowserAt, suppressUntil                                                                                                                                                                                             time.Time
	blocked                                                                                                                                                                                                                  string
	trayReady, quit                                                                                                                                                                                                          bool
	mutex                                                                                                                                                                                                                    uintptr
	selectedDomain                                                                                                                                                                                                           string
	monitorEnabled                                                                                                                                                                                                           bool
	lastRedirectKey                                                                                                                                                                                                          string
	lastRedirectAt                                                                                                                                                                                                           time.Time
	lastSeenBrowser                                                                                                                                                                                                          string
	seenConsecutive                                                                                                                                                                                                          int
}

var current *App

const (
	idAdd           = 1001
	idRemove        = 1002
	idPause         = 1003
	idAuto          = 1004
	idSafe          = 1005
	idExit          = 1006
	idReset         = 1007
	idExclude       = 1008
	idMonitor       = 1009
	idStop          = 1010
	idGitHub        = 1011
	idPresetStart   = 1200
	idPageStart     = 1100
	idCategoryStart = 1500
	idPeriod        = 1600
	idPin           = 1601
	idClear         = 1602
	idEdit          = 1603
	idRules         = 1604
	idNewTab        = 2001
	idThemeSystem   = 1710
	idThemeLight    = 1711
	idThemeDark     = 1712
	trayClick       = wmApp + 12
	updated         = wmApp + 10
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
		if s.Host != "" {
			a.lastBrowserWindow = s.Window
		}
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
	case 0x001A, 0x031A: // WM_SETTINGCHANGE, WM_THEMECHANGED
		if a.themeMode == "system" {
			a.applyTheme("system")
		}
		return 0
	case wmPaint:
		a.renderBackground(hwnd)
		return 0
	case wmCtlColorStatic, wmCtlColorBtn, wmCtlColorEdit, wmCtlColorListbox:
		return a.ctlColor(w, l)
	case wmDrawItem:
		if a.paintRuleRow(l) != 0 {
			return 1
		}
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
		case idPin:
			a.togglePinned()
		case idClear:
			a.clearEditor()
		case idEdit:
			a.editSelected()
		case idPeriod:
			a.updateModeInfo()
		case idRules:
			a.showPage(0)
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
		case idThemeSystem, idThemeLight, idThemeDark:
			modes := []string{"system", "light", "dark"}
			mode := modes[id-idThemeSystem]
			if err := a.store.SetTheme(mode); err == nil {
				a.applyTheme(mode)
				a.save()
			}
		case idExit:
			call(pDestroyWindow, a.main)
		}
		if id >= idPageStart && id < idPageStart+4 {
			a.showPage(id - idPageStart)
		}
		if id >= idCategoryStart && id < idCategoryStart+len(catalog.Categories) {
			a.showCategory(id - idCategoryStart)
		}
		if id >= idPresetStart && id < idPresetStart+len(a.siteButtons) {
			picks := catalog.InCategory(a.category)
			if id-idPresetStart < len(picks) {
				a.addPreset(catalog.Sites[picks[id-idPresetStart]])
			}
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
	a := &App{store: store, done: make(chan struct{}), mutex: h, buildPage: -1, scaleX: 1, scaleY: 1}
	current = a
	cfg, _ := store.Snapshot(time.Now())
	a.themeMode = cfg.Theme
	a.initPalette(cfg.Theme)
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
	appW, appH := int32(1190), int32(815)
	if screenW > 100 && screenW-40 < appW {
		appW = screenW - 40
	}
	if screenH > 130 && screenH-72 < appH {
		appH = screenH - 72
	}
	a.scaleX = float64(appW) / 1190
	a.scaleY = float64(appH) / 815
	x, y := (screenW-appW)/2, (screenH-appH)/2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	a.main = create("FocoLocal.Main.v2", title, 0, wsCaption|wsSysmenu|wsMinimize, x, y, appW, appH, 0, 0)
	if a.main == 0 {
		return
	}
	a.setDarkTitleBar(a.isDark)
	a.font = makeFont(-17, 400)
	a.smallFont = makeFont(-14, 400)
	a.bigFont = makeFont(-28, 700)
	a.sectionFont = makeFont(-22, 700)
	a.heroFont = makeFont(-34, 700)

	a.styledLabel("◉  FOCOLOCAL", 24, 29, 193, 36, a.sectionFont, colorWhite, colorSidebar)
	a.styledLabel("MENOS RUIDO. MÁS VIDA.", 29, 71, 190, 21, a.smallFont, colorSubtle, colorSidebar)
	a.styledLabel("TU ESPACIO", 27, 133, 180, 20, a.smallFont, colorSubtle, colorSidebar)
	nav := []string{"▦  Resumen", "◈  Explorar sitios", "⊕  Crear regla", "⚙  Preferencias"}
	for i, label := range nav {
		a.navButtons[i] = a.styledButton(label, 19, int32(165+i*56), 199, 47, idPageStart+i, colorSidebar, colorSidebarSoft, colorWhite, colorSidebar)
	}
	a.styledLabel("CONTROL LOCAL", 29, 432, 184, 19, a.smallFont, colorSubtle, colorSidebar)
	a.styledLabel("Sin extensiones", 29, 461, 183, 23, a.font, colorWhite, colorSidebar)
	a.styledLabel("Sin cortar Internet", 29, 486, 183, 21, a.smallFont, colorSubtle, colorSidebar)
	a.btnPause = a.styledButton("Ⅱ  Pausar control", 19, 567, 199, 44, idPause, colorSidebarSoft, colorSidebar, colorWhite, colorSidebarSoft)
	a.styledButton("↗  Repositorio GitHub", 19, 628, 199, 41, idGitHub, colorSidebarSoft, colorSidebar, colorWhite, colorSidebarSoft)
	a.styledButton("Salir por completo", 19, 680, 199, 41, idExit, colorSidebar, colorSidebarSoft, colorSubtle, colorSidebarSoft)
	a.styledLabel("v0.3.1  •  LOCAL", 29, 740, 186, 19, a.smallFont, colorSubtle, colorSidebar)
	a.styledLabel("FOCOLOCAL DESKTOP", 269, 29, 245, 17, a.smallFont, colorBlue, colorCanvas)
	a.styledLabel("Tu tiempo, mejor utilizado", 268, 53, 622, 45, a.heroFont, colorText, colorCanvas)
	a.styledLabel("Gestiona distracciones sin afectar tus proyectos ni tu privacidad.", 271, 101, 695, 23, a.font, colorMuted, colorCanvas)
	a.styledButton("↗  Abrir GitHub", 980, 52, 159, 39, idGitHub, colorWhite, colorPaleBlue, colorBlue, colorBorder)

	// Resumen
	a.buildPage = 0
	a.styledLabel("SITIOS CONTROLADOS", 282, 166, 225, 20, a.smallFont, colorMuted, colorWhite)
	a.statRules = a.styledLabel("0 sitios", 282, 192, 240, 38, a.heroFont, colorText, colorWhite)
	a.styledLabel("USO REAL REGISTRADO HOY", 572, 166, 265, 20, a.smallFont, colorMuted, colorWhite)
	a.statUsed = a.styledLabel("00:00", 573, 192, 235, 38, a.heroFont, colorText, colorWhite)
	a.statUsedExplain = a.styledLabel("Suma de sitios, sin duplicados", 573, 233, 265, 16, a.smallFont, colorMuted, colorWhite)
	a.styledLabel("ESTADO ACTUAL", 869, 166, 235, 20, a.smallFont, colorMuted, colorWhite)
	a.statMode = a.styledLabel("● Activo", 869, 194, 218, 34, a.sectionFont, colorGreen, colorWhite)
	a.styledLabel("Tus reglas de navegación", 285, 281, 520, 34, a.sectionFont, colorText, colorWhite)
	a.styledLabel("Hoy es distinto de tu presupuesto semanal o mensual. Selecciona una regla para editarla.", 285, 316, 790, 22, a.smallFont, colorMuted, colorWhite)
	a.list = a.control("LISTBOX", "", wsBorder|wsVScroll|lbsNotify|wsTabstop|lbsOwnerDrawFixed|lbsHasStrings, 286, 354, 826, 142, 0)
	send(a.list, lbSetItemHeight, 0, 68)
	a.styledButton("✎  Editar selección", 287, 510, 237, 43, idEdit, colorPaleBlue, colorBorder, colorBlue, colorPaleBlue)
	a.styledButton("−  Eliminar", 539, 510, 183, 43, idRemove, colorWhite, colorPaleBlue, colorError, colorBorder)
	a.styledButton("↻  Reiniciar contadores", 735, 510, 243, 43, idReset, colorWhite, colorPaleBlue, colorText, colorBorder)
	a.styledLabel("Actividad en directo", 286, 606, 400, 29, a.sectionFont, colorText, colorWhite)
	a.active = a.styledLabel("Esperando actividad del navegador...", 286, 645, 540, 51, a.font, colorText, colorWhite)
	a.styledButton("◷  Monitor flotante", 876, 638, 235, 42, idMonitor, colorPaleGreen, colorBorder, colorGreen, colorPaleGreen)
	a.styledButton("⊞  Seguir otra pantalla", 876, 689, 235, 42, idPin, colorPaleBlue, colorBorder, colorBlue, colorPaleBlue)
	a.info = a.styledLabel("Cuenta únicamente uso interactivo o ventanas seguidas por ti.", 283, 719, 577, 32, a.smallFont, colorMuted, colorWhite)

	// Biblioteca por categorías; los botones solo preparan el formulario.
	a.buildPage = 1
	a.styledLabel("Descubre y elige", 288, 161, 500, 34, a.sectionFont, colorText, colorWhite)
	a.styledLabel("Seleccionar un sitio NO lo activa; primero define cómo limitarlo.", 289, 199, 725, 23, a.font, colorMuted, colorWhite)
	for i, cat := range catalog.Categories {
		col, row := i%3, i/3
		x, y := int32(285+col*282), int32(248+row*48)
		a.categoryButtons = append(a.categoryButtons, a.styledButton(cat, x, y, 265, 40, idCategoryStart+i, colorWhite, colorPaleBlue, colorText, colorBorder))
	}
	for i := 0; i < 6; i++ {
		col, row := i%3, i/3
		x, y := int32(285+col*282), int32(387+row*109)
		// six reusable buttons; domain assignment is changed by showCategory.
		btn := a.styledButton("", x, y, 265, 91, idPresetStart+i, colorWhite, colorPaleBlue, colorText, colorBorder)
		a.siteButtons = append(a.siteButtons, btn)
	}
	a.styledLabel("Cómo funciona", 290, 641, 320, 28, a.sectionFont, colorText, colorWhite)
	a.styledLabel("1. Elige un sitio.     2. Ajusta el horario o presupuesto.     3. Guarda tu regla.", 290, 681, 790, 24, a.font, colorText, colorWhite)
	a.styledLabel("Más de 30 opciones locales, sin cargar iconos ni datos desde Internet.", 290, 715, 777, 20, a.smallFont, colorMuted, colorWhite)

	// Editor; soporta todas las modalidades sin extensiones.
	a.buildPage = 2
	a.styledLabel("Diseña tu regla", 288, 160, 500, 34, a.sectionFont, colorText, colorWhite)
	a.styledLabel("Nada se aplicará hasta que pulses Guardar regla.", 290, 198, 780, 22, a.font, colorMuted, colorWhite)
	a.styledLabel("SITIO O DOMINIO", 290, 255, 370, 19, a.smallFont, colorMuted, colorWhite)
	a.fieldDomain = a.control("EDIT", "", wsBorder|esAutoHScroll|wsTabstop, 290, 282, 524, 39, 0)
	a.styledLabel("TIPO DE REGLA", 834, 255, 282, 19, a.smallFont, colorMuted, colorWhite)
	a.comboPeriod = a.control("COMBOBOX", "", cbsDropdownList|wsVScroll|wsTabstop, 834, 282, 282, 240, idPeriod)
	for _, name := range []string{"Diario", "Semanal", "Mensual", "Todo el día (bloquear)", "Por horario", "Por intervalos"} {
		send(a.comboPeriod, cbAddString, 0, su(name))
	}
	send(a.comboPeriod, cbSetCurSel, 0, 0)
	a.styledLabel("PRESUPUESTO DE USO", 290, 358, 375, 19, a.smallFont, colorMuted, colorWhite)
	a.fieldMinutes = a.control("EDIT", "30", wsBorder|esAutoHScroll|wsTabstop, 290, 386, 193, 37, 0)
	a.comboUnit = a.control("COMBOBOX", "", cbsDropdownList|wsVScroll|wsTabstop, 493, 386, 178, 160, 0)
	for _, name := range []string{"Minutos", "Horas"} {
		send(a.comboUnit, cbAddString, 0, su(name))
	}
	send(a.comboUnit, cbSetCurSel, 0, 0)
	a.styledLabel("DÍAS PERMITIDOS", 718, 358, 385, 19, a.smallFont, colorMuted, colorWhite)
	a.comboDays = a.control("COMBOBOX", "", cbsDropdownList|wsVScroll|wsTabstop, 718, 386, 397, 160, 0)
	for _, name := range []string{"Todos los días", "Lunes a viernes", "Sábado y domingo"} {
		send(a.comboDays, cbAddString, 0, su(name))
	}
	send(a.comboDays, cbSetCurSel, 0, 0)
	a.styledLabel("DESDE (HH:MM)", 289, 462, 190, 20, a.smallFont, colorMuted, colorWhite)
	a.fieldFrom = a.control("EDIT", "09:00", wsBorder|esAutoHScroll|wsTabstop, 290, 488, 191, 37, 0)
	a.styledLabel("HASTA (HH:MM)", 497, 462, 190, 20, a.smallFont, colorMuted, colorWhite)
	a.fieldTo = a.control("EDIT", "18:00", wsBorder|esAutoHScroll|wsTabstop, 499, 488, 191, 37, 0)
	a.styledLabel("MESES (VACÍO = TODOS)", 717, 462, 390, 20, a.smallFont, colorMuted, colorWhite)
	a.fieldMonths = a.control("EDIT", "", wsBorder|esAutoHScroll|wsTabstop, 718, 488, 397, 37, 0)
	a.styledLabel("Ej.: 1,2,12 o 4-9", 719, 533, 387, 19, a.smallFont, colorMuted, colorWhite)
	a.styledLabel("USO POR TRAMO (MIN)", 290, 558, 255, 20, a.smallFont, colorMuted, colorWhite)
	a.fieldSession = a.control("EDIT", "25", wsBorder|esAutoHScroll|wsTabstop, 290, 585, 194, 37, 0)
	a.styledLabel("DESCANSO (MIN)", 498, 558, 230, 20, a.smallFont, colorMuted, colorWhite)
	a.fieldBreak = a.control("EDIT", "5", wsBorder|esAutoHScroll|wsTabstop, 499, 585, 194, 37, 0)
	a.styledLabel("Modalidad: uso diario, semanal, mensual, horario o ciclos de pausa.", 291, 641, 819, 24, a.font, colorMuted, colorWhite)
	a.styledButton("✓  Guardar regla", 287, 680, 297, 50, idAdd, colorBlue, colorBlueDark, colorWhite, colorBlue)
	a.styledButton("Limpiar formulario", 598, 680, 251, 50, idClear, colorWhite, colorPaleBlue, colorText, colorBorder)
	a.styledButton("Volver a mis reglas", 861, 680, 254, 50, idRules, colorWhite, colorPaleBlue, colorBlue, colorBorder)

	// Preferencias y transparencia.
	a.buildPage = 3
	a.styledLabel("Control y privacidad", 290, 165, 560, 34, a.sectionFont, colorText, colorWhite)
	a.styledLabel("Decide cómo trabaja FocoLocal. Los cambios quedan guardados en tu equipo.", 290, 206, 796, 23, a.font, colorMuted, colorWhite)
	a.chkSafe = a.control("BUTTON", "Excluir scripts y automatizaciones (Python, Node, WebDriver)", bsAutoCheck|wsTabstop, 296, 279, 794, 31, idSafe)
	a.labelColors(a.chkSafe, colorText, colorWhite)
	a.styledLabel("No bloqueamos procesos por DNS, proxy o firewall.", 323, 314, 758, 25, a.smallFont, colorMuted, colorWhite)
	a.chkStop = a.control("BUTTON", "Detener vídeo y audio al agotar el tiempo", bsAutoCheck|wsTabstop, 296, 365, 794, 31, idStop)
	a.labelColors(a.chkStop, colorText, colorWhite)
	a.styledLabel("Sustituye la pestaña activa por una página local amable.", 323, 400, 755, 25, a.smallFont, colorMuted, colorWhite)
	a.chkAuto = a.control("BUTTON", "Abrir automáticamente con Windows", bsAutoCheck|wsTabstop, 296, 454, 794, 31, idAuto)
	a.labelColors(a.chkAuto, colorText, colorWhite)
	a.styledLabel("Puedes salir completamente desde la barra lateral.", 323, 491, 755, 23, a.smallFont, colorMuted, colorWhite)
	a.styledLabel("DOS MONITORES", 295, 546, 445, 21, a.smallFont, colorMuted, colorWhite)
	a.styledButton("⊞  Seguir / dejar de seguir ventana", 295, 580, 380, 48, idPin, colorPaleBlue, colorBorder, colorBlue, colorPaleBlue)
	a.styledLabel("Cuenta una ventana elegida aunque otra esté en primer plano,", 296, 643, 803, 23, a.font, colorText, colorWhite)
	a.styledLabel("solo si sigue visible y hay actividad reciente en el PC.", 296, 669, 802, 22, a.smallFont, colorMuted, colorWhite)
	a.styledLabel("No detecta la mirada. Las exclusiones de scripts siguen vigentes.", 296, 709, 790, 20, a.smallFont, colorMuted, colorWhite)
	a.styledLabel("APARIENCIA", 730, 548, 348, 21, a.smallFont, colorMuted, colorWhite)
	for i, option := range []string{"Sistema", "Claro", "Oscuro"} {
		a.themeButtons[i] = a.styledButton(option, int32(730+i*130), 579, 120, 43, idThemeSystem+i, colorWhite, colorPaleBlue, colorText, colorBorder)
	}
	a.styledLabel("Usa la apariencia que prefieras.", 733, 638, 392, 22, a.smallFont, colorMuted, colorWhite)
	a.buildPage = -1
	a.overlay = create("FocoLocal.Overlay.v2", "Límite alcanzado", exToolWindow|exTopMost|exNoActivate, wsPopup, 0, 0, 500, 400, 0, 0)
	show(a.overlay, swHide)
	a.overlayTitle = a.overlayLabel("Tiempo de descanso", 220, 145, 760, 58, true)
	a.overlayNote = a.overlayLabel("Puedes cambiar de actividad.", 220, 225, 760, 100, false)
	a.overlayControl("BUTTON", "Abrir otra pestaña", 0, 220, 350, 340, 46, idNewTab)
	mx := screenW - 515
	if mx < 0 {
		mx = 0
	}
	a.monitor = create("FocoLocal.Monitor.v2", "Monitor FocoLocal", exToolWindow|exTopMost|exNoActivate, wsPopup, mx, 40, 490, 94, 0, 0)
	if a.monitor != 0 {
		a.monitorText = create("STATIC", "Monitor: esperando navegador...", 0, wsVisible|wsChild|ssLeft, 12, 8, 465, 80, a.monitor, 0)
		setFont(a.monitorText, a.font)
		show(a.monitor, swHide)
	}
	a.showCategory(0)
	a.showPage(0)
	a.updateModeInfo()
	a.applyTheme(a.themeMode)
	a.tray(0)
}

func (a *App) control(class, text string, style uint32, x, y, w, h int32, id int) uintptr {
	hwnd := create(class, text, 0, wsVisible|wsChild|style, a.sx(x), a.sy(y), a.sx(w), a.sy(h), a.main, uintptr(id))
	if a.buildPage >= 0 && a.buildPage < 4 {
		a.pages[a.buildPage] = append(a.pages[a.buildPage], hwnd)
	}
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
	// A new catalog selection must not inherit days/months/sessions from
	// the previously edited rule; there are no hidden active settings.
	a.clearEditor()
	text(a.fieldDomain, p.Domain)
	// The catalog never enables, creates or updates a rule automatically.
	cfg, _ := a.store.Snapshot(time.Now())
	for _, r := range cfg.Rules {
		if r.Domain == p.Domain {
			a.populateRule(r)
			a.showPage(2)
			return
		}
	}
	send(a.comboPeriod, cbSetCurSel, 0, 0)
	send(a.comboUnit, cbSetCurSel, 0, 0)
	text(a.fieldMinutes, strconv.Itoa(p.Minutes))
	a.updateModeInfo()
	a.showPage(2)
}

func (a *App) addRule() {
	amount, err := strconv.Atoi(strings.TrimSpace(getText(a.fieldMinutes)))
	if err != nil {
		alert(a.main, "Ingresa una cantidad de tiempo válida.")
		return
	}
	if int(send(a.comboUnit, cbGetCurSel, 0, 0)) == 1 {
		amount *= 60
	}
	modes := []string{"day", "week", "month", "blocked", "schedule", "interval"}
	idx := int(send(a.comboPeriod, cbGetCurSel, 0, 0))
	if idx < 0 || idx >= len(modes) {
		idx = 0
	}
	days := uint8(0)
	switch int(send(a.comboDays, cbGetCurSel, 0, 0)) {
	case 1:
		days = 0b0111110
	case 2:
		days = 0b1000001
	}
	months, e := parseMonths(getText(a.fieldMonths))
	if e != nil {
		alert(a.main, e.Error())
		return
	}
	r := limit.Rule{Domain: getText(a.fieldDomain), Minutes: amount, Enabled: true, Period: modes[idx],
		From: strings.TrimSpace(getText(a.fieldFrom)), To: strings.TrimSpace(getText(a.fieldTo)), Weekdays: days, Months: months}
	r.SessionMinutes, _ = strconv.Atoi(strings.TrimSpace(getText(a.fieldSession)))
	r.BreakMinutes, _ = strconv.Atoi(strings.TrimSpace(getText(a.fieldBreak)))
	if r.Period == "blocked" {
		r.Minutes = 0
	}
	if err = a.store.UpsertRule(r); err != nil {
		alert(a.main, err.Error())
		return
	}
	a.save()
	a.refreshRules()
	a.showPage(0)
	text(a.info, "Guardado: "+limit.NormalizeDomain(r.Domain)+". La regla ya está activa.")
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
	a.ruleRows = nil
	totalToday, today := a.store.TodayUsage(time.Now())
	activeCount := 0
	for i, r := range cfg.Rules {
		spent := used[r.Domain]
		budget := int64(r.Minutes) * 60000
		remaining := budget - spent
		if remaining < 0 {
			remaining = 0
		}
		state := "● Activo"
		if !r.Enabled {
			state = "Ⅱ Pausado"
		}
		if budget > 0 && spent >= budget {
			state = "✓ Agotado"
		}
		if r.Enabled {
			activeCount++
		}
		suffix := r.Description()
		if r.Period == "blocked" {
			state = "⛔ Bloqueado"
		}
		line := fmt.Sprintf("%-23s   %-19s   hoy %s   |   período %s / %s   |   %s", r.Domain, suffix, formatDuration(today[r.Domain]), formatDuration(spent), formatDuration(budget), state)
		send(a.list, lbAddString, 0, su(line))
		a.ruleRows = append(a.ruleRows, ruleRow{Domain: r.Domain, Period: suffix, Today: today[r.Domain], Spent: spent, Budget: budget, State: state, Enabled: r.Enabled, Blocked: r.Period == "blocked"})
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
	text(a.statUsed, formatDuration(totalToday))
	if cfg.DailyTrackingSince != "" && strings.HasPrefix(cfg.DailyTrackingSince, time.Now().Format("2006-01-02")) {
		if started, e := time.Parse(time.RFC3339, cfg.DailyTrackingSince); e == nil {
			text(a.statUsedExplain, "Desde la actualización: "+started.Format("15:04"))
		}
	} else {
		deleted := false
		for d := range today {
			found := false
			for _, r := range cfg.Rules {
				if r.Domain == d {
					found = true
					break
				}
			}
			if !found {
				deleted = true
				break
			}
		}
		if deleted {
			text(a.statUsedExplain, "Incluye sitios eliminados hoy")
		} else {
			text(a.statUsedExplain, "Suma de sitios, sin duplicados")
		}
	}
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
	if s.Host != "" && s.Window != 0 && (fg() == s.Window || s.Pinned) && time.Since(s.At) <= 3300*time.Millisecond {
		text(a.active, fmt.Sprintf("● Contando: %s · %s\r\n%s", s.Browser, s.Host, map[bool]string{true: "Segunda pantalla • modo manual", false: "Ventana en primer plano • 1 s"}[s.Pinned]))
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
	if totalSeconds >= 3600 {
		return fmt.Sprintf("%02d:%02d:%02d", totalSeconds/3600, (totalSeconds/60)%60, totalSeconds%60)
	}
	return fmt.Sprintf("%02d:%02d", totalSeconds/60, totalSeconds%60)
}

func (a *App) updateMonitor() {
	if !a.monitorEnabled || a.monitorText == 0 {
		return
	}
	s := a.sampleCopy()
	if s.Host == "" || s.Window == 0 || (fg() != s.Window && !s.Pinned) || time.Since(s.At) > 3300*time.Millisecond {
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
	if (active != s.Window && !s.Pinned) || time.Since(s.At) > 3300*time.Millisecond || (s.Pinned && (!browserWindowVisible(s.Window) || !activeWithin(180000))) {
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

func (a *App) sx(v int32) int32 { return int32(float64(v) * a.scaleX) }
func (a *App) sy(v int32) int32 { return int32(float64(v) * a.scaleY) }
