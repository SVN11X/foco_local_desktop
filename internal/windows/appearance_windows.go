package windows

import (
	"syscall"
	"unsafe"
)

const (
	wmCtlColorEdit    = 0x0133
	wmCtlColorListbox = 0x0134
)

var (
	dwmapi                 = syscall.NewLazyDLL("dwmapi.dll")
	pDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
	pRegOpenKeyExW         = advapi32.NewProc("RegOpenKeyExW")
	pRegQueryValueExW      = advapi32.NewProc("RegQueryValueExW")
	uxtheme                = syscall.NewLazyDLL("uxtheme.dll")
	pSetWindowTheme        = uxtheme.NewProc("SetWindowTheme")
)

type palette [14]uint32

func currentPalette() palette {
	return palette{colorCanvas, colorSidebar, colorSidebarSoft, colorWhite, colorText, colorMuted, colorSubtle, colorBlue, colorBlueDark, colorPaleBlue, colorGreen, colorPaleGreen, colorBorder, colorError}
}
func usePalette(p palette) {
	colorCanvas, colorSidebar, colorSidebarSoft, colorWhite = p[0], p[1], p[2], p[3]
	colorText, colorMuted, colorSubtle, colorBlue = p[4], p[5], p[6], p[7]
	colorBlueDark, colorPaleBlue, colorGreen, colorPaleGreen = p[8], p[9], p[10], p[11]
	colorBorder, colorError = p[12], p[13]
}

var lightPalette = palette{
	color(244, 248, 252), color(15, 31, 54), color(29, 53, 82), color(255, 255, 255),
	color(30, 47, 72), color(87, 105, 130), color(155, 171, 193), color(43, 91, 221),
	color(27, 79, 192), color(233, 241, 255), color(20, 125, 104), color(228, 249, 242),
	color(222, 231, 243), color(169, 54, 65),
}
var darkPalette = palette{
	color(15, 22, 34), color(8, 17, 33), color(26, 43, 67), color(27, 40, 58),
	color(244, 247, 253), color(184, 201, 221), color(140, 164, 191), color(120, 168, 255),
	color(63, 116, 228), color(35, 60, 94), color(103, 226, 187), color(25, 63, 58),
	color(57, 78, 107), color(250, 138, 145),
}

func windowsDarkPreference() bool {
	const hkcu = uintptr(0x80000001)
	var key uintptr
	if call(pRegOpenKeyExW, hkcu, su(`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`), 0, 0x20019, uintptr(unsafe.Pointer(&key))) != 0 {
		return false
	}
	defer call(pRegCloseKey, key)
	var val, typ, size uint32
	size = 4
	if call(pRegQueryValueExW, key, su("AppsUseLightTheme"), 0, uintptr(unsafe.Pointer(&typ)), uintptr(unsafe.Pointer(&val)), uintptr(unsafe.Pointer(&size))) != 0 {
		return false
	}
	return typ == 4 && val == 0
}
func resolveDark(mode string) bool {
	return mode == "dark" || (mode == "system" && windowsDarkPreference())
}
func (a *App) initPalette(mode string) {
	a.isDark = resolveDark(mode)
	if a.isDark {
		usePalette(darkPalette)
	} else {
		usePalette(lightPalette)
	}
}
func (a *App) setDarkTitleBar(dark bool) {
	if a.main == 0 {
		return
	}
	var value uint32
	if dark {
		value = 1
	}
	call(pDwmSetWindowAttribute, a.main, 20, uintptr(unsafe.Pointer(&value)), 4)
}
func (a *App) applyTheme(mode string) {
	a.themeMode = mode
	dark := resolveDark(mode)
	old := currentPalette()
	next := lightPalette
	if dark {
		next = darkPalette
	}
	colors := make(map[uint32]uint32, len(old))
	for i, c := range old {
		colors[c] = next[i]
	}
	translate := func(c uint32) uint32 {
		if n, ok := colors[c]; ok {
			return n
		}
		return c
	}
	for h, s := range a.labelStyles {
		if !s.fixedInk {
			s.ink = translate(s.ink)
		}
		s.backdrop = translate(s.backdrop)
		a.labelStyles[h] = s
	}
	for h, s := range a.buttonStyles {
		s.normal = translate(s.normal)
		s.pressed = translate(s.pressed)
		if !s.fixedInk {
			s.ink = translate(s.ink)
		}
		s.stroke = translate(s.stroke)
		a.buttonStyles[h] = s
	}
	usePalette(next)
	a.isDark = dark
	a.setDarkTitleBar(dark)
	for i, h := range a.themeButtons {
		if h == 0 {
			continue
		}
		s := a.buttonStyles[h]
		if (i == 0 && mode == "system") || (i == 1 && mode == "light") || (i == 2 && mode == "dark") {
			s.normal = colorBlueDark
			s.pressed = colorBlue
			s.ink = color(255, 255, 255)
			s.stroke = colorBlueDark
		} else {
			s.normal = colorWhite
			s.pressed = colorPaleBlue
			s.ink = colorText
			s.stroke = colorBorder
		}
		a.buttonStyles[h] = s
	}
	// Native edit/combo/list widgets do not automatically inherit custom palette.
	// SetWindowTheme is best effort; WM_CTLCOLOREDIT/LISTBOX provide a safe fallback.
	theme := "Explorer"
	if dark {
		theme = "DarkMode_Explorer"
	}
	for _, h := range []uintptr{a.list, a.fieldDomain, a.fieldMinutes, a.fieldFrom, a.fieldTo, a.fieldSession, a.fieldBreak, a.fieldMonths, a.comboPeriod, a.comboDays, a.comboUnit, a.chkSafe, a.chkAuto, a.chkStop} {
		if h != 0 {
			call(pSetWindowTheme, h, su(theme), 0)
			call(pInvalidateRect, h, 0, 1)
		}
	}
	if a.main != 0 {
		for h := range a.labelStyles {
			call(pInvalidateRect, h, 0, 1)
		}
		for h := range a.buttonStyles {
			call(pInvalidateRect, h, 0, 1)
		}
		call(pInvalidateRect, a.main, 0, 1)
	}
}
