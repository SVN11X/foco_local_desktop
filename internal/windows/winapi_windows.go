package windows

import (
	"syscall"
	"unsafe"
)

var (
	user32                    = syscall.NewLazyDLL("user32.dll")
	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	gdi32                     = syscall.NewLazyDLL("gdi32.dll")
	shell32                   = syscall.NewLazyDLL("shell32.dll")
	advapi32                  = syscall.NewLazyDLL("advapi32.dll")
	ole32                     = syscall.NewLazyDLL("ole32.dll")
	oleaut32                  = syscall.NewLazyDLL("oleaut32.dll")
	pCreateWindowEx           = user32.NewProc("CreateWindowExW")
	pRegisterClass            = user32.NewProc("RegisterClassExW")
	pDefWindowProc            = user32.NewProc("DefWindowProcW")
	pShowWindow               = user32.NewProc("ShowWindow")
	pUpdateWindow             = user32.NewProc("UpdateWindow")
	pSetWindowText            = user32.NewProc("SetWindowTextW")
	pGetWindowText            = user32.NewProc("GetWindowTextW")
	pGetWindowTextLength      = user32.NewProc("GetWindowTextLengthW")
	pSendMessage              = user32.NewProc("SendMessageW")
	pPostMessage              = user32.NewProc("PostMessageW")
	pGetMessage               = user32.NewProc("GetMessageW")
	pTranslateMessage         = user32.NewProc("TranslateMessage")
	pDispatchMessage          = user32.NewProc("DispatchMessageW")
	pPostQuit                 = user32.NewProc("PostQuitMessage")
	pSetTimer                 = user32.NewProc("SetTimer")
	pGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	pGetWindowRect            = user32.NewProc("GetWindowRect")
	pSetWindowPos             = user32.NewProc("SetWindowPos")
	pGetWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")
	pGetWindowLong            = user32.NewProc("GetWindowLongPtrW")
	pGetModuleHandle          = kernel32.NewProc("GetModuleHandleW")
	pGetCurrentThreadId       = kernel32.NewProc("GetCurrentThreadId")
	pOpenProcess              = kernel32.NewProc("OpenProcess")
	pCloseHandle              = kernel32.NewProc("CloseHandle")
	pQueryProcess             = kernel32.NewProc("QueryFullProcessImageNameW")
	pCreateSnapshot           = kernel32.NewProc("CreateToolhelp32Snapshot")
	pProcessFirst             = kernel32.NewProc("Process32FirstW")
	pProcessNext              = kernel32.NewProc("Process32NextW")
	pSetProcessDPIAware       = user32.NewProc("SetProcessDPIAware")
	pLoadIcon                 = user32.NewProc("LoadIconW")
	pLoadCursor               = user32.NewProc("LoadCursorW")
	pShellNotify              = shell32.NewProc("Shell_NotifyIconW")
	pShellExecute             = shell32.NewProc("ShellExecuteW")
	pCreateFont               = gdi32.NewProc("CreateFontW")
	pDeleteObject             = gdi32.NewProc("DeleteObject")
	pGetSysColorBrush         = user32.NewProc("GetSysColorBrush")
	pRegCreateKey             = advapi32.NewProc("RegCreateKeyExW")
	pRegSetValue              = advapi32.NewProc("RegSetValueExW")
	pRegDeleteValue           = advapi32.NewProc("RegDeleteValueW")
	pRegCloseKey              = advapi32.NewProc("RegCloseKey")
	pGetAsyncKeyState         = user32.NewProc("GetAsyncKeyState")
	pKeybdEvent               = user32.NewProc("keybd_event")
)

func ptr(s string) *uint16                              { return syscall.StringToUTF16Ptr(s) }
func su(s string) uintptr                               { return uintptr(unsafe.Pointer(ptr(s))) }
func call(p *syscall.LazyProc, args ...uintptr) uintptr { v, _, _ := p.Call(args...); return v }
func send(hwnd uintptr, msg uint32, w, l uintptr) uintptr {
	return call(pSendMessage, hwnd, uintptr(msg), w, l)
}
func text(hwnd uintptr, s string) { call(pSetWindowText, hwnd, su(s)) }
func getText(hwnd uintptr) string {
	n := call(pGetWindowTextLength, hwnd)
	if n > 8192 {
		n = 8192
	}
	b := make([]uint16, n+2)
	call(pGetWindowText, hwnd, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
	return syscall.UTF16ToString(b)
}
func fg() uintptr { return call(pGetForegroundWindow) }
func windowPID(hwnd uintptr) uint32 {
	var p uint32
	call(pGetWindowThreadProcessID, hwnd, uintptr(unsafe.Pointer(&p)))
	return p
}
func position(hwnd uintptr) rect {
	var r rect
	call(pGetWindowRect, hwnd, uintptr(unsafe.Pointer(&r)))
	return r
}
func show(hwnd uintptr, v int32)                  { call(pShowWindow, hwnd, uintptr(v)) }
func post(hwnd uintptr, msg uint32, w, l uintptr) { call(pPostMessage, hwnd, uintptr(msg), w, l) }

type point struct{ X, Y int32 }
type rect struct{ Left, Top, Right, Bottom int32 }
type msg struct {
	Hwnd           uintptr
	Message        uint32
	Wparam, Lparam uintptr
	Time           uint32
	Pt             point
	Lprivate       uint32
}
type wndClass struct {
	Size                               uint32
	Style                              uint32
	WndProc                            uintptr
	ClsExtra, WndExtra                 int32
	Instance, Icon, Cursor, Background uintptr
	MenuName, ClassName                *uint16
	IconSm                             uintptr
}

func create(class, title string, exStyle, style uint32, x, y, w, h int32, parent, menu uintptr) uintptr {
	return call(pCreateWindowEx, uintptr(exStyle), su(class), su(title), uintptr(style), uintptr(x), uintptr(y), uintptr(w), uintptr(h), parent, menu, call(pGetModuleHandle, 0), 0)
}
func classRegister(name string, callback uintptr, bg uintptr) {
	wc := wndClass{Size: uint32(unsafe.Sizeof(wndClass{})), WndProc: callback, Instance: call(pGetModuleHandle, 0), Cursor: call(pLoadCursor, 0, 32512), Background: bg, ClassName: ptr(name)}
	call(pRegisterClass, uintptr(unsafe.Pointer(&wc)))
}

const (
	wmCreate           = 0x0001
	wmDestroy          = 0x0002
	wmSize             = 0x0005
	wmClose            = 0x0010
	wmCommand          = 0x0111
	wmTimer            = 0x0113
	wmApp              = 0x8000
	wmSetFont          = 0x0030
	wmUser             = 0x0400
	bmGetCheck         = 0x00f0
	bmSetCheck         = 0x00f1
	lbAddString        = 0x0180
	lbResetContent     = 0x0184
	lbGetCurSel        = 0x0188
	wsVisible          = 0x10000000
	wsChild            = 0x40000000
	wsBorder           = 0x00800000
	wsTabstop          = 0x00010000
	wsVScroll          = 0x00200000
	wsOverlappedWindow = 0x00CF0000
	wsCaption          = 0x00C00000
	wsSysmenu          = 0x00080000
	wsMinimize         = 0x00020000
	wsPopup            = 0x80000000
	ssLeft             = 0
	esAutoHScroll      = 0x0080
	bsPush             = 0
	bsAutoCheck        = 0x3
	lbsNotify          = 0x1
	swHide             = 0
	swShow             = 5
	swShowNA           = 8
	swMinimized        = 2
	swRestore          = 9
	exToolWindow       = 0x00000080
	exTopMost          = 0x00000008
	exNoActivate       = 0x08000000
	swpNoActivate      = 0x0010
	swpShowWindow      = 0x0040
	swpNoZOrder        = 0x0004
	hwndTopMost        = ^uintptr(0) // -1
)

func makeFont(height int32, weight int32) uintptr {
	return call(pCreateFont, uintptr(height), 0, 0, 0, uintptr(weight), 0, 0, 0, 1, 0, 0, 5, 0, su("Segoe UI"))
}
func setFont(hwnd, font uintptr) { send(hwnd, wmSetFont, font, 1) }

func foregroundExe(pid uint32) string {
	h := call(pOpenProcess, 0x1000, 0, uintptr(pid))
	if h == 0 {
		return ""
	}
	defer call(pCloseHandle, h)
	buf := make([]uint16, 32768)
	n := uint32(len(buf))
	if call(pQueryProcess, h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n))) == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:n])
}
