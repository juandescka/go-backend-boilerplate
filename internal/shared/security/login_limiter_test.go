package securityevent

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLoginLimiterWindowsCapacityAndCancellation(t *testing.T) {
	ctx := context.Background()
	now := time.Unix(1000, 0)
	l, err := NewLoginLimiter(2, LoginPolicy{2, time.Minute}, LoginPolicy{1, 5 * time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	l.now = func() time.Time { return now }
	allow := func(id bool, key string, want time.Duration) {
		t.Helper()
		var retry time.Duration
		var err error
		if id {
			retry, err = l.AllowIdentifier(ctx, key)
		} else {
			retry, err = l.AllowIP(ctx, key)
		}
		if err != nil || retry != want {
			t.Fatalf("retry=%v error=%v, want %v", retry, err, want)
		}
	}
	allow(false, "192.0.2.1", 0)
	allow(true, "user@example.test", 0)
	allow(false, "192.0.2.1", 0)
	allow(false, "192.0.2.1", time.Minute)
	allow(true, "user@example.test", 5*time.Minute)
	// New keys fail closed, but do not extend occupied records or evict them.
	allow(true, "other@example.test", time.Minute)
	now = now.Add(30 * time.Second)
	allow(false, "192.0.2.2", 30*time.Second)
	now = now.Add(30 * time.Second)
	allow(false, "192.0.2.2", 0)
	allow(false, "192.0.2.2", 0)
	allow(true, "other@example.test", time.Minute)
	if len(l.records) != 2 || len(l.expirations) != 2 {
		t.Fatal("unbounded records")
	}
	now = now.Add(4 * time.Minute)
	allow(false, "192.0.2.2", 0)
	allow(true, "other@example.test", 0)
	if len(l.records) != 2 {
		t.Fatal("capacity did not recover for a full login pair")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := l.AllowIP(canceled, "new IP"); err != context.Canceled {
		t.Fatal("cancellation not respected")
	}
	if _, err := l.AllowIdentifier(canceled, "new identifier"); err != context.Canceled {
		t.Fatal("cancellation not respected")
	}
	if len(l.records) != 2 {
		t.Fatal("canceled attempt changed capacity")
	}
	if _, err := NewLoginLimiter(1, LoginPolicy{1, time.Second}, LoginPolicy{1, time.Second}); err == nil {
		t.Fatal("unusable capacity accepted")
	}
	var missing *LoginLimiter
	if _, err := missing.AllowIP(ctx, "IP"); err == nil {
		t.Fatal("missing limiter failed open")
	}
}

func TestLoginLimiterConcurrentThresholdAndChurn(t *testing.T) {
	l, _ := NewLoginLimiter(100, LoginPolicy{7, time.Minute}, LoginPolicy{3, time.Minute})
	var allowed atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			retry, err := l.AllowIP(context.Background(), "192.0.2.1")
			if err != nil {
				t.Error(err)
			}
			if retry == 0 {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()
	if allowed.Load() != 7 {
		t.Fatalf("admitted %d", allowed.Load())
	}
	for i := 0; i < 1000; i++ {
		_, _ = l.AllowIdentifier(context.Background(), fmt.Sprintf("identifier-%d", i))
	}
	if len(l.records) != 100 || len(l.expirations) != 100 {
		t.Fatal("attacker key churn grew storage")
	}
}

func BenchmarkLoginLimiter(b *testing.B) {
	b.Run("concurrent known IP", func(b *testing.B) {
		l, _ := NewLoginLimiter(100, LoginPolicy{Limit: 1000000000, Window: time.Hour}, LoginPolicy{Limit: 1, Window: time.Hour})
		ctx := context.Background()
		_, _ = l.AllowIP(ctx, "192.0.2.1")
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_, _ = l.AllowIP(ctx, "192.0.2.1")
			}
		})
	})
	b.Run("saturated new keys", func(b *testing.B) {
		l, _ := NewLoginLimiter(10000, LoginPolicy{Limit: 1, Window: time.Hour}, LoginPolicy{Limit: 1, Window: time.Hour})
		ctx := context.Background()
		for i := 0; i < 10000; i++ {
			_, _ = l.AllowIP(ctx, fmt.Sprint(i))
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = l.AllowIdentifier(ctx, "new@example.test")
		}
	})
}
