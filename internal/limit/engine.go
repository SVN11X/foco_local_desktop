package limit

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Rule matches a hostname and every subdomain; rules share one global budget across browsers.
type Rule struct {
	Domain  string `json:"domain"`
	Minutes int    `json:"minutes"`
	Enabled bool   `json:"enabled"`
}

type Config struct {
	Rules          []Rule                      `json:"rules"`
	Usage          map[string]map[string]int64 `json:"usage_ms"` // YYYY-MM-DD -> domain -> spent milliseconds
	AutoStart      bool                        `json:"auto_start"`
	SafeAutomation bool                        `json:"safe_automation"`
	Paused         bool                        `json:"paused"`
	ExcludedPIDs   []uint32                    `json:"excluded_pids,omitempty"` // ephemeral, not saved
}

type Store struct {
	mu         sync.Mutex
	Path       string
	Data       Config
	lastSample string
	lastTick   time.Time
	dirty      bool
}

func Open(path string) (*Store, error) {
	s := &Store{Path: path, Data: Config{Usage: make(map[string]map[string]int64), SafeAutomation: true}}
	b, err := os.ReadFile(path)
	if err == nil {
		if e := json.Unmarshal(b, &s.Data); e != nil {
			return nil, fmt.Errorf("config inválida: %w", e)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if s.Data.Usage == nil {
		s.Data.Usage = make(map[string]map[string]int64)
	}
	s.Data.ExcludedPIDs = nil // never persist PID exemptions across reboots
	for i := range s.Data.Rules {
		s.Data.Rules[i].Domain = NormalizeDomain(s.Data.Rules[i].Domain)
	}
	return s, nil
}

var domainRE = regexp.MustCompile(`(?i)^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

func NormalizeDomain(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	raw = strings.TrimPrefix(raw, "*.")
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, e := url.Parse(raw)
	if e != nil {
		return ""
	}
	h := strings.TrimSuffix(u.Hostname(), ".")
	if net.ParseIP(h) != nil || !domainRE.MatchString(h) {
		return ""
	}
	return h
}

func HostFromAddress(address string) string {
	address = strings.TrimSpace(address)
	if address == "" || strings.ContainsAny(address, " \n\r\t") {
		return ""
	}
	if !strings.Contains(address, "://") {
		address = "https://" + address
	}
	u, e := url.Parse(address)
	if e != nil {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	return NormalizeDomain(u.Hostname())
}

func Matches(host, domain string) bool { return host == domain || strings.HasSuffix(host, "."+domain) }
func dateKey(t time.Time) string       { return t.Local().Format("2006-01-02") }

func (s *Store) Add(raw string, minutes int) error {
	d := NormalizeDomain(raw)
	if d == "" {
		return errors.New("Escribe un dominio válido: youtube.com")
	}
	if minutes < 1 || minutes > 1440 {
		return errors.New("El límite debe estar entre 1 y 1440 minutos")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.Data.Rules {
		if s.Data.Rules[i].Domain == d {
			s.Data.Rules[i].Minutes = minutes
			s.Data.Rules[i].Enabled = true
			s.dirty = true
			return nil
		}
	}
	s.Data.Rules = append(s.Data.Rules, Rule{d, minutes, true})
	s.dirty = true
	return nil
}
func (s *Store) Remove(domain string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rules := s.Data.Rules[:0]
	for _, r := range s.Data.Rules {
		if r.Domain != domain {
			rules = append(rules, r)
		}
	}
	s.Data.Rules = rules
	s.dirty = true
}
func (s *Store) SetPause(pause bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data.Paused = pause
	s.lastTick = time.Time{}
	s.dirty = true
}
func (s *Store) SetSafeAutomation(safe bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data.SafeAutomation = safe
	s.dirty = true
}
func (s *Store) SetAutoStart(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data.AutoStart = enabled
	s.dirty = true
}
func (s *Store) ExcludePID(pid uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.Data.ExcludedPIDs {
		if p == pid {
			return
		}
	}
	s.Data.ExcludedPIDs = append(s.Data.ExcludedPIDs, pid)
}
func (s *Store) IsExcluded(pid uint32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.Data.ExcludedPIDs {
		if p == pid {
			return true
		}
	}
	return false
}
func (s *Store) Snapshot(t time.Time) (Config, map[string]int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg := s.Data
	cfg.Rules = append([]Rule(nil), s.Data.Rules...)
	cfg.ExcludedPIDs = append([]uint32(nil), s.Data.ExcludedPIDs...)
	used := make(map[string]int64)
	for k, v := range s.Data.Usage[dateKey(t)] {
		used[k] = v
	}
	return cfg, used
}

// Observe charges elapsed time to one matching rule only while a recognized browser is foreground.
// Long suspensions are capped to avoid counting idle time while the computer sleeps.
// More specific domain rules win. Input sampleID includes HWND:PID:hostname to prevent carryover.
func (s *Store) Observe(host, sampleID string, now time.Time) (blocked string, usedMS int64, limitMS int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dt := time.Duration(0)
	if !s.lastTick.IsZero() && s.lastSample == sampleID && sampleID != "" {
		dt = now.Sub(s.lastTick)
	}
	if dt < 0 {
		dt = 0
	}
	if dt > 2*time.Second {
		dt = 0
	}
	s.lastTick = now
	s.lastSample = sampleID
	if s.Data.Paused || host == "" {
		return "", 0, 0
	}
	idx := -1
	for i, r := range s.Data.Rules {
		if r.Enabled && Matches(host, r.Domain) && (idx < 0 || len(r.Domain) > len(s.Data.Rules[idx].Domain)) {
			idx = i
		}
	}
	if idx < 0 {
		return "", 0, 0
	}
	r := s.Data.Rules[idx]
	day := dateKey(now)
	if s.Data.Usage[day] == nil {
		s.Data.Usage[day] = map[string]int64{}
	}
	if dt > 0 {
		s.Data.Usage[day][r.Domain] += dt.Milliseconds()
		s.dirty = true
	}
	used := s.Data.Usage[day][r.Domain]
	cap := int64(r.Minutes) * 60 * 1000
	if used >= cap {
		return r.Domain, used, cap
	}
	return "", used, cap
}
func (s *Store) ResetTimer(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	day := dateKey(now)
	delete(s.Data.Usage, day)
	s.lastTick = time.Time{}
	s.dirty = true
}
func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.dirty {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0700); err != nil {
		return err
	}
	// Do not let usage grow without bound: keep a rolling 30-day window.
	for day := range s.Data.Usage {
		d, e := time.ParseInLocation("2006-01-02", day, time.Local)
		if e != nil || time.Since(d) > 31*24*time.Hour {
			delete(s.Data.Usage, day)
		}
	}
	copy := s.Data
	copy.ExcludedPIDs = nil
	b, e := json.MarshalIndent(copy, "", "  ")
	if e != nil {
		return e
	}
	tmp := s.Path + ".tmp"
	if e = os.WriteFile(tmp, b, 0600); e != nil {
		return e
	}
	if e = os.Rename(tmp, s.Path); e != nil {
		// On Windows Go os.Rename replaces regular files when allowed, but antivirus or
		// ACL restrictions can prevent replacement. Never delete the last good config.
		_ = os.Remove(tmp)
		return fmt.Errorf("no se pudo reemplazar la configuración en %s: %w", runtime.GOOS, e)
	}
	s.dirty = false
	return nil
}
