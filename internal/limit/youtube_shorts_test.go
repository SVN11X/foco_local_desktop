package limit

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestYouTubeAddressRouting(t *testing.T) {
	cases := []struct{ addr, host, site string }{
		{"https://www.youtube.com/shorts/ABC123_def0?feature=share", "www.youtube.com", YouTubeShortsTarget},
		{"youtube.com/shorts/ABC123_def0", "youtube.com", YouTubeShortsTarget},
		{"https://m.youtube.com/shorts/ABC123_def0", "m.youtube.com", YouTubeShortsTarget},
		{"https://www.youtube.com/shorts", "www.youtube.com", YouTubeShortsTarget},
		{"https://www.youtube.com/watch?v=ABC123_def0", "www.youtube.com", "www.youtube.com"},
		{"https://www.youtube.com/", "www.youtube.com", "www.youtube.com"},
		{"https://www.youtube.com/shortsfeed", "www.youtube.com", "www.youtube.com"},
		{"https://other.example.com/shorts/ABC123_def0", "other.example.com", "other.example.com"},
		{"file:///C:/Users/test/file.html", "", ""},
	}
	for _, tc := range cases {
		if got := HostFromAddress(tc.addr); got != tc.host {
			t.Errorf("HostFromAddress(%q)=%q want %q", tc.addr, got, tc.host)
		}
		if got := SiteFromAddress(tc.addr); got != tc.site {
			t.Errorf("SiteFromAddress(%q)=%q want %q", tc.addr, got, tc.site)
		}
	}
	for input, want := range map[string]string{
		"youtube.com/shorts":                        YouTubeShortsTarget,
		"https://www.youtube.com/shorts/Zzi_abc-11": YouTubeShortsTarget,
		"m.youtube.com/shorts/ID":                   YouTubeShortsTarget,
		"youtube.com":                               "youtube.com",
		"https://youtube.com/watch?v=2":             "youtube.com",
		"https://example.org/about":                 "example.org",
	} {
		if got := NormalizeRuleTarget(input); got != want {
			t.Errorf("NormalizeRuleTarget(%q)=%q want %q", input, got, want)
		}
	}
}

func TestYouTubeRoutesDoNotOverlap(t *testing.T) {
	for _, tc := range []struct {
		site, domain string
		want         bool
	}{
		{YouTubeShortsTarget, "youtube.com", false},
		{YouTubeShortsTarget, "www.youtube.com", false},
		{YouTubeShortsTarget, YouTubeShortsTarget, true},
		{"www.youtube.com", YouTubeShortsTarget, false},
		{"m.youtube.com", "youtube.com", true},
		{"www.youtube.com", "youtube.com", true},
	} {
		if got := Matches(tc.site, tc.domain); got != tc.want {
			t.Errorf("Matches(%q,%q)=%v want %v", tc.site, tc.domain, got, tc.want)
		}
	}
}

func TestIndependentShortsAndVideosBudgetsAcrossBrowsersAndRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Add("youtube.com", 1); err != nil {
		t.Fatal(err)
	}
	if err := store.Add(YouTubeShortsTarget, 2); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 11, 6, 14, 0, 0, 0, time.Local)
	video := SiteFromAddress("https://www.youtube.com/watch?v=ABC123_def0")
	short := SiteFromAddress("https://www.youtube.com/shorts/ABC123_def0")
	store.Observe(video, "edge:window:watch", start)
	store.Observe(video, "edge:window:watch", start.Add(time.Second))
	// Switch to Shorts without switching browser window. First sample of
	// the new route must not charge time for the old page.
	store.Observe(short, "edge:window:shorts", start.Add(2*time.Second))
	store.Observe(short, "edge:window:shorts", start.Add(3*time.Second))
	// Switching browsers must share the same Shorts budget without charging
	// the transition interval or the previous browser's background activity.
	store.Observe(short, "chrome:window:shorts", start.Add(4*time.Second))
	store.Observe(short, "chrome:window:shorts", start.Add(5*time.Second))
	_, usage := store.Snapshot(start)
	if usage["youtube.com"] != 1000 || usage[YouTubeShortsTarget] != 2000 {
		t.Fatalf("overlapping budgets: %v", usage)
	}
	total, byRule := store.TodayUsage(start)
	if total != 3000 || byRule[YouTubeShortsTarget] != 2000 {
		t.Fatalf("overlapping daily totals: %d %v", total, byRule)
	}
	if err := store.Save(); err != nil {
		t.Fatal(err)
	}
	restored, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	_, usage = restored.Snapshot(start)
	if usage["youtube.com"] != 1000 || usage[YouTubeShortsTarget] != 2000 {
		t.Fatalf("lost counters after restart: %v", usage)
	}
}

func TestExistingYoutubeRuleDoesNotChargeShortsAndRemainsIntact(t *testing.T) {
	store, _ := Open(filepath.Join(t.TempDir(), "config.json"))
	_ = store.Add("www.youtube.com", 1)
	when := time.Date(2026, 11, 6, 14, 0, 0, 0, time.Local)
	for i := 0; i < 5; i++ {
		store.Observe(SiteFromAddress("https://www.youtube.com/shorts/abc"), "chrome:shorts", when.Add(time.Duration(i)*time.Second))
	}
	_, used := store.Snapshot(when)
	if used["www.youtube.com"] != 0 {
		t.Fatalf("old YouTube rule should exclude Shorts: %v", used)
	}
	cfg, _ := store.Snapshot(when)
	if len(cfg.Rules) != 1 || cfg.Rules[0].Domain != "www.youtube.com" {
		t.Fatalf("legacy rule changed: %+v", cfg.Rules)
	}
	if strings.Contains(RuleLabel(YouTubeShortsTarget), "/") {
		t.Fatal("unfriendly Shorts title")
	}
}
