package catalog

import (
	"focolocal-desktop/internal/limit"
	"testing"
)

func TestPresetDomainsAndMinutes(t *testing.T) {
	seen := map[string]bool{}
	if len(Sites) < 6 {
		t.Fatal("insufficient presets")
	}
	for _, p := range Sites {
		if p.Name == "" || limit.NormalizeRuleTarget(p.Domain) != p.Domain {
			t.Errorf("invalid preset: %+v", p)
		}
		if seen[p.Domain] {
			t.Errorf("duplicate domain: %s", p.Domain)
		}
		if p.Minutes < 1 || p.Minutes > 1440 {
			t.Errorf("invalid daily time: %+v", p)
		}
		seen[p.Domain] = true
	}
}

func TestYouTubePresetsAreIndependent(t *testing.T) {
	var shorts, videos bool
	for _, preset := range Sites {
		if preset.Domain == limit.YouTubeShortsTarget {
			shorts = preset.Name == "YouTube Shorts" && preset.Minutes > 0
		}
		if preset.Domain == "youtube.com" {
			videos = preset.Name == "YouTube · vídeos" && preset.Minutes > 0
		}
	}
	if !shorts || !videos {
		t.Fatalf("YouTube presets missing: shorts=%v videos=%v", shorts, videos)
	}
	if n := len(InCategory(0)); n != 7 {
		t.Fatalf("unexpected count in streaming category: %d", n)
	}
}
