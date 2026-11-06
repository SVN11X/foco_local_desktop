package limit

import (
	"path/filepath"
	"testing"
	"time"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	s, e := Open(filepath.Join(t.TempDir(), "data.json"))
	if e != nil {
		t.Fatal(e)
	}
	return s
}

func TestPresetsDoNotChangeExistingConfig(t *testing.T) {
	s := testStore(t)
	if e := s.Add("youtube.com", 2); e != nil {
		t.Fatal(e)
	}
	cfg, _ := s.Snapshot(time.Now())
	if cfg.Rules[0].Period != "day" {
		t.Fatal("old rules must remain daily")
	}
}
func TestPeriodKeysAndMonthlyPersistence(t *testing.T) {
	s := testStore(t)
	if e := s.UpsertRule(Rule{Domain: "example.com", Minutes: 120, Enabled: true, Period: "month"}); e != nil {
		t.Fatal(e)
	}
	a := time.Date(2026, 11, 2, 10, 0, 0, 0, time.Local)
	s.Observe("example.com", "browser", a)
	s.Observe("example.com", "browser", a.Add(time.Second))
	_, u := s.Snapshot(a.AddDate(0, 0, 10))
	if u["example.com"] != 1000 {
		t.Fatalf("monthly budget disappeared: %v", u)
	}
	_, u = s.Snapshot(a.AddDate(0, 1, 0))
	if u["example.com"] != 0 {
		t.Fatalf("monthly rollover failed %v", u)
	}
	if e := s.Save(); e != nil {
		t.Fatal(e)
	}
	loaded, e := Open(s.Path)
	if e != nil {
		t.Fatal(e)
	}
	_, u = loaded.Snapshot(a)
	if u["example.com"] != 1000 {
		t.Fatalf("monthly persistence %v", u)
	}
}
func TestScheduleAndWeekdays(t *testing.T) {
	r := Rule{Domain: "example.com", Period: "schedule", Minutes: 30, Enabled: true, From: "09:00", To: "17:00", Weekdays: 0b0111110}
	s := testStore(t)
	if e := s.UpsertRule(r); e != nil {
		t.Fatal(e)
	}
	monday := time.Date(2026, 11, 2, 8, 59, 0, 0, time.Local)
	if b, _, _ := s.Observe("example.com", "x", monday); b == "" {
		t.Fatal("outside hours should block")
	}
	if b, _, _ := s.Observe("example.com", "x", monday.Add(time.Minute)); b != "" {
		t.Fatal("inside hours should allow")
	}
	if b, _, _ := s.Observe("example.com", "x", monday.Add(9*time.Hour)); b == "" {
		t.Fatal("17:59 should block")
	}
	saturday := monday.AddDate(0, 0, 5).Add(time.Hour)
	if b, _, _ := s.Observe("example.com", "x", saturday); b == "" {
		t.Fatal("weekends should block")
	}
}
func TestBreakIntervals(t *testing.T) {
	s := testStore(t)
	if e := s.UpsertRule(Rule{Domain: "example.com", Enabled: true, Period: "interval", Minutes: 10, SessionMinutes: 1, BreakMinutes: 2}); e != nil {
		t.Fatal(e)
	}
	start := time.Date(2026, 11, 6, 8, 0, 0, 0, time.Local)
	s.Observe("example.com", "x", start)
	var b string
	for i := 1; i <= 60; i++ {
		b, _, _ = s.Observe("example.com", "x", start.Add(time.Duration(i)*time.Second))
	}
	if b != "example.com" {
		t.Fatal("session cap did not begin rest")
	}
	b, _, _ = s.Observe("example.com", "x", start.Add(61*time.Second))
	if b == "" {
		t.Fatal("should remain blocked during rest")
	}
	b, _, _ = s.Observe("example.com", "x", start.Add(3*time.Minute+2*time.Second))
	if b != "" {
		t.Fatal("after rest interval should reopen")
	}
}
func TestValidation(t *testing.T) {
	for _, r := range []Rule{
		{Domain: "bad host", Minutes: 20, Period: "day"},
		{Domain: "example.com", Minutes: 0, Period: "day"},
		{Domain: "example.com", Minutes: 30, Period: "schedule", From: "8:00", To: "18:00"},
		{Domain: "example.com", Minutes: 30, Period: "interval", SessionMinutes: 0, BreakMinutes: 5},
	} {
		if _, err := ValidateRule(r); err == nil {
			t.Errorf("accepted invalid rule %+v", r)
		}
	}
}
func TestResetCurrentPeriod(t *testing.T) {
	s := testStore(t)
	_ = s.UpsertRule(Rule{Domain: "example.com", Minutes: 300, Enabled: true, Period: "week"})
	now := time.Date(2026, 11, 6, 9, 0, 0, 0, time.Local)
	s.Observe("example.com", "x", now)
	s.Observe("example.com", "x", now.Add(time.Second))
	s.ResetTimer(now)
	_, u := s.Snapshot(now)
	if u["example.com"] != 0 {
		t.Fatal("reset must clear weekly period")
	}
}
