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
		if p.Name == "" || limit.NormalizeDomain(p.Domain) != p.Domain {
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
