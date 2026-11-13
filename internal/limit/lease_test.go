package limit

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

var leaseNow = time.Date(2026, 11, 13, 10, 0, 0, 0, time.Local)

// leaseStore returns a store whose user rules are the given domains, all enabled with a day budget.
func leaseStore(t *testing.T, domains ...string) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range domains {
		if err := s.Add(d, 30); err != nil {
			t.Fatal(err)
		}
	}
	return s, path
}

func request(domain, key string, minutes int) LeaseRequest {
	return LeaseRequest{
		Owner: "guardian", Domain: domain, Reason: "pending quiz", IdempotencyKey: key,
		ExpiresAt: leaseNow.Add(time.Duration(minutes) * time.Minute),
	}
}

func TestLeaseBlocksWithoutChargingAndExpires(t *testing.T) {
	s, _ := leaseStore(t, "youtube.com")
	lease, created, err := s.ApplyLease(request("youtube.com", "k1", 15), leaseNow)
	if err != nil || !created {
		t.Fatalf("apply: created=%v err=%v", created, err)
	}
	if lease.ID == "" || lease.Owner != "guardian" {
		t.Fatalf("lease fields: %+v", lease)
	}
	blocked, used, limit := s.Observe("youtube.com", "a", leaseNow)
	if blocked != "youtube.com" || used != 0 || limit != 0 {
		t.Fatalf("while leased: %q %d %d", blocked, used, limit)
	}
	// A subdomain of the leased site is blocked too, an unrelated site is not.
	if blocked, _, _ := s.Observe("m.youtube.com", "a", leaseNow.Add(time.Second)); blocked != "youtube.com" {
		t.Fatalf("subdomain should be blocked, got %q", blocked)
	}
	if blocked, _, _ := s.Observe("example.org", "a", leaseNow.Add(time.Second)); blocked != "" {
		t.Fatalf("unrelated site blocked: %q", blocked)
	}
	// Blocked time is not charged to the user's budget.
	if _, spent := s.Snapshot(leaseNow); spent["youtube.com"] != 0 {
		t.Fatalf("lease charged the budget: %d", spent["youtube.com"])
	}
	// After the expiry the user's own rule decides again and the lease is gone.
	after := leaseNow.Add(16 * time.Minute)
	if blocked, _, _ := s.Observe("youtube.com", "b", after); blocked != "" {
		t.Fatalf("expired lease still blocks: %q", blocked)
	}
	if got := s.ActiveLeases("guardian", after); len(got) != 0 {
		t.Fatalf("expired lease listed: %+v", got)
	}
}

func TestLeaseIsIdempotentAndRejectsKeyReuseForAnotherSite(t *testing.T) {
	s, _ := leaseStore(t, "youtube.com", "reddit.com")
	first, created, err := s.ApplyLease(request("youtube.com", "same-key", 10), leaseNow)
	if err != nil || !created {
		t.Fatal(err)
	}
	again, created, err := s.ApplyLease(request("youtube.com", "same-key", 30), leaseNow)
	if err != nil || created || again.ID != first.ID {
		t.Fatalf("repeat must return the same lease: %+v created=%v err=%v", again, created, err)
	}
	if !again.ExpiresAt.Equal(first.ExpiresAt) {
		t.Fatal("a repeat must not extend the lease")
	}
	_, _, err = s.ApplyLease(request("reddit.com", "same-key", 10), leaseNow)
	if !errors.Is(err, ErrLeaseInvalid) {
		t.Fatalf("key reuse for another site: %v", err)
	}
	if n := len(s.ActiveLeases("guardian", leaseNow)); n != 1 {
		t.Fatalf("active leases: %d", n)
	}
}

func TestLeaseOnlyCoversSitesTheUserAlreadyListed(t *testing.T) {
	s, _ := leaseStore(t, "www.youtube.com")
	for _, domain := range []string{"github.com", "docs.google.com", "youtube.com"} {
		_, _, err := s.ApplyLease(request(domain, "k-"+domain, 10), leaseNow)
		if !errors.Is(err, ErrLeaseTarget) {
			t.Fatalf("%s must be refused (not listed or wider than the rule), got %v", domain, err)
		}
	}
	// A narrower site under a listed domain is allowed.
	if _, _, err := s.ApplyLease(request("music.www.youtube.com", "ok", 10), leaseNow); err != nil {
		t.Fatalf("subdomain of a listed rule: %v", err)
	}
}

func TestDisablingTheUserRuleStopsTheLeaseImmediately(t *testing.T) {
	s, _ := leaseStore(t, "youtube.com")
	if _, _, err := s.ApplyLease(request("youtube.com", "k", 30), leaseNow); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertRule(Rule{Domain: "youtube.com", Minutes: 30, Enabled: false, Period: "day"}); err != nil {
		t.Fatal(err)
	}
	if blocked, _, _ := s.Observe("youtube.com", "a", leaseNow); blocked != "" {
		t.Fatalf("lease kept blocking after the user disabled the rule: %q", blocked)
	}
	// And a new lease on a disabled rule is refused.
	if _, _, err := s.ApplyLease(request("youtube.com", "k2", 10), leaseNow); !errors.Is(err, ErrLeaseTarget) {
		t.Fatalf("lease on a disabled rule: %v", err)
	}
}

func TestPausingFocoLocalPausesLeases(t *testing.T) {
	s, _ := leaseStore(t, "youtube.com")
	if _, _, err := s.ApplyLease(request("youtube.com", "k", 30), leaseNow); err != nil {
		t.Fatal(err)
	}
	s.SetPause(true)
	if blocked, _, _ := s.Observe("youtube.com", "a", leaseNow); blocked != "" {
		t.Fatalf("a paused FocoLocal must not block: %q", blocked)
	}
	s.SetPause(false)
	if blocked, _, _ := s.Observe("youtube.com", "a", leaseNow.Add(time.Second)); blocked != "youtube.com" {
		t.Fatalf("lease should resume after the pause: %q", blocked)
	}
}

func TestLeasesAreNeverWrittenToDisk(t *testing.T) {
	s, path := leaseStore(t, "youtube.com")
	lease, _, err := s.ApplyLease(request("youtube.com", "secret-key", 30), leaseNow)
	if err != nil {
		t.Fatal(err)
	}
	s.Observe("youtube.com", "a", leaseNow)
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{"lease", "guardian", "secret-key", lease.ID, "pending quiz"} {
		if strings.Contains(strings.ToLower(string(raw)), strings.ToLower(needle)) {
			t.Fatalf("config.json leaks %q", needle)
		}
	}
	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if blocked, _, _ := reopened.Observe("youtube.com", "a", leaseNow); blocked != "" {
		t.Fatalf("a lease survived a restart: %q", blocked)
	}
}

func TestReleaseRulesAndEmergencyRelease(t *testing.T) {
	s, _ := leaseStore(t, "youtube.com", "reddit.com")
	a, _, _ := s.ApplyLease(request("youtube.com", "a", 30), leaseNow)
	if _, _, err := s.ApplyLease(request("reddit.com", "b", 30), leaseNow); err != nil {
		t.Fatal(err)
	}
	other := request("youtube.com", "c", 30)
	other.Owner = "someone-else"
	if _, _, err := s.ApplyLease(other, leaseNow); err != nil {
		t.Fatal(err)
	}
	if s.ReleaseLease("someone-else", a.ID) {
		t.Fatal("another owner released a lease that is not theirs")
	}
	if !s.ReleaseLease("guardian", a.ID) || s.ReleaseLease("guardian", a.ID) {
		t.Fatal("release must work once for the owner")
	}
	if n := s.ReleaseOwnerLeases("guardian"); n != 1 {
		t.Fatalf("emergency release removed %d, want 1", n)
	}
	if n := len(s.ActiveLeases("someone-else", leaseNow)); n != 1 {
		t.Fatalf("emergency release must not touch other owners: %d", n)
	}
	if blocked, _, _ := s.Observe("reddit.com", "a", leaseNow); blocked != "" {
		t.Fatalf("reddit still blocked after emergency release: %q", blocked)
	}
}

func TestLeaseRequestValidation(t *testing.T) {
	s, _ := leaseStore(t, "youtube.com")
	bad := map[string]LeaseRequest{
		"empty owner":     {Domain: "youtube.com", Reason: "r", IdempotencyKey: "k", ExpiresAt: leaseNow.Add(time.Minute)},
		"upper owner":     {Owner: "Guardian", Domain: "youtube.com", Reason: "r", IdempotencyKey: "k", ExpiresAt: leaseNow.Add(time.Minute)},
		"no reason":       {Owner: "guardian", Domain: "youtube.com", IdempotencyKey: "k", ExpiresAt: leaseNow.Add(time.Minute)},
		"long reason":     {Owner: "guardian", Domain: "youtube.com", Reason: strings.Repeat("x", 121), IdempotencyKey: "k", ExpiresAt: leaseNow.Add(time.Minute)},
		"no key":          {Owner: "guardian", Domain: "youtube.com", Reason: "r", ExpiresAt: leaseNow.Add(time.Minute)},
		"past expiry":     {Owner: "guardian", Domain: "youtube.com", Reason: "r", IdempotencyKey: "k", ExpiresAt: leaseNow},
		"too long":        {Owner: "guardian", Domain: "youtube.com", Reason: "r", IdempotencyKey: "k", ExpiresAt: leaseNow.Add(MaxLeaseDuration + time.Second)},
		"bad domain":      {Owner: "guardian", Domain: "", Reason: "r", IdempotencyKey: "k", ExpiresAt: leaseNow.Add(time.Minute)},
		"long key":        {Owner: "guardian", Domain: "youtube.com", Reason: "r", IdempotencyKey: strings.Repeat("k", 65), ExpiresAt: leaseNow.Add(time.Minute)},
		"path-like owner": {Owner: "../x", Domain: "youtube.com", Reason: "r", IdempotencyKey: "k", ExpiresAt: leaseNow.Add(time.Minute)},
	}
	for name, req := range bad {
		if _, _, err := s.ApplyLease(req, leaseNow); !errors.Is(err, ErrLeaseInvalid) {
			t.Errorf("%s: want ErrLeaseInvalid, got %v", name, err)
		}
	}
	// The maximum duration itself is allowed.
	if _, _, err := s.ApplyLease(request("youtube.com", "max", 60), leaseNow); err != nil {
		t.Fatalf("exactly the maximum duration: %v", err)
	}
}

func TestLeaseLimitPerOwner(t *testing.T) {
	var domains []string
	for i := 0; i < MaxLeasesPerOwner+1; i++ {
		domains = append(domains, fmt.Sprintf("site%d.example", i))
	}
	s, _ := leaseStore(t, domains...)
	for i, d := range domains[:MaxLeasesPerOwner] {
		if _, _, err := s.ApplyLease(request(d, fmt.Sprintf("k%d", i), 10), leaseNow); err != nil {
			t.Fatalf("lease %d: %v", i, err)
		}
	}
	_, _, err := s.ApplyLease(request(domains[MaxLeasesPerOwner], "overflow", 10), leaseNow)
	if !errors.Is(err, ErrLeaseLimit) {
		t.Fatalf("want ErrLeaseLimit, got %v", err)
	}
	// Expired leases free their slot.
	later := leaseNow.Add(11 * time.Minute)
	req := request(domains[MaxLeasesPerOwner], "after-expiry", 10)
	req.ExpiresAt = later.Add(10 * time.Minute)
	if _, _, err := s.ApplyLease(req, later); err != nil {
		t.Fatalf("after expiry: %v", err)
	}
}

func TestYouTubeShortsLeaseDoesNotBlockNormalYouTube(t *testing.T) {
	s, _ := leaseStore(t, YouTubeShortsTarget, "youtube.com")
	if _, _, err := s.ApplyLease(request(YouTubeShortsTarget, "shorts", 20), leaseNow); err != nil {
		t.Fatal(err)
	}
	if blocked, _, _ := s.Observe(YouTubeShortsTarget, "a", leaseNow); blocked != YouTubeShortsTarget {
		t.Fatalf("shorts should be blocked: %q", blocked)
	}
	if blocked, _, _ := s.Observe("youtube.com", "a", leaseNow.Add(time.Second)); blocked != "" {
		t.Fatalf("regular YouTube must stay open: %q", blocked)
	}
}

func TestLeasesAreSafeForConcurrentUse(t *testing.T) {
	s, _ := leaseStore(t, "youtube.com")
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				key := fmt.Sprintf("k%d-%d", i, j%5)
				s.ApplyLease(request("youtube.com", key, 10), leaseNow)
				s.Observe("youtube.com", "a", leaseNow)
				s.ActiveLeases("guardian", leaseNow)
				if j%7 == 0 {
					s.ReleaseOwnerLeases("guardian")
				}
			}
		}(i)
	}
	wg.Wait()
}

func TestUsersOwnRulesAreUnchangedWithoutLeases(t *testing.T) {
	s, _ := leaseStore(t, "youtube.com")
	for i := 0; i < 3; i++ {
		s.Observe("youtube.com", "a", leaseNow.Add(time.Duration(i)*time.Second))
	}
	if _, spent := s.Snapshot(leaseNow); spent["youtube.com"] != 2000 {
		t.Fatalf("without leases the budget must still count: %d ms", spent["youtube.com"])
	}
}
