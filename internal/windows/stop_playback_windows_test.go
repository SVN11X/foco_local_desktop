package windows

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unsafe"
)

func TestWindowsSendInputSize(t *testing.T) {
	if got := unsafe.Sizeof(winKeyboardInput{}); got != 40 {
		t.Fatalf("Win64 INPUT must be 40 bytes, got %d", got)
	}
}

func TestBlockedPageContent(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "config.json")
	path, err := blockedPagePath(cfg, "youtube.com")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Lo hiciste bien", "youtube.com", "vídeo y el audio", "FOCOLOCAL", "¿Qué puedes hacer ahora?", "about:blank"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("blocked page missing %q", want)
		}
	}
	if _, err := blockedPagePath(cfg, "../../config.json"); err == nil {
		t.Fatal("path traversal accepted")
	}
}

func TestFileURL(t *testing.T) {
	got := pathToFileURL(`C:\Users\Ale\AppData\Roaming\FocoLocalDesktop\bloqueado-youtube.com.html`)
	if !strings.HasPrefix(got, "file:///C:/Users/Ale/") {
		t.Fatalf("unexpected file URL %q", got)
	}
}

func TestBlockedPageEscapesDomain(t *testing.T) {
	html := blockedPageHTML(`<script>alert(1)</script>`)
	if strings.Contains(html, `<script>alert`) || !strings.Contains(html, `&lt;script&gt;`) {
		t.Fatal("domain markup was not escaped")
	}
}
