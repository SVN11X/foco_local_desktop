package windows

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"focolocal-desktop/internal/catalog"
	"focolocal-desktop/internal/limit"
)

func parseMonths(raw string) (uint16, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	var mask uint16
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return 0, errors.New("Meses: usa 1,2,12 o 4-9")
		}
		nums := strings.Split(part, "-")
		if len(nums) > 2 {
			return 0, errors.New("Rango de meses incorrecto")
		}
		start, e := strconv.Atoi(strings.TrimSpace(nums[0]))
		if e != nil || start < 1 || start > 12 {
			return 0, errors.New("Los meses deben ir del 1 al 12")
		}
		end := start
		if len(nums) == 2 {
			end, e = strconv.Atoi(strings.TrimSpace(nums[1]))
			if e != nil || end < start || end > 12 {
				return 0, errors.New("Rango de meses inválido")
			}
		}
		for i := start; i <= end; i++ {
			mask |= 1 << uint(i-1)
		}
	}
	return mask, nil
}
func monthsLabel(mask uint16) string {
	if mask == 0 {
		return ""
	}
	var parts []string
	for i := 1; i <= 12; i++ {
		if mask&(1<<uint(i-1)) != 0 {
			parts = append(parts, strconv.Itoa(i))
		}
	}
	return strings.Join(parts, ",")
}
func (a *App) showPage(page int) {
	if page < 0 || page >= 4 {
		return
	}
	a.page = page
	for idx, elements := range a.pages {
		how := int32(swHide)
		if idx == page {
			how = swShow
		}
		for _, hwnd := range elements {
			show(hwnd, how)
		}
	}
	for idx, hwnd := range a.navButtons {
		if hwnd == 0 {
			continue
		}
		style := a.buttonStyles[hwnd]
		if idx == page {
			style.normal = colorBlue
			style.pressed = colorBlueDark
			style.stroke = colorBlue
		} else {
			style.normal = colorSidebar
			style.pressed = colorSidebarSoft
			style.stroke = colorSidebar
		}
		a.buttonStyles[hwnd] = style
		call(pInvalidateRect, hwnd, 0, 1)
	}
	call(pInvalidateRect, a.main, 0, 1)
}
func (a *App) showCategory(category int) {
	if category < 0 || category >= len(catalog.Categories) {
		return
	}
	a.category = category
	ids := catalog.InCategory(category)
	for i, btn := range a.siteButtons {
		if i >= len(ids) {
			show(btn, swHide)
			continue
		}
		p := catalog.Sites[ids[i]]
		text(btn, p.Name)
		style := a.buttonStyles[btn]
		style.badge = p.Icon
		style.logo = p.Domain
		style.accent = color(p.Color[0], p.Color[1], p.Color[2])
		a.buttonStyles[btn] = style
		show(btn, swShow)
		call(pInvalidateRect, btn, 0, 1)
	}
	for i, btn := range a.categoryButtons {
		style := a.buttonStyles[btn]
		if category == i {
			style.normal = colorPaleBlue
			style.pressed = colorBorder
			style.ink = colorBlue
			style.stroke = colorBlue
		} else {
			style.normal = colorWhite
			style.pressed = colorPaleBlue
			style.ink = colorText
			style.stroke = colorBorder
		}
		a.buttonStyles[btn] = style
		call(pInvalidateRect, btn, 0, 1)
	}
}
func (a *App) updateModeInfo() {
	mode := int(send(a.comboPeriod, cbGetCurSel, 0, 0))
	// Controls stay visible, but irrelevant options are disabled and greyed out.
	for _, h := range []uintptr{a.fieldFrom, a.fieldTo} {
		call(pEnableWindow, h, boolptr(mode == 4))
	}
	for _, h := range []uintptr{a.fieldSession, a.fieldBreak} {
		call(pEnableWindow, h, boolptr(mode == 5))
	}
	for _, h := range []uintptr{a.fieldMinutes, a.comboUnit} {
		call(pEnableWindow, h, boolptr(mode != 3))
	}
}
func boolptr(v bool) uintptr {
	if v {
		return 1
	}
	return 0
}
func (a *App) clearEditor() {
	text(a.fieldDomain, "")
	text(a.fieldMinutes, "30")
	text(a.fieldFrom, "09:00")
	text(a.fieldTo, "18:00")
	text(a.fieldSession, "25")
	text(a.fieldBreak, "5")
	text(a.fieldMonths, "")
	send(a.comboPeriod, cbSetCurSel, 0, 0)
	send(a.comboUnit, cbSetCurSel, 0, 0)
	send(a.comboDays, cbSetCurSel, 0, 0)
	a.updateModeInfo()
}
func (a *App) populateRule(r limit.Rule) {
	text(a.fieldDomain, r.Domain)
	modes := []string{"day", "week", "month", "blocked", "schedule", "interval"}
	idx := 0
	for i, mode := range modes {
		if r.Period == mode {
			idx = i
		}
	}
	send(a.comboPeriod, cbSetCurSel, uintptr(idx), 0)
	quantity := r.Minutes
	unit := 0
	if quantity >= 60 && quantity%60 == 0 {
		quantity /= 60
		unit = 1
	}
	if r.Period == "blocked" {
		quantity = 0
	}
	text(a.fieldMinutes, strconv.Itoa(quantity))
	send(a.comboUnit, cbSetCurSel, uintptr(unit), 0)
	if r.From != "" {
		text(a.fieldFrom, r.From)
	} else {
		text(a.fieldFrom, "09:00")
	}
	if r.To != "" {
		text(a.fieldTo, r.To)
	} else {
		text(a.fieldTo, "18:00")
	}
	if r.SessionMinutes > 0 {
		text(a.fieldSession, strconv.Itoa(r.SessionMinutes))
	} else {
		text(a.fieldSession, "25")
	}
	if r.BreakMinutes > 0 {
		text(a.fieldBreak, strconv.Itoa(r.BreakMinutes))
	} else {
		text(a.fieldBreak, "5")
	}
	text(a.fieldMonths, monthsLabel(r.Months))
	idxDays := 0
	if r.Weekdays == 0b0111110 {
		idxDays = 1
	} else if r.Weekdays == 0b1000001 {
		idxDays = 2
	}
	send(a.comboDays, cbSetCurSel, uintptr(idxDays), 0)
	a.updateModeInfo()
}
func (a *App) editSelected() {
	domain := a.selected()
	if domain == "" {
		alert(a.main, "Selecciona una regla para editar.")
		return
	}
	cfg, _ := a.store.Snapshot(time.Now())
	for _, r := range cfg.Rules {
		if r.Domain == domain {
			a.populateRule(r)
			a.showPage(2)
			return
		}
	}
}
func (a *App) togglePinned() {
	a.mu.Lock()
	if a.pinnedWindow != 0 {
		a.pinnedWindow = 0
		a.pinnedPID = 0
		a.mu.Unlock()
		alert(a.main, "Se dejó de seguir la ventana secundaria. Ahora solo contará el navegador en primer plano.")
		return
	}
	hwnd, pid := a.lastBrowserWindow, a.lastBrowserPID
	a.mu.Unlock()
	if hwnd == 0 || pid == 0 || windowPID(hwnd) != pid || !browserWindowVisible(hwnd) {
		alert(a.main, "Primero activa el navegador de la segunda pantalla durante unos segundos y luego pulsa este botón.")
		return
	}
	cfg, _ := a.store.Snapshot(time.Now())
	if cfg.SafeAutomation && isAutomated(pid) {
		alert(a.main, "La ventana pertenece a una automatización protegida.")
		return
	}
	a.mu.Lock()
	a.pinnedWindow = hwnd
	a.pinnedPID = pid
	a.mu.Unlock()
	alert(a.main, fmt.Sprintf("Siguiendo ventana %d de PID %d. Se contabilizará mientras sea visible y el PC tenga actividad reciente. Pulsa de nuevo para detener.", hwnd, pid))
}
