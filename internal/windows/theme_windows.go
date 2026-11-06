package windows

import (
	"syscall"
	"unsafe"
)

// The native Win32 look is intentionally self-contained: no WebView runtime,
// third-party DLLs, extension, network request, or system-wide theme change.
func color(r, g, b uint8) uint32 { return uint32(r) | uint32(g)<<8 | uint32(b)<<16 }

var (
	colorCanvas      = color(244, 248, 252)
	colorSidebar     = color(15, 31, 54)
	colorSidebarSoft = color(29, 53, 82)
	colorWhite       = color(255, 255, 255)
	colorText        = color(30, 47, 72)
	colorMuted       = color(87, 105, 130)
	colorSubtle      = color(155, 171, 193)
	colorBlue        = color(43, 91, 221)
	colorBlueDark    = color(27, 79, 192)
	colorPaleBlue    = color(233, 241, 255)
	colorGreen       = color(20, 125, 104)
	colorPaleGreen   = color(228, 249, 242)
	colorBorder      = color(222, 231, 243)
	colorError       = color(169, 54, 65)
)

type labelStyle struct{ ink, backdrop uint32 }
type buttonStyle struct {
	normal, pressed, ink, stroke uint32
	badge                        string
	logo                         string
	accent                       uint32
}

var (
	pBeginPaint       = user32.NewProc("BeginPaint")
	pEndPaint         = user32.NewProc("EndPaint")
	pGetClientRect    = user32.NewProc("GetClientRect")
	pSetTextColor     = gdi32.NewProc("SetTextColor")
	pSetBkColor       = gdi32.NewProc("SetBkColor")
	pSetBkMode        = gdi32.NewProc("SetBkMode")
	pCreateSolidBrush = gdi32.NewProc("CreateSolidBrush")
	pFillRect         = user32.NewProc("FillRect")
	pCreatePen        = gdi32.NewProc("CreatePen")
	pSelectObject     = gdi32.NewProc("SelectObject")
	pRoundRect        = gdi32.NewProc("RoundRect")
	pDrawText         = user32.NewProc("DrawTextW")
	pDrawFocusRect    = user32.NewProc("DrawFocusRect")
	pInvalidateRect   = user32.NewProc("InvalidateRect")
	pMoveMemory       = syscall.NewLazyDLL("ntdll.dll").NewProc("RtlMoveMemory")
)

const (
	wmPaint           = 0x000F
	wmEraseBackground = 0x0014
	wmDrawItem        = 0x002B
	wmCtlColorStatic  = 0x0138
	wmCtlColorBtn     = 0x0135
	wmSetRedraw       = 0x000B
	bsOwnerDraw       = 0x000B
	dtCenter          = 0x00000001
	dtVCenter         = 0x00000004
	dtSingleLine      = 0x00000020
	dtEndEllipsis     = 0x00008000
	odsSelected       = 0x0001
	odsFocus          = 0x0010
	odsDisabled       = 0x0004
	psSolid           = 0
)

type paintStruct struct {
	DC        uintptr
	Erase     int32
	RC        rect
	Restore   int32
	IncUpdate int32
	Reserved  [32]byte
}
type drawItemStruct struct {
	CtlType, CtlID, ItemID, ItemAction, ItemState uint32
	Window, DC                                    uintptr
	RC                                            rect
	Data                                          uintptr
}

func (a *App) brush(c uint32) uintptr {
	if a.brushes == nil {
		a.brushes = map[uint32]uintptr{}
	}
	if b := a.brushes[c]; b != 0 {
		return b
	}
	b := call(pCreateSolidBrush, uintptr(c))
	a.brushes[c] = b
	return b
}

func (a *App) labelColors(hwnd uintptr, ink, bg uint32) {
	if a.labelStyles == nil {
		a.labelStyles = map[uintptr]labelStyle{}
	}
	a.labelStyles[hwnd] = labelStyle{ink, bg}
}
func (a *App) styledLabel(value string, x, y, w, h int32, font uintptr, ink, bg uint32) uintptr {
	hwnd := a.label(value, x, y, w, h, false)
	if font != 0 {
		setFont(hwnd, font)
	}
	a.labelColors(hwnd, ink, bg)
	return hwnd
}

func (a *App) styledButton(value string, x, y, w, h int32, id int, bg, pressed, ink, stroke uint32) uintptr {
	hwnd := a.control("BUTTON", value, bsOwnerDraw|wsTabstop, x, y, w, h, id)
	if a.buttonStyles == nil {
		a.buttonStyles = map[uintptr]buttonStyle{}
	}
	a.buttonStyles[hwnd] = buttonStyle{normal: bg, pressed: pressed, ink: ink, stroke: stroke}
	return hwnd
}
func (a *App) paintRounded(dc uintptr, r rect, radius int32, bg, outline uint32) {
	pen := call(pCreatePen, psSolid, 1, uintptr(outline))
	oldPen := call(pSelectObject, dc, pen)
	oldBrush := call(pSelectObject, dc, a.brush(bg))
	call(pRoundRect, dc, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right), uintptr(r.Bottom), uintptr(radius), uintptr(radius))
	call(pSelectObject, dc, oldBrush)
	call(pSelectObject, dc, oldPen)
	call(pDeleteObject, pen)
}

func (a *App) renderBackground(hwnd uintptr) {
	var ps paintStruct
	dc := call(pBeginPaint, hwnd, uintptr(unsafe.Pointer(&ps)))
	if dc == 0 {
		return
	}
	defer call(pEndPaint, hwnd, uintptr(unsafe.Pointer(&ps)))
	var client rect
	call(pGetClientRect, hwnd, uintptr(unsafe.Pointer(&client)))
	call(pFillRect, dc, uintptr(unsafe.Pointer(&client)), a.brush(colorCanvas))
	sidebar := rect{0, 0, a.sx(238), client.Bottom}
	call(pFillRect, dc, uintptr(unsafe.Pointer(&sidebar)), a.brush(colorSidebar))
	// Pages keep consistent reading order, generous whitespace, and crisp cards.
	panels := []rect{}
	switch a.page {
	case 0:
		panels = []rect{{261, 152, 553, 250}, {563, 152, 855, 250}, {865, 152, 1153, 250},
			{261, 267, 1153, 577}, {261, 592, 1153, 768}}
	default:
		panels = []rect{{261, 152, 1153, 768}}
	}
	for _, r := range panels {
		a.paintRounded(dc, rect{a.sx(r.Left), a.sy(r.Top), a.sx(r.Right), a.sy(r.Bottom)}, 24, colorWhite, colorBorder)
	}
	if a.page == 2 {
		a.paintRounded(dc, rect{a.sx(277), a.sy(175), a.sx(283), a.sy(224)}, 6, colorBlue, colorBlue)
	}
	if a.page == 1 {
		a.paintRounded(dc, rect{a.sx(277), a.sy(175), a.sx(283), a.sy(222)}, 6, colorGreen, colorGreen)
	}
	if a.page == 3 {
		a.paintRounded(dc, rect{a.sx(277), a.sy(175), a.sx(283), a.sy(222)}, 6, colorBlue, colorBlue)
	}
}

func (a *App) paintButton(l uintptr) uintptr {
	if l == 0 {
		return 0
	}
	var storage drawItemStruct
	// lParam is a Windows-owned DRAWITEMSTRUCT pointer. Copy it via the OS
	// rather than reconstructing a Go pointer from a uintptr (go vet).
	call(pMoveMemory, uintptr(unsafe.Pointer(&storage)), l, unsafe.Sizeof(storage))
	item := &storage
	style, found := a.buttonStyles[item.Window]
	if !found {
		return 0
	}
	bg := style.normal
	if item.ItemState&odsSelected != 0 {
		bg = style.pressed
	}
	ink := style.ink
	if item.ItemState&odsDisabled != 0 {
		bg = colorBorder
		ink = colorMuted
	}
	a.paintRounded(item.DC, item.RC, 14, bg, style.stroke)
	call(pSetBkMode, item.DC, 1)
	call(pSetTextColor, item.DC, uintptr(ink))
	old := call(pSelectObject, item.DC, a.font)
	value := getText(item.Window)
	r := item.RC
	flags := uintptr(dtCenter | dtVCenter | dtSingleLine | dtEndEllipsis)
	if style.badge != "" {
		mark := rect{r.Left + 15, r.Top + 16, r.Left + 67, r.Bottom - 16}
		if !a.paintBrand(item.DC, style.logo, mark.Left, mark.Top-10, 45, bg) {
			a.paintRounded(item.DC, mark, 15, style.accent, style.accent)
			call(pSetTextColor, item.DC, uintptr(color(255, 255, 255)))
			call(pDrawText, item.DC, su(style.badge), ^uintptr(0), uintptr(unsafe.Pointer(&mark)), dtCenter|dtVCenter|dtSingleLine)
		}
		call(pSetTextColor, item.DC, uintptr(ink))
		r.Left += 80
		flags = uintptr(dtVCenter | dtSingleLine | dtEndEllipsis)
	}
	call(pDrawText, item.DC, su(value), ^uintptr(0), uintptr(unsafe.Pointer(&r)), flags)
	call(pSelectObject, item.DC, old)
	if item.ItemState&odsFocus != 0 {
		r.Left += 5
		r.Top += 4
		r.Right -= 5
		r.Bottom -= 4
		call(pDrawFocusRect, item.DC, uintptr(unsafe.Pointer(&r)))
	}
	return 1
}

func (a *App) ctlColor(dc, hwnd uintptr) uintptr {
	style, ok := a.labelStyles[hwnd]
	if !ok {
		style = labelStyle{colorText, colorWhite}
	}
	call(pSetBkMode, dc, 1)
	call(pSetTextColor, dc, uintptr(style.ink))
	call(pSetBkColor, dc, uintptr(style.backdrop))
	return a.brush(style.backdrop)
}

// Color is not the only state indicator; labels also describe status.
