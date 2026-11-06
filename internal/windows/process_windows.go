package windows

import (
	"path/filepath"
	"strings"
	"unicode/utf16"
	"unsafe"
)

var browserNames = map[string]bool{
	"chrome.exe": true, "msedge.exe": true, "firefox.exe": true, "brave.exe": true,
	"opera.exe": true, "opera_gx.exe": true, "vivaldi.exe": true, "chromium.exe": true,
	"waterfox.exe": true, "librewolf.exe": true, "zen.exe": true, "floorp.exe": true,
	"thorium.exe": true, "arc.exe": true,
}

func browserName(exe string) string {
	name := strings.ToLower(filepath.Base(exe))
	if browserNames[name] {
		return name
	}
	return ""
}

type procEntry struct {
	Size, Usage, PID             uint32
	Heap                         uintptr
	ModuleID, Threads, ParentPID uint32
	BasePriority                 int32
	Flags                        uint32
	Exe                          [260]uint16
}
type procInfo struct {
	parent uint32
	name   string
}

func processTree() map[uint32]procInfo {
	m := make(map[uint32]procInfo)
	h := call(pCreateSnapshot, 2, 0)
	if h == 0 || h == ^uintptr(0) {
		return m
	}
	defer call(pCloseHandle, h)
	var p procEntry
	p.Size = uint32(unsafe.Sizeof(p))
	if call(pProcessFirst, h, uintptr(unsafe.Pointer(&p))) == 0 {
		return m
	}
	for {
		var chars []uint16
		for _, r := range p.Exe {
			if r == 0 {
				break
			}
			chars = append(chars, r)
		}
		m[p.PID] = procInfo{p.ParentPID, strings.ToLower(string(utf16.Decode(chars)))}
		p.Size = uint32(unsafe.Sizeof(p))
		if call(pProcessNext, h, uintptr(unsafe.Pointer(&p))) == 0 {
			break
		}
	}
	return m
}
func isAutomated(pid uint32) bool {
	procs := processTree()
	prefixes := map[string]bool{"python.exe": true, "pythonw.exe": true, "node.exe": true, "chromedriver.exe": true, "geckodriver.exe": true, "msedgedriver.exe": true, "playwright.exe": true, "pytest.exe": true, "java.exe": false}
	// Only exclude based on *ancestors*; the browser process itself must still be identified.
	for depth := 0; depth < 9; depth++ {
		p, exists := procs[pid]
		if !exists || p.parent == 0 || p.parent == pid {
			break
		}
		pid = p.parent
		ancestor, ok := procs[pid]
		if !ok {
			break
		}
		if prefixes[ancestor.name] {
			return true
		}
	}
	return false
}
