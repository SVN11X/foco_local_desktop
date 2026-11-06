package windows

import (
	"syscall"
	"unsafe"
)

// The native Win32 look is intentionally self-contained: no WebView runtime,
// third-party DLLs, extension, network request, or system-wide theme change.
func color(r, g, b uint8) uint32 { return uint32(r) | uint32(g)<<8 | uint32(b)<<16 }

var (
	colorCanvas      = color(246, 249, 253)
	colorSidebar     = color(18, 36, 62)
	colorSidebarSoft = color(28, 51, 79)
	colorWhite       = color(255, 255, 255)
	colorText        = color(30, 47, 72)
	colorMuted       = color(87, 105, 130)
	colorSubtle      = color(155, 171, 193)
	colorBlue        = color(37, 99, 235)
	colorBlueDark    = color(27, 79, 192)
	colorPaleBlue    = color(233, 241, 255)
	colorGreen       = color(20, 125, 104)
	colorPaleGreen   = color(228, 249, 242)
	colorBorder      = color(219, 228, 241)
	colorError       = color(169, 54, 65)
)

type labelStyle struct{ ink, backdrop uint32 }
type buttonStyle struct{ normal, pressed, ink, stroke uint32 }

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
	a.buttonStyles[hwnd] = buttonStyle{bg, pressed, ink, stroke}
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
	left := rect{0, 0, 225, client.Bottom}
	call(pFillRect, dc, uintptr(unsafe.Pointer(&left)), a.brush(colorSidebar))
	// A few quiet surfaces provide hierarchy without decorative clutter.
	a.paintRounded(dc, rect{18, 122, 207, 219}, 18, colorSidebarSoft, colorSidebarSoft)
	for _, r := range []rect{
		{247, 99, 510, 182}, {524, 99, 787, 182}, {801, 99, 1066, 182},
		{247, 194, 1066, 405}, {247, 414, 1066, 591}, {247, 601, 1066, 710},
	} {
		a.paintRounded(dc, r, 18, colorWhite, colorBorder)
	}
	// Calm accent bar at the top of the new-rule card.
	a.paintRounded(dc, rect{271, 206, 276, 238}, 4, colorBlue, colorBlue)
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
	a.paintRounded(item.DC, item.RC, 12, bg, style.stroke)
	call(pSetBkMode, item.DC, 1)
	call(pSetTextColor, item.DC, uintptr(ink))
	old := call(pSelectObject, item.DC, a.font)
	value := getText(item.Window)
	r := item.RC
	call(pDrawText, item.DC, su(value), ^uintptr(0), uintptr(unsafe.Pointer(&r)), dtCenter|dtVCenter|dtSingleLine|dtEndEllipsis)
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
		style = labelStyle{colorText, colorCanvas}
	}
	call(pSetBkMode, dc, 1)
	call(pSetTextColor, dc, uintptr(style.ink))
	call(pSetBkColor, dc, uintptr(style.backdrop))
	return a.brush(style.backdrop)
}

// Color is not the only state indicator; labels also describe status.
