package windows

import (
	"fmt"
	"unsafe"
)

const (
	lbsOwnerDrawFixed = 0x0010
	lbsHasStrings     = 0x0040
	lbSetItemHeight   = 0x01A0
	odtListbox        = 2
)

type ruleRow struct {
	Domain, Period, State string
	Today, Spent, Budget  int64
	Enabled, Blocked      bool
}

// This custom-drawn list is a true Win32 LISTBOX: it keeps keyboard navigation,
// focus, accessibility role and selected index, but adds branded icons and
// truthful per-period / per-day numbers in the two-line row.
func (a *App) paintRuleRow(l uintptr) uintptr {
	if l == 0 || a.list == 0 {
		return 0
	}
	var item drawItemStruct
	call(pMoveMemory, uintptr(unsafe.Pointer(&item)), l, unsafe.Sizeof(item))
	if item.CtlType != odtListbox || item.Window != a.list {
		return 0
	}
	if int(item.ItemID) >= len(a.ruleRows) {
		return 1
	}
	row := a.ruleRows[item.ItemID]
	selected := item.ItemState&odsSelected != 0
	bg := colorWhite
	if selected {
		bg = colorPaleBlue
	}
	fill := item.RC
	call(pFillRect, item.DC, uintptr(unsafe.Pointer(&fill)), a.brush(bg))
	left := item.RC.Left + 15
	top := item.RC.Top + 8
	if !a.paintBrand(item.DC, row.Domain, left, top, 43, bg) {
		// Unavailable marks fall back to colored monograms, not invented logos.
		fallback := rect{left, top, left + 43, top + 43}
		a.paintRounded(item.DC, fallback, 14, colorBlueDark, colorBlueDark)
		call(pSetBkMode, item.DC, 1)
		call(pSetTextColor, item.DC, uintptr(color(255, 255, 255)))
		old := call(pSelectObject, item.DC, a.font)
		initial := string([]rune(row.Domain)[:1])
		call(pDrawText, item.DC, su(initial), ^uintptr(0), uintptr(unsafe.Pointer(&fallback)), dtVCenter|dtCenter|dtSingleLine)
		call(pSelectObject, item.DC, old)
	}
	x := left + 59
	w := item.RC.Right - x - 10
	if w < 100 {
		return 1
	}
	drawLine := func(msg string, r rect, font uintptr, ink uint32) {
		call(pSetBkMode, item.DC, 1)
		call(pSetTextColor, item.DC, uintptr(ink))
		old := call(pSelectObject, item.DC, font)
		call(pDrawText, item.DC, su(msg), ^uintptr(0), uintptr(unsafe.Pointer(&r)), dtSingleLine|dtEndEllipsis|dtVCenter)
		call(pSelectObject, item.DC, old)
	}
	drawLine(row.Domain, rect{x, top, x + w/2, top + 22}, a.font, colorText)
	today := fmt.Sprintf("Hoy: %s", formatDuration(row.Today))
	drawLine(today, rect{x + w/2, top, x + w - 10, top + 22}, a.font, colorBlue)
	text := fmt.Sprintf("%s · período: %s / %s · %s", row.Period, formatDuration(row.Spent), formatDuration(row.Budget), row.State)
	if row.Blocked {
		text = "Bloqueo continuo · " + row.State
	}
	drawLine(text, rect{x, top + 23, x + w, top + 42}, a.smallFont, colorMuted)
	base := rect{x, top + 49, x + w - 14, top + 54}
	if base.Right > base.Left {
		call(pFillRect, item.DC, uintptr(unsafe.Pointer(&base)), a.brush(colorBorder))
		if row.Budget > 0 {
			progress := float64(row.Spent) / float64(row.Budget)
			if progress < 0 {
				progress = 0
			}
			if progress > 1 {
				progress = 1
			}
			base.Right = base.Left + int32(float64(base.Right-base.Left)*progress)
			if base.Right > base.Left {
				call(pFillRect, item.DC, uintptr(unsafe.Pointer(&base)), a.brush(colorBlue))
			}
		}
	}
	if item.ItemState&odsFocus != 0 {
		focus := item.RC
		focus.Left += 1
		focus.Right -= 1
		call(pDrawFocusRect, item.DC, uintptr(unsafe.Pointer(&focus)))
	}
	return 1
}
