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
	"strconv"
	"strings"
	"sync"
	"time"
)

// Rule matches a hostname and every subdomain; rules share one global budget across browsers.
type Rule struct {
	Domain         string `json:"domain"`
	Minutes        int    `json:"minutes"`
	Enabled        bool   `json:"enabled"`
	Period         string `json:"period,omitempty"` // day, week, month, blocked, schedule, interval
	From           string `json:"from,omitempty"`
	To             string `json:"to,omitempty"`
	Weekdays       uint8  `json:"weekdays,omitempty"` // 0 = all, Sun bit 0 ... Sat bit 6
	Months         uint16 `json:"months,omitempty"`   // 0 = all, Jan bit 0 ... Dec bit 11
	SessionMinutes int    `json:"session_minutes,omitempty"`
	BreakMinutes   int    `json:"break_minutes,omitempty"`
}

type IntervalState struct {
	UsedMS  int64 `json:"used_ms"`
	UntilMS int64 `json:"until_ms"`
}

type Config struct {
	Rules              []Rule                              `json:"rules"`
	Usage              map[string]map[string]int64         `json:"usage_ms"`
	DailyUsage         map[string]map[string]int64         `json:"daily_usage_ms,omitempty"`
	Theme              string                              `json:"theme,omitempty"` // system, light, dark
	DailyTrackingSince string                              `json:"daily_tracking_since,omitempty"`
	Intervals          map[string]map[string]IntervalState `json:"intervals,omitempty"` // YYYY-MM-DD -> domain -> spent milliseconds
	AutoStart          bool                                `json:"auto_start"`
	SafeAutomation     bool                                `json:"safe_automation"`
	Paused             bool                                `json:"paused"`
	BlockMode          string                              `json:"block_mode,omitempty"`    // navigate (stop playback) or overlay (visual only)
	ExcludedPIDs       []uint32                            `json:"excluded_pids,omitempty"` // ephemeral, not saved
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
	if s.Data.Intervals == nil {
		s.Data.Intervals = make(map[string]map[string]IntervalState)
	}
	if s.Data.DailyUsage == nil {
		s.Data.DailyUsage = make(map[string]map[string]int64)
		// When migrating from v0.3.0, daily rules can be safely recovered.
		// Weekly/monthly totals cannot be allocated backwards to individual days.
		for _, r := range s.Data.Rules {
			if r.Period == "week" || r.Period == "month" {
				if s.Data.Usage[periodKey(r, time.Now())][r.Domain] > 0 {
					s.Data.DailyTrackingSince = time.Now().Format(time.RFC3339)
					break
				}
			}
		}
		for _, r := range s.Data.Rules {
			if r.Period == "" || r.Period == "day" || r.Period == "schedule" || r.Period == "interval" {
				day := dateKey(time.Now())
				if s.Data.Usage[day][r.Domain] > 0 {
					if s.Data.DailyUsage[day] == nil {
						s.Data.DailyUsage[day] = make(map[string]int64)
					}
					s.Data.DailyUsage[day][r.Domain] = s.Data.Usage[day][r.Domain]
				}
			}
		}
		s.dirty = true
	}
	if s.Data.Theme != "system" && s.Data.Theme != "light" && s.Data.Theme != "dark" {
		s.Data.Theme = "system"
		s.dirty = true
	}
	// Existing v0.2.1 configurations opt into stopping playback by default.
	// The user can switch back to the less intrusive overlay mode in the hub.
	if s.Data.BlockMode == "" {
		s.Data.BlockMode = "navigate"
		s.dirty = true
	}
	if s.Data.BlockMode != "navigate" && s.Data.BlockMode != "overlay" {
		s.Data.BlockMode = "overlay" // unknown settings fail conservatively
		s.dirty = true
	}
	s.Data.ExcludedPIDs = nil // never persist PID exemptions across reboots
	for i := range s.Data.Rules {
		s.Data.Rules[i].Domain = NormalizeDomain(s.Data.Rules[i].Domain)
		if s.Data.Rules[i].Period == "" {
			s.Data.Rules[i].Period = "day"
		}
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

func periodKey(r Rule, now time.Time) string {
	switch r.Period {
	case "week":
		y, w := now.Local().ISOWeek()
		return fmt.Sprintf("week-%04d-W%02d", y, w)
	case "month":
		return now.Local().Format("month-2006-01")
	default:
		return dateKey(now)
	}
}

func minutesMax(r Rule) int {
	switch r.Period {
	case "week":
		return 7 * 24 * 60
	case "month":
		return 31 * 24 * 60
	default:
		return 24 * 60
	}
}

func validClock(v string) bool {
	if len(v) != 5 || v[2] != ':' {
		return false
	}
	h, e1 := strconv.Atoi(v[:2])
	m, e2 := strconv.Atoi(v[3:])
	return e1 == nil && e2 == nil && h >= 0 && h <= 23 && m >= 0 && m <= 59
}

func ValidateRule(r Rule) (Rule, error) {
	r.Domain = NormalizeDomain(r.Domain)
	if r.Domain == "" {
		return r, errors.New("Ingresa un dominio válido (por ejemplo youtube.com)")
	}
	if r.Period == "" {
		r.Period = "day"
	}
	switch r.Period {
	case "day", "week", "month", "blocked", "schedule", "interval":
	default:
		return r, errors.New("Selecciona un tipo de límite válido")
	}
	if r.Period != "blocked" && (r.Minutes < 1 || r.Minutes > minutesMax(r)) {
		return r, fmt.Errorf("El límite debe estar entre 1 y %d minutos", minutesMax(r))
	}
	if r.Period == "schedule" && (!validClock(r.From) || !validClock(r.To) || r.From == r.To) {
		return r, errors.New("Horario inválido: usa HH:MM, por ejemplo 09:00 y 18:00")
	}
	if r.Period == "interval" && (r.SessionMinutes < 1 || r.SessionMinutes > 240 || r.BreakMinutes < 1 || r.BreakMinutes > 240) {
		return r, errors.New("Configura intervalos de 1 a 240 minutos de uso y descanso")
	}
	if r.Months > 4095 {
		return r, errors.New("Meses inválidos")
	}
	return r, nil
}

func (s *Store) UpsertRule(rule Rule) error {
	r, e := ValidateRule(rule)
	if e != nil {
		return e
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.Data.Rules {
		if s.Data.Rules[i].Domain == r.Domain {
			s.Data.Rules[i] = r
			s.dirty = true
			return nil
		}
	}
	s.Data.Rules = append(s.Data.Rules, r)
	s.dirty = true
	return nil
}
func (s *Store) Add(raw string, minutes int) error {
	return s.UpsertRule(Rule{Domain: raw, Minutes: minutes, Enabled: true, Period: "day"})
}

func withinSchedule(r Rule, now time.Time) bool {
	local := now.Local()
	if r.Months != 0 && r.Months&(1<<uint(local.Month()-1)) == 0 {
		return false
	}
	day := uint(local.Weekday())
	mins := local.Hour()*60 + local.Minute()
	if r.Period != "schedule" {
		return r.Weekdays == 0 || r.Weekdays&(1<<day) != 0
	}
	start, _ := strconv.Atoi(r.From[:2])
	sm, _ := strconv.Atoi(r.From[3:])
	from := start*60 + sm
	end, _ := strconv.Atoi(r.To[:2])
	em, _ := strconv.Atoi(r.To[3:])
	to := end*60 + em
	if from < to {
		return mins >= from && mins < to && (r.Weekdays == 0 || r.Weekdays&(1<<day) != 0)
	}
	if mins >= from {
		return r.Weekdays == 0 || r.Weekdays&(1<<day) != 0
	}
	if mins < to {
		previous := (day + 6) % 7
		return r.Weekdays == 0 || r.Weekdays&(1<<previous) != 0
	}
	return false
}

// Evaluate a rule before charging; schedule restrictions are independent of time budget.
func (r Rule) Description() string {
	switch r.Period {
	case "week":
		return "Semanal"
	case "month":
		return "Mensual"
	case "blocked":
		return "Bloqueo permanente"
	case "schedule":
		return "Horario " + r.From + "–" + r.To
	case "interval":
		return fmt.Sprintf("Ciclos %dm / %dm pausa", r.SessionMinutes, r.BreakMinutes)
	default:
		return "Diario"
	}
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
func (s *Store) SetBlockMode(mode string) error {
	if mode != "navigate" && mode != "overlay" {
		return errors.New("modo de bloqueo inválido")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data.BlockMode = mode
	s.dirty = true
	return nil
}
func (s *Store) SetSafeAutomation(safe bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data.SafeAutomation = safe
	s.dirty = true
}
func (s *Store) SetTheme(mode string) error {
	if mode != "system" && mode != "light" && mode != "dark" {
		return fmt.Errorf("tema inválido: %s", mode)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data.Theme = mode
	s.dirty = true
	return nil
}

// TodayUsage is a separate, non-overlapping tally. A rule's weekly or monthly
// usage is NOT the same as time spent today. Only Observe adds to this tally.
func (s *Store) TodayUsage(now time.Time) (int64, map[string]int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]int64)
	var total int64
	for domain, ms := range s.Data.DailyUsage[dateKey(now)] {
		if ms < 0 {
			continue
		}
		out[domain] = ms
		total += ms
	}
	return total, out
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
	for _, r := range cfg.Rules {
		used[r.Domain] = s.Data.Usage[periodKey(r, t)][r.Domain]
	}
	return cfg, used
}

// Observe charges only consecutive samples from the intended, eligible browser window.
// A schedule outside its time range blocks without debiting the period budget.
// All rules retain budgets across application restarts and browser changes.
func (s *Store) Observe(host, sampleID string, now time.Time) (blocked string, usedMS int64, limitMS int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dt := time.Duration(0)
	if !s.lastTick.IsZero() && s.lastSample == sampleID && sampleID != "" {
		dt = now.Sub(s.lastTick)
	}
	if dt < 0 || dt > 2*time.Second {
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
	key := periodKey(r, now)
	if s.Data.Usage[key] == nil {
		s.Data.Usage[key] = make(map[string]int64)
	}
	spent := s.Data.Usage[key][r.Domain]
	cap := int64(r.Minutes) * 60000
	if r.Period == "blocked" {
		if withinSchedule(r, now) {
			return r.Domain, spent, 0
		}
		return "", spent, 0
	}
	if !withinSchedule(r, now) {
		return r.Domain, spent, cap
	}
	if spent >= cap {
		return r.Domain, spent, cap
	}
	previousSpent := spent
	if r.Period == "interval" {
		day := dateKey(now)
		if s.Data.Intervals == nil {
			s.Data.Intervals = make(map[string]map[string]IntervalState)
		}
		if s.Data.Intervals[day] == nil {
			s.Data.Intervals[day] = make(map[string]IntervalState)
		}
		state := s.Data.Intervals[day][r.Domain]
		if state.UntilMS > now.UnixMilli() {
			return r.Domain, spent, cap
		}
		if state.UntilMS > 0 && state.UntilMS <= now.UnixMilli() {
			state = IntervalState{}
		}
		if dt > 0 {
			delta := dt.Milliseconds()
			spent += delta
			state.UsedMS += delta
			s.dirty = true
		}
		if state.UsedMS >= int64(r.SessionMinutes)*60000 {
			state.UntilMS = now.Add(time.Duration(r.BreakMinutes) * time.Minute).UnixMilli()
			s.Data.Intervals[day][r.Domain] = state
			s.Data.Usage[key][r.Domain] = spent
			if spent > previousSpent {
				if s.Data.DailyUsage[day] == nil {
					s.Data.DailyUsage[day] = make(map[string]int64)
				}
				s.Data.DailyUsage[day][r.Domain] += spent - previousSpent
			}
			return r.Domain, spent, cap
		}
		s.Data.Intervals[day][r.Domain] = state
	} else if dt > 0 {
		spent += dt.Milliseconds()
		s.dirty = true
	}
	s.Data.Usage[key][r.Domain] = spent
	if spent > previousSpent {
		day := dateKey(now)
		if s.Data.DailyUsage == nil {
			s.Data.DailyUsage = make(map[string]map[string]int64)
		}
		if s.Data.DailyUsage[day] == nil {
			s.Data.DailyUsage[day] = make(map[string]int64)
		}
		s.Data.DailyUsage[day][r.Domain] += spent - previousSpent
	}
	if spent >= cap {
		return r.Domain, spent, cap
	}
	return "", spent, cap
}
func (s *Store) ResetTimer(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	day := dateKey(now)
	for _, r := range s.Data.Rules {
		key := periodKey(r, now)
		delete(s.Data.Usage[key], r.Domain)
		delete(s.Data.Intervals[day], r.Domain)
		delete(s.Data.DailyUsage[day], r.Domain)
	}
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
	// Keep a bounded local history. Do not drop active weekly/monthly budgets.
	for day := range s.Data.Usage {
		switch {
		case strings.HasPrefix(day, "month-"):
			d, e := time.ParseInLocation("month-2006-01", day, time.Local)
			if e != nil || time.Since(d) > 420*24*time.Hour {
				delete(s.Data.Usage, day)
			}
		case strings.HasPrefix(day, "week-"):
			var year, week int
			n, e := fmt.Sscanf(day, "week-%d-W%d", &year, &week)
			if e != nil || n != 2 || year < time.Now().Year()-2 || week < 1 || week > 53 {
				delete(s.Data.Usage, day)
			}
		default:
			d, e := time.ParseInLocation("2006-01-02", day, time.Local)
			if e != nil || time.Since(d) > 366*24*time.Hour {
				delete(s.Data.Usage, day)
			}
		}
	}
	for day := range s.Data.DailyUsage {
		d, e := time.ParseInLocation("2006-01-02", day, time.Local)
		if e != nil || time.Since(d) > 366*24*time.Hour {
			delete(s.Data.DailyUsage, day)
		}
	}
	for day := range s.Data.Intervals {
		d, e := time.ParseInLocation("2006-01-02", day, time.Local)
		if e != nil || time.Since(d) > 31*24*time.Hour {
			delete(s.Data.Intervals, day)
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
