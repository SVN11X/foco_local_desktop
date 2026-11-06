package limit

import (
	"path/filepath"
	"testing"
	"time"
)

func TestTodayUsageSeparatesWeeklyAndMonthlyBudgets(t *testing.T) {
	s := testStore(t)
	base := time.Date(2026, 11, 5, 9, 0, 0, 0, time.Local)
	if e := s.UpsertRule(Rule{Domain: "youtube.com", Minutes: 30, Period: "week", Enabled: true}); e != nil {
		t.Fatal(e)
	}
	if e := s.UpsertRule(Rule{Domain: "instagram.com", Minutes: 20, Period: "month", Enabled: true}); e != nil {
		t.Fatal(e)
	}
	s.Observe("youtube.com", "edge:y", base)
	s.Observe("youtube.com", "edge:y", base.Add(time.Second))
	s.Observe("", "", base.Add(2*time.Second)) // inactive tab: no time
	s.Observe("instagram.com", "chrome:i", base.Add(3*time.Second))
	s.Observe("instagram.com", "chrome:i", base.Add(4*time.Second))
	total, bySite := s.TodayUsage(base.Add(4 * time.Second))
	if total != 2000 || bySite["youtube.com"] != 1000 || bySite["instagram.com"] != 1000 {
		t.Fatalf("daily total %d breakdown %v", total, bySite)
	}
	_, used := s.Snapshot(base.AddDate(0, 0, 1))
	if used["youtube.com"] != 1000 || used["instagram.com"] != 1000 {
		t.Fatalf("period budgets missing: %v", used)
	}
	total, _ = s.TodayUsage(base.AddDate(0, 0, 1))
	if total != 0 {
		t.Fatalf("today must reset on next day: %d", total)
	}
	if e := s.Save(); e != nil {
		t.Fatal(e)
	}
	reopened, e := Open(s.Path)
	if e != nil {
		t.Fatal(e)
	}
	total, _ = reopened.TodayUsage(base)
	if total != 2000 {
		t.Fatalf("daily data lost across restart: %d", total)
	}
}

func TestThemeIsPersisted(t *testing.T) {
	s := testStore(t)
	if err := s.SetTheme("sepia"); err == nil {
		t.Fatal("invalid theme accepted")
	}
	if err := s.SetTheme("dark"); err != nil {
		t.Fatal(err)
	}
	if e := s.Save(); e != nil {
		t.Fatal(e)
	}
	loaded, e := Open(filepath.Join(filepath.Dir(s.Path), "data.json"))
	if e != nil {
		t.Fatal(e)
	}
	cfg, _ := loaded.Snapshot(time.Now())
	if cfg.Theme != "dark" {
		t.Fatalf("theme missing: %q", cfg.Theme)
	}
}

func TestDailyMigrationFromLegacyDayRule(t *testing.T) {
	s := testStore(t)
	now := time.Now()
	r := Rule{Domain: "example.com", Minutes: 30, Period: "day", Enabled: true}
	if e := s.UpsertRule(r); e != nil {
		t.Fatal(e)
	}
	s.Observe("example.com", "x", now)
	s.Observe("example.com", "x", now.Add(time.Second))
	s.Data.DailyUsage = nil // simulate old config
	if e := s.Save(); e != nil {
		t.Fatal(e)
	}
	restored, e := Open(s.Path)
	if e != nil {
		t.Fatal(e)
	}
	total, _ := restored.TodayUsage(now)
	if total != 1000 {
		t.Fatalf("legacy daily usage not recovered: %d", total)
	}
}
