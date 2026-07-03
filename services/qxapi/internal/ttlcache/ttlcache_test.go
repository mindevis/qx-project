package ttlcache

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestCacheGetSetAndExpiry(t *testing.T) {
	cache := New[string](time.Millisecond*50, 10)
	cache.Set("key", "value")

	got, ok := cache.Get("key")
	if !ok || got != "value" {
		t.Fatalf("got %q ok=%v", got, ok)
	}

	time.Sleep(time.Millisecond * 60)
	if _, ok := cache.Get("key"); ok {
		t.Fatal("expected expired entry to miss")
	}
}

func TestCacheGetOrLoadDedupesConcurrentLoads(t *testing.T) {
	cache := New[int](time.Second, 10)
	var calls int
	var mu sync.Mutex
	load := func() (int, error) {
		mu.Lock()
		calls++
		mu.Unlock()
		time.Sleep(time.Millisecond * 20)
		return 42, nil
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		value, err := cache.GetOrLoad("key", load)
		if err != nil || value != 42 {
			t.Errorf("first load: value=%d err=%v", value, err)
		}
	}()
	go func() {
		defer wg.Done()
		value, err := cache.GetOrLoad("key", load)
		if err != nil || value != 42 {
			t.Errorf("second load: value=%d err=%v", value, err)
		}
	}()
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Fatalf("expected one upstream load, got %d", calls)
	}
}

func TestCacheGetOrLoadPropagatesError(t *testing.T) {
	cache := New[string](time.Second, 10)
	errBoom := errors.New("boom")
	_, err := cache.GetOrLoad("key", func() (string, error) {
		return "", errBoom
	})
	if !errors.Is(err, errBoom) {
		t.Fatalf("expected boom, got %v", err)
	}
}
