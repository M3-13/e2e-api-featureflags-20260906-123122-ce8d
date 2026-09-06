package store

import (
	"errors"
	"sync"
	"testing"
)

func TestCreateAndGet(t *testing.T) {
	s := NewStore()
	f := Flag{Key: "myflag", Enabled: true, Description: "d", RolloutPercent: 50}

	if err := s.Create(f); err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}

	got, ok := s.Get("myflag")
	if !ok {
		t.Fatalf("expected flag to exist")
	}
	if got != f {
		t.Fatalf("expected %+v, got %+v", f, got)
	}
}

func TestCreateDuplicate(t *testing.T) {
	s := NewStore()
	if err := s.Create(Flag{Key: "dup"}); err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}
	if err := s.Create(Flag{Key: "dup"}); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}
}

func TestList(t *testing.T) {
	s := NewStore()
	s.Create(Flag{Key: "a"})
	s.Create(Flag{Key: "b"})

	flags := s.List()
	if len(flags) != 2 {
		t.Fatalf("expected 2 flags, got %d", len(flags))
	}
}

func TestGetMissing(t *testing.T) {
	s := NewStore()
	if _, ok := s.Get("missing"); ok {
		t.Fatalf("expected missing flag to not exist")
	}
}

func TestUpdate(t *testing.T) {
	s := NewStore()
	s.Create(Flag{Key: "k", Enabled: false})

	updated, ok := s.Update("k", Flag{Enabled: true, Description: "new", RolloutPercent: 100})
	if !ok {
		t.Fatalf("expected update to succeed")
	}
	if updated.Key != "k" || !updated.Enabled || updated.Description != "new" || updated.RolloutPercent != 100 {
		t.Fatalf("unexpected updated flag: %+v", updated)
	}

	got, _ := s.Get("k")
	if !got.Enabled {
		t.Fatalf("expected stored flag to reflect update")
	}
}

func TestUpdateMissing(t *testing.T) {
	s := NewStore()
	if _, ok := s.Update("missing", Flag{}); ok {
		t.Fatalf("expected update of missing flag to fail")
	}
}

func TestDelete(t *testing.T) {
	s := NewStore()
	s.Create(Flag{Key: "k"})

	if !s.Delete("k") {
		t.Fatalf("expected delete to succeed")
	}
	if _, ok := s.Get("k"); ok {
		t.Fatalf("expected flag to be gone after delete")
	}
}

func TestDeleteMissing(t *testing.T) {
	s := NewStore()
	if s.Delete("missing") {
		t.Fatalf("expected delete of missing flag to fail")
	}
}

func TestConcurrentAccess(t *testing.T) {
	s := NewStore()
	const n = 100

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := string(rune('a'+i%26)) + string(rune('0'+i%10))
			s.Create(Flag{Key: key})
			s.Get(key)
			s.List()
			s.Update(key, Flag{Enabled: true})
			s.Delete(key)
		}(i)
	}
	wg.Wait()
}
