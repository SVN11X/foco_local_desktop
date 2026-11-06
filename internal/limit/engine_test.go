package limit

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNormalizeDomain(t *testing.T) {
	for input, want := range map[string]string{"https://www.youtube.com/watch?v=2": "www.youtube.com", "*.youtube.com": "youtube.com", "youtube.com": "youtube.com", "localhost": "", "https://bad domain.com": "", "127.0.0.1": ""} {
		if got := NormalizeDomain(input); got != want {
			t.Errorf("%s: got %q want %q", input, got, want)
		}
	}
}
func TestTrackingAcrossWindows(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "db.json"))
	_ = s.Add("youtube.com", 1)
	now := time.Date(2026, 11, 6, 8, 0, 0, 0, time.Local)
	s.Observe("www.youtube.com", "Chrome:1:youtube", now)
	for i := 1; i <= 30; i++ {
		s.Observe("www.youtube.com", "Chrome:1:youtube", now.Add(time.Duration(i)*time.Second))
	}
	s.Observe("www.youtube.com", "Edge:2:youtube", now.Add(31*time.Second)) // first Edge sample no charge
	for i := 32; i <= 62; i++ {
		s.Observe("www.youtube.com", "Edge:2:youtube", now.Add(time.Duration(i)*time.Second))
	}
	blocked, spent, limit := s.Observe("www.youtube.com", "Edge:2:youtube", now.Add(63*time.Second))
	if blocked != "youtube.com" || spent < 60000 || limit != 60000 {
		t.Fatalf("not shared: %q %d/%d", blocked, spent, limit)
	}
	if e := s.Save(); e != nil {
		t.Fatal(e)
	}
	reloaded, e := Open(s.Path)
	if e != nil {
		t.Fatal(e)
	}
	_, usage := reloaded.Snapshot(now)
	if usage["youtube.com"] < 60000 {
		t.Fatal("didn't persist")
	}
}
func TestPauseAndHostRules(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "db.json"))
	_ = s.Add("example.com", 1)
	_ = s.Add("sub.example.com", 2)
	now := time.Now()
	s.Observe("sub.example.com", "a", now)
	s.Observe("sub.example.com", "a", now.Add(time.Second))
	_, u := s.Snapshot(now)
	if u["sub.example.com"] != 1000 || u["example.com"] != 0 {
		t.Fatalf("rule select %v", u)
	}
	s.SetPause(true)
	s.Observe("sub.example.com", "a", now.Add(2*time.Second))
	_, u = s.Snapshot(now)
	if u["sub.example.com"] != 1000 {
		t.Fatalf("pause %v", u)
	}
}
