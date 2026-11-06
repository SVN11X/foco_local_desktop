package windows

import "testing"

func TestEmbeddedBrandIcons(t *testing.T) {
	for _, site := range []string{"youtube.com", "www.youtube.com", "instagram.com", "www.instagram.com", "github.com", "linkedin.com", "tiktok.com"} {
		img, ok := brandAsset(site)
		if !ok || img == nil {
			t.Errorf("missing embedded mark for %s", site)
		}
	}
	if _, ok := brandAsset("unknown.example.com"); ok {
		t.Fatal("unknown brand must use neutral fallback")
	}
}
