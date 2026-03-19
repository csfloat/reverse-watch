package leader

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"reverse-watch/domain/models/constants"
	"reverse-watch/internal/testutil"
	"reverse-watch/repository/factory"
	"reverse-watch/secret"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func isLockHeld(t *testing.T, db *gorm.DB, lockKey uint32) bool {
	t.Helper()

	var count int64
	if err := db.Raw("SELECT COUNT(*) FROM pg_locks WHERE locktype = 'advisory' AND granted = true AND objid = $1", lockKey).Scan(&count).Error; err != nil {
		t.Fatalf("failed to check if lock is held: %v", err)
	}
	return count > 0
}

func TestElector_TryAdvisoryLock_AcquiresLock(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
	f, err := factory.NewFactoryWithConfig(&factory.Config{
		PrivateDB: db,
		PublicDB:  db,
		KeyGen:    keygen,
	})
	if err != nil {
		t.Fatalf("NewFactoryWithConfig(): %v", err)
	}
	defer f.Close()

	log := zap.NewNop().Sugar()
	e := New(f, log).(*elector)

	ctx := context.Background()
	tx, acquired := e.tryAdvisoryLock(ctx, 12345)
	if !acquired {
		t.Fatal("expected to acquire lock")
	}

	if !isLockHeld(t, db, 12345) {
		t.Fatal("expected lock to be held")
	}

	tx.Rollback()
}

func TestElector_TryAdvisoryLock_FailsWhenAlreadyHeld(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
	f, err := factory.NewFactoryWithConfig(&factory.Config{
		PrivateDB: db,
		PublicDB:  db,
		KeyGen:    keygen,
	})
	if err != nil {
		t.Fatalf("NewFactoryWithConfig(): %v", err)
	}
	defer f.Close()

	log := zap.NewNop().Sugar()
	e := New(f, log).(*elector)

	ctx := context.Background()
	lockKey := uint32(99999)

	// First elector acquires the lock
	tx1, acquired1 := e.tryAdvisoryLock(ctx, lockKey)
	if !acquired1 {
		t.Fatal("first elector should acquire lock")
	}
	defer tx1.Rollback()

	if !isLockHeld(t, db, lockKey) {
		t.Fatal("expected lock to be held")
	}

	// Second attempt should fail (same lock key, different transaction)
	tx2, acquired2 := e.tryAdvisoryLock(ctx, lockKey)
	if acquired2 {
		t.Fatal("second elector should NOT acquire lock")
	}
	if tx2 != nil {
		t.Fatal("transaction should be nil when lock not acquired")
	}
}

func TestElector_TryAdvisoryLock_ReleasesOnRollback(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
	f, err := factory.NewFactoryWithConfig(&factory.Config{
		PrivateDB: db,
		PublicDB:  db,
		KeyGen:    keygen,
	})
	if err != nil {
		t.Fatalf("NewFactoryWithConfig(): %v", err)
	}
	defer f.Close()

	log := zap.NewNop().Sugar()
	e := New(f, log).(*elector)

	ctx := context.Background()
	lockKey := uint32(77777)

	// Acquire and release
	tx1, acquired1 := e.tryAdvisoryLock(ctx, lockKey)
	if !acquired1 {
		t.Fatal("should acquire lock")
	}
	tx1.Rollback()

	if isLockHeld(t, db, lockKey) {
		t.Fatal("expected lock to be released")
	}

	// Should be able to acquire again after release
	tx2, acquired2 := e.tryAdvisoryLock(ctx, lockKey)
	if !acquired2 {
		t.Fatal("should acquire lock after previous holder released")
	}

	if !isLockHeld(t, db, lockKey) {
		t.Fatal("expected lock to be held")
	}
	tx2.Rollback()
}

func TestElector_TryAdvisoryLock_ContextCancellation(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
	f, err := factory.NewFactoryWithConfig(&factory.Config{
		PrivateDB: db,
		PublicDB:  db,
		KeyGen:    keygen,
	})
	if err != nil {
		t.Fatalf("NewFactoryWithConfig(): %v", err)
	}
	defer f.Close()

	log := zap.NewNop().Sugar()
	e := New(f, log).(*elector)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, acquired := e.tryAdvisoryLock(ctx, 66666)
	if acquired {
		t.Fatal("should not acquire lock with canceled context")
	}
}

func TestElector_Run_ExitsOnContextCancel(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
	f, err := factory.NewFactoryWithConfig(&factory.Config{
		PrivateDB: db,
		PublicDB:  db,
		KeyGen:    keygen,
	})
	if err != nil {
		t.Fatalf("NewFactoryWithConfig(): %v", err)
	}
	defer f.Close()

	log := zap.NewNop().Sugar()
	e := New(f, log)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		e.Run(ctx, 33333, time.Hour, func(ctx context.Context) {
			cancel()
			t.Fatal("onWork should not be called")
		})
	}()

	// Cancel immediately
	cancel()

	select {
	case <-done:
		// Success - Run exited
	case <-time.After(2 * time.Second):
		t.Fatal("Run should exit when context is canceled")
	}
}

func TestElector_Run_ExecutesOnWorkWhenLockAcquired(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
	f, err := factory.NewFactoryWithConfig(&factory.Config{
		PrivateDB: db,
		PublicDB:  db,
		KeyGen:    keygen,
	})
	if err != nil {
		t.Fatalf("NewFactoryWithConfig(): %v", err)
	}
	defer f.Close()

	log := zap.NewNop().Sugar()
	e := New(f, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var workCount atomic.Int32
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		e.Run(ctx, 44444, 10*time.Millisecond, func(ctx context.Context) {
			workCount.Add(1)
			cancel() // Stop after first execution
		})
	}()

	wg.Wait()

	if workCount.Load() != 1 {
		t.Fatalf("expected onWork to be called once, got %d", workCount.Load())
	}
}

func TestElector_Run_OnlyOneLeaderExecutesWork(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
	f, err := factory.NewFactoryWithConfig(&factory.Config{
		PrivateDB: db,
		PublicDB:  db,
		KeyGen:    keygen,
	})
	if err != nil {
		t.Fatalf("NewFactoryWithConfig(): %v", err)
	}
	defer f.Close()

	log := zap.NewNop().Sugar()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	lockKey := uint32(55555)
	period := 50 * time.Millisecond

	var concurrentWorkers atomic.Int32
	var maxConcurrent atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e := New(f, log)
			e.Run(ctx, lockKey, period, func(ctx context.Context) {
				// Increment concurrent count
				current := concurrentWorkers.Add(1)

				// Track max seen
				for {
					max := maxConcurrent.Load()
					if current <= max || maxConcurrent.CompareAndSwap(max, current) {
						break
					}
				}

				time.Sleep(30 * time.Millisecond) // Hold lock during work
				concurrentWorkers.Add(-1)
			})
		}()
	}

	wg.Wait()

	if maxConcurrent.Load() > 1 {
		t.Fatalf("expected max 1 concurrent worker, got %d", maxConcurrent.Load())
	}
}
