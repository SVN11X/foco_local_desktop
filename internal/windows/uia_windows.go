package windows

import (
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"focolocal-desktop/internal/limit"
)

// UI Automation is used strictly read-only: no extensions, proxy, TLS interception or browser APIs.
// All COM calls must occur on a single locked OS thread.
type guid struct {
	Data1        uint32
	Data2, Data3 uint16
	Data4        [8]byte
}

var clsidUI = guid{0xff48dba4, 0x60ef, 0x4201, [8]byte{0xaa, 0x87, 0x54, 0x10, 0x3e, 0xef, 0x59, 0x4e}}
var iidUI = guid{0x30cbe57d, 0xd9d0, 0x452a, [8]byte{0xab, 0x13, 0x7a, 0xc5, 0xac, 0x48, 0x25, 0xee}}
var pCoInit = ole32.NewProc("CoInitializeEx")
var pCoUninit = ole32.NewProc("CoUninitialize")
var pCoCreate = ole32.NewProc("CoCreateInstance")
var pBStrLen = oleaut32.NewProc("SysStringLen")
var pBStrFree = oleaut32.NewProc("SysFreeString")

// VARIANT 24 bytes on win-amd64; VT_I4 = 3.
type variant struct {
	VT                              uint16
	Reserved1, Reserved2, Reserved3 uint16
	Val                             uint64
	Extra                           uint64
}

func ok(hr uintptr) bool { return int32(hr) >= 0 }
func vcall(obj unsafe.Pointer, idx uintptr, args ...uintptr) uintptr {
	if obj == nil {
		return ^uintptr(0)
	}
	// COM owns these allocations; preserve their pointer type instead of turning
	// them into integers and reconstructing pointers from uintptr values.
	vt := *(*unsafe.Pointer)(obj)
	if vt == nil {
		return ^uintptr(0)
	}
	fn := *(*uintptr)(unsafe.Add(vt, idx*unsafe.Sizeof(uintptr(0))))
	params := make([]uintptr, 0, 1+len(args))
	params = append(params, uintptr(obj))
	params = append(params, args...)
	hr, _, _ := syscall.SyscallN(fn, params...)
	runtime.KeepAlive(obj)
	return hr
}
func release(obj unsafe.Pointer) {
	if obj != nil {
		vcall(obj, 2)
	}
}
func bstring(v unsafe.Pointer) string {
	if v == nil {
		return ""
	}
	n := call(pBStrLen, uintptr(v))
	if n > 8192 {
		n = 8192
	}
	str := syscall.UTF16ToString(unsafe.Slice((*uint16)(v), int(n)))
	call(pBStrFree, uintptr(v))
	return str
}
func uiastr(obj unsafe.Pointer, idx uintptr) string {
	var b unsafe.Pointer
	if !ok(vcall(obj, idx, uintptr(unsafe.Pointer(&b)))) {
		return ""
	}
	return bstring(b)
}

type detectedSite struct {
	Host   string
	Target string
}

type scanner struct {
	auto      unsafe.Pointer
	editCond  unsafe.Pointer
	comboCond unsafe.Pointer
}

func newScanner() *scanner {
	// Created and used exclusively by the locked worker thread.
	if !ok(call(pCoInit, 0, 2)) {
		return nil
	}
	var auto unsafe.Pointer
	hr := call(pCoCreate, uintptr(unsafe.Pointer(&clsidUI)), 0, 1, uintptr(unsafe.Pointer(&iidUI)), uintptr(unsafe.Pointer(&auto)))
	if !ok(hr) || auto == nil {
		call(pCoUninit)
		return nil
	}
	conditionFor := func(controlType uint64) unsafe.Pointer {
		v := variant{VT: 3, Val: controlType}
		var c unsafe.Pointer
		hr := vcall(auto, 23, 30003, uintptr(unsafe.Pointer(&v)), uintptr(unsafe.Pointer(&c)))
		if !ok(hr) {
			return nil
		}
		return c
	}
	edit := conditionFor(50004)  // UIA_EditControlTypeId
	combo := conditionFor(50003) // UIA_ComboBoxControlTypeId
	if edit == nil && combo == nil {
		release(auto)
		call(pCoUninit)
		return nil
	}
	return &scanner{auto, edit, combo}
}
func (s *scanner) close() {
	if s != nil {
		release(s.editCond)
		release(s.comboCond)
		release(s.auto)
		call(pCoUninit)
	}
}

// Recognize address fields by multilingual accessibility name/automation ID.
// Name heuristics intentionally prefer a false negative over matching input boxes in a page.
func addressScore(name, id string) int {
	s := strings.ToLower(name + " " + id)
	switch {
	case strings.Contains(s, "omnibox"), strings.Contains(s, "urlbar"), strings.Contains(s, "address and search"), strings.Contains(s, "barra de direc"), strings.Contains(s, "barra de búsq"), strings.Contains(s, "barre d'adresse"), strings.Contains(s, "adressleiste"), strings.Contains(s, "address bar"), strings.Contains(s, "search or enter address"), strings.Contains(s, "search with google or enter address"):
		return 10
	case strings.Contains(s, "address"), strings.Contains(s, "dirección"), strings.Contains(s, "direcciones"), strings.Contains(s, "location"), strings.Contains(s, "adresse"), strings.Contains(s, "endereço"), strings.Contains(s, "dirección web"):
		return 7
	case strings.Contains(s, "search"), strings.Contains(s, "búsqueda"), strings.Contains(s, "recherche"), strings.Contains(s, "pesquisa"):
		return 1 // not confident: we refuse to use lone search fields
	}
	return 0
}
func (s *scanner) urlForWindow(hwnd uintptr) detectedSite {
	if s == nil {
		return detectedSite{}
	}
	var root unsafe.Pointer
	if !ok(vcall(s.auto, 6, hwnd, uintptr(unsafe.Pointer(&root)))) || root == nil {
		return detectedSite{}
	}
	defer release(root)
	for _, condition := range []unsafe.Pointer{s.editCond, s.comboCond} {
		if condition == nil {
			continue
		}
		var arr unsafe.Pointer
		if !ok(vcall(root, 6, 4, uintptr(condition), uintptr(unsafe.Pointer(&arr)))) || arr == nil {
			continue
		}
		var n int32
		if !ok(vcall(arr, 3, uintptr(unsafe.Pointer(&n)))) {
			release(arr)
			continue
		}
		if n > 100 {
			n = 100
		}
		bestScore := 0
		best := detectedSite{}
		for i := int32(0); i < n; i++ {
			var el unsafe.Pointer
			if !ok(vcall(arr, 4, uintptr(i), uintptr(unsafe.Pointer(&el)))) || el == nil {
				continue
			}
			score := addressScore(uiastr(el, 23), uiastr(el, 29))
			if score >= 7 {
				var pattern unsafe.Pointer
				if ok(vcall(el, 16, 10002, uintptr(unsafe.Pointer(&pattern)))) && pattern != nil {
					value := uiastr(pattern, 4)
					release(pattern)
					host := limit.HostFromAddress(value)
					if host != "" && score > bestScore {
						bestScore = score
						best = detectedSite{Host: host, Target: limit.SiteFromAddress(value)}
					}
				}
			}
			release(el)
			if bestScore >= 10 {
				break
			}
		}
		release(arr)
		if best.Host != "" {
			return best
		}
	}
	return detectedSite{}
}

func runScanner(app *App) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	s := newScanner()
	if s != nil {
		defer s.close()
	}
	// Diagnostic status is shown in the hub if UIA fails; do not guess visited sites.
	app.publish(Sample{Reason: "Esperando navegador activo...", Working: s != nil})
	for {
		select {
		case <-app.done:
			return
		default:
		}
		hwnd := fg()
		sample := Sample{Working: s != nil, Reason: "Fuera del navegador"}
		app.mu.Lock()
		pin, pinPID := app.pinnedWindow, app.pinnedPID
		app.mu.Unlock()
		// A selected window on another display may be counted without changing
		// keyboard focus; no webcam or gaze inference. Require recent PC activity.
		pinned := false
		if pin != 0 && pinPID != 0 && hwnd != pin && hwnd != app.main && hwnd != app.overlay && browserWindowVisible(pin) && windowPID(pin) == pinPID && activeWithin(180000) {
			// Manual pin takes priority until disabled, even if another
			// browser is active on the primary monitor. We never pretend to
			// know which screen the user's eyes are looking at.
			hwnd = pin
			pinned = true
		}
		if hwnd == app.main && !pinned {
			sample.Reason = "Hub en primer plano: el navegador no se está contando"
		} else if hwnd != 0 && hwnd != app.overlay && hwnd != app.monitor {
			pid := windowPID(hwnd)
			exe := foregroundExe(pid)
			if browserName(exe) != "" {
				sample.PID = pid
				sample.Window = hwnd
				sample.Pinned = pinned
				sample.Browser = browserName(exe)
				sample.Reason = "Buscando dirección en " + sample.Browser
				if s != nil {
					cfg, _ := app.store.Snapshot(app.now())
					if app.store.IsExcluded(pid) {
						sample.Reason = "Proceso excluido manualmente"
					} else if cfg.SafeAutomation && isAutomated(pid) {
						sample.Reason = "Automatización protegida"
					} else {
						detected := s.urlForWindow(hwnd)
						sample.Host, sample.Site = detected.Host, detected.Target
						if sample.Host == "" {
							sample.Reason = "No se reconoce la barra de direcciones (UIA)"
						} else {
							sample.Reason = "Detectado"
						}
					}
				} else {
					sample.Reason = "UI Automation no disponible"
				}
			}
		}
		app.publish(sample)
		select {
		case <-app.done:
			return
		case <-app.scanDelay:
		}
	}
}
