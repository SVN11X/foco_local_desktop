package limit

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// A lease is a temporary block that a trusted local owner (for example Guardian) asks for.
//
// Safety rules, all enforced here and covered by tests:
//   - Leases live only in memory. They are never written to config.json and never survive a restart.
//   - A lease can only cover a site that the user already has as an enabled rule. An owner cannot
//     introduce a block on a site the user never listed. Disabling that rule stops the lease at once.
//   - Every lease has an owner, a reason and an expiry of at most MaxLeaseDuration.
//   - Pausing FocoLocal pauses leases too: the user's own control always wins.
//   - The owner can release its leases at any time (emergency release) and expired leases vanish.
//   - Nothing here changes how the user's own rules count time or block.
const (
	MaxLeaseDuration  = 60 * time.Minute
	MaxLeasesPerOwner = 20
	maxLeaseReason    = 120
	maxLeaseKey       = 64
)

var ownerPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,39}$`)

var (
	// ErrLeaseInvalid means the request itself is wrong. The message says what.
	ErrLeaseInvalid = errors.New("lease inválido")
	// ErrLeaseTarget means the site is not among the user's enabled rules.
	ErrLeaseTarget = errors.New("el sitio no está entre las reglas activas del usuario")
	// ErrLeaseLimit means the owner already holds MaxLeasesPerOwner leases.
	ErrLeaseLimit = errors.New("demasiados leases activos para este propietario")
)

// Lease is one active temporary block.
type Lease struct {
	ID             string
	Owner          string
	Domain         string
	Reason         string
	IdempotencyKey string
	ExpiresAt      time.Time
}

// LeaseRequest is what an owner sends to ApplyLease.
type LeaseRequest struct {
	Owner          string
	Domain         string
	Reason         string
	IdempotencyKey string
	ExpiresAt      time.Time
}

func invalidLease(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrLeaseInvalid, fmt.Sprintf(format, args...))
}

// ApplyLease creates a lease, or returns the existing one when the same owner repeats the same
// idempotency key for the same site. created is false for a repeat. Repeating a key for another
// site is an error, so a retry can never silently block something else.
func (s *Store) ApplyLease(req LeaseRequest, now time.Time) (lease Lease, created bool, err error) {
	owner := strings.TrimSpace(req.Owner)
	reason := strings.TrimSpace(req.Reason)
	key := strings.TrimSpace(req.IdempotencyKey)
	switch {
	case !ownerPattern.MatchString(owner):
		return Lease{}, false, invalidLease("el propietario debe ser minúsculas, números, punto, guion o guion bajo (máx. 40)")
	case reason == "" || len(reason) > maxLeaseReason:
		return Lease{}, false, invalidLease("el motivo es obligatorio y admite hasta %d caracteres", maxLeaseReason)
	case key == "" || len(key) > maxLeaseKey:
		return Lease{}, false, invalidLease("la clave de idempotencia es obligatoria y admite hasta %d caracteres", maxLeaseKey)
	case !req.ExpiresAt.After(now):
		return Lease{}, false, invalidLease("la expiración debe ser futura")
	case req.ExpiresAt.After(now.Add(MaxLeaseDuration)):
		return Lease{}, false, invalidLease("la duración máxima es %d minutos", int(MaxLeaseDuration/time.Minute))
	}
	domain := NormalizeRuleTarget(req.Domain)
	if domain == "" {
		return Lease{}, false, invalidLease("el sitio no es válido")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLeasesLocked(now)
	for _, l := range s.leases {
		if l.Owner == owner && l.IdempotencyKey == key {
			if l.Domain != domain {
				return Lease{}, false, invalidLease("la clave ya se usó con otro sitio")
			}
			return l, false, nil
		}
	}
	if !s.ruleCoversLocked(domain) {
		return Lease{}, false, ErrLeaseTarget
	}
	held := 0
	for _, l := range s.leases {
		if l.Owner == owner {
			held++
		}
	}
	if held >= MaxLeasesPerOwner {
		return Lease{}, false, ErrLeaseLimit
	}
	id, err := newLeaseID()
	if err != nil {
		return Lease{}, false, err
	}
	lease = Lease{ID: id, Owner: owner, Domain: domain, Reason: reason, IdempotencyKey: key, ExpiresAt: req.ExpiresAt}
	s.leases = append(s.leases, lease)
	return lease, true, nil
}

// ReleaseLease removes one lease. Only its owner can. It reports whether something was removed.
func (s *Store) ReleaseLease(owner, id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, l := range s.leases {
		if l.ID == id && l.Owner == owner {
			s.leases = append(s.leases[:i], s.leases[i+1:]...)
			return true
		}
	}
	return false
}

// ReleaseOwnerLeases is the emergency release: it removes every lease of one owner.
func (s *Store) ReleaseOwnerLeases(owner string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.leases[:0]
	removed := 0
	for _, l := range s.leases {
		if l.Owner == owner {
			removed++
			continue
		}
		kept = append(kept, l)
	}
	s.leases = kept
	return removed
}

// ActiveLeases returns a copy of the unexpired leases of one owner, soonest expiry first.
func (s *Store) ActiveLeases(owner string, now time.Time) []Lease {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLeasesLocked(now)
	var out []Lease
	for _, l := range s.leases {
		if l.Owner == owner {
			out = append(out, l)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].ExpiresAt.Equal(out[j].ExpiresAt) {
			return out[i].ExpiresAt.Before(out[j].ExpiresAt)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// leaseBlockLocked returns the leased site that blocks host, or "". The caller holds s.mu.
func (s *Store) leaseBlockLocked(host string, now time.Time) string {
	if len(s.leases) == 0 {
		return ""
	}
	s.pruneLeasesLocked(now)
	for _, l := range s.leases {
		if Matches(host, l.Domain) && s.ruleCoversLocked(l.Domain) {
			return l.Domain
		}
	}
	return ""
}

// ruleCoversLocked reports whether an enabled user rule covers the site: the rule is for the same
// site or for a parent domain. A lease can never be wider than what the user listed.
func (s *Store) ruleCoversLocked(domain string) bool {
	for _, r := range s.Data.Rules {
		if r.Enabled && Matches(domain, r.Domain) {
			return true
		}
	}
	return false
}

func (s *Store) pruneLeasesLocked(now time.Time) {
	kept := s.leases[:0]
	for _, l := range s.leases {
		if l.ExpiresAt.After(now) {
			kept = append(kept, l)
		}
	}
	s.leases = kept
}

func newLeaseID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("no se pudo crear el identificador: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}
