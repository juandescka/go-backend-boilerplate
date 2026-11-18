package securityevent

import (
	"container/heap"
	"context"
	"crypto/sha256"
	"errors"
	"sync"
	"time"
)

type LoginPolicy struct {
	Limit  int
	Window time.Duration
}

type loginRecord struct {
	key     [32]byte
	count   int
	expires time.Time
}

type loginExpirations []*loginRecord

func (q loginExpirations) Len() int           { return len(q) }
func (q loginExpirations) Less(i, j int) bool { return q[i].expires.Before(q[j].expires) }
func (q loginExpirations) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q *loginExpirations) Push(v any)        { *q = append(*q, v.(*loginRecord)) }
func (q *loginExpirations) Pop() any {
	old := *q
	v := old[len(old)-1]
	old[len(old)-1] = nil
	*q = old[:len(old)-1]
	return v
}

// LoginLimiter bounds admitted attempts, including successful logins. Fixed
// windows and no reset on success avoid account-state-dependent throttle behavior.
// Limits are process-local; restarting or adding processes supplies fresh budgets.
type LoginLimiter struct {
	mu             sync.Mutex
	records        map[[32]byte]*loginRecord
	expirations    loginExpirations
	capacity       int
	ip, identifier LoginPolicy
	now            func() time.Time
}

func NewLoginLimiter(capacity int, ip, identifier LoginPolicy) (*LoginLimiter, error) {
	if capacity < 2 || ip.Limit < 1 || identifier.Limit < 1 || ip.Window <= 0 || identifier.Window <= 0 {
		return nil, errors.New("invalid login limiter policy")
	}
	return &LoginLimiter{records: make(map[[32]byte]*loginRecord), capacity: capacity, ip: ip, identifier: identifier, now: time.Now}, nil
}

func (l *LoginLimiter) AllowIP(ctx context.Context, ip string) (time.Duration, error) {
	if l == nil {
		return 0, errors.New("login limiter unavailable")
	}
	return l.allow(ctx, "ip\x00"+ip, l.ip)
}

// AllowIdentifier requires the same database-normalized identifier used by login.
// Only a fixed-length digest is retained; raw identifiers never become map keys.
func (l *LoginLimiter) AllowIdentifier(ctx context.Context, identifier string) (time.Duration, error) {
	if l == nil {
		return 0, errors.New("login limiter unavailable")
	}
	return l.allow(ctx, "identifier\x00"+identifier, l.identifier)
}

func (l *LoginLimiter) allow(ctx context.Context, value string, policy LoginPolicy) (time.Duration, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	key := sha256.Sum256([]byte(value))
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	now := l.now()
	for len(l.expirations) > 0 && !l.expirations[0].expires.After(now) {
		record := heap.Pop(&l.expirations).(*loginRecord)
		delete(l.records, record.key)
	}
	if record := l.records[key]; record != nil {
		if record.count >= policy.Limit {
			return record.expires.Sub(now), nil
		}
		record.count++
		return 0, nil
	}
	// Never evict live limits: churn must not reopen an attacker's budget.
	// Exhaustion rejects only new keys until the earliest expiration, without
	// extending that deadline or preventing known keys using remaining budgets.
	if len(l.records) >= l.capacity {
		return l.expirations[0].expires.Sub(now), nil
	}
	record := &loginRecord{key: key, count: 1, expires: now.Add(policy.Window)}
	l.records[key] = record
	heap.Push(&l.expirations, record)
	return 0, nil
}
