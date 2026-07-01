package qlab

import "testing"

func TestDeriveRegistryGenesisOnlyIsEmpty(t *testing.T) {
	c := newTestChain(t) // genesis only
	r, err := DeriveRegistry(c)
	if err != nil {
		t.Fatalf("DeriveRegistry: %v", err)
	}
	if got := len(r.All()); got != 0 {
		t.Fatalf("expected 0 entries from genesis-only chain, got %d", got)
	}
}

// TestDeriveRegistryFullLifecycle replays submit -> harden -> reopen and checks
// that level 5 ends reopened and level 6 is opened.
func TestDeriveRegistryFullLifecycle(t *testing.T) {
	c := newTestChain(t)
	sub := Submission{Solution: "36", CircuitHash: "sha256:abc", VerifiedAt: "2026-01-01T00:00:00Z"}
	_, _ = c.Append(Event{Type: EventSubmit, Level: 5, Submission: &sub, Timestamp: "2026-01-01T00:00:00Z"})
	_, _ = c.Append(Event{Type: EventHarden, Level: 5, Timestamp: "2026-01-02T00:00:00Z"})
	_, _ = c.Append(Event{Type: EventReopen, Level: 5, Timestamp: "2026-01-03T00:00:00Z"})

	r, err := DeriveRegistry(c)
	if err != nil {
		t.Fatalf("DeriveRegistry: %v", err)
	}
	e5, _ := r.Entry(5)
	if e5.State != StateReopened {
		t.Fatalf("level 5 state = %s, want reopened", e5.State)
	}
	if e5.Submission == nil || e5.Submission.Solution != "36" {
		t.Fatalf("submission not derived: %+v", e5.Submission)
	}
	e6, _ := r.Entry(6)
	if e6.State != StateOpen {
		t.Fatalf("level 6 state = %s, want open", e6.State)
	}
}

// TestDeriveRegistryRejectsInvalidEvent: hardening a level that was never broken
// (or opened) must fail derivation.
func TestDeriveRegistryRejectsInvalidEvent(t *testing.T) {
	c := newTestChain(t)
	// Harden without any prior submit: level 5 is open -> hardened is not a valid edge.
	_, _ = c.Append(Event{Type: EventHarden, Level: 5, Timestamp: "t"})
	if _, err := DeriveRegistry(c); err == nil {
		t.Fatal("expected DeriveRegistry to reject an invalid harden event")
	}
}

// TestDeriveRegistryRejectsDoubleSubmit: a second submit on a broken level is
// invalid and must fail derivation.
func TestDeriveRegistryRejectsDoubleSubmit(t *testing.T) {
	c := newTestChain(t)
	sub := Submission{Solution: "36", CircuitHash: "sha256:abc", VerifiedAt: "t"}
	_, _ = c.Append(Event{Type: EventSubmit, Level: 5, Submission: &sub, Timestamp: "t"})
	_, _ = c.Append(Event{Type: EventSubmit, Level: 5, Submission: &sub, Timestamp: "t"})
	if _, err := DeriveRegistry(c); err == nil {
		t.Fatal("expected DeriveRegistry to reject a double submit")
	}
}

// TestDeriveRegistryRoundTrip: saving and reloading the chain yields the same
// derived state.
func TestDeriveRegistryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/chain.json"
	c1 := NewChain(path)
	_ = c1.Load()
	sub := Submission{Solution: "36", CircuitHash: "sha256:abc", VerifiedAt: "t"}
	_, _ = c1.Append(Event{Type: EventSubmit, Level: 5, Submission: &sub, Timestamp: "t"})
	_ = c1.Save()

	c2 := NewChain(path)
	_ = c2.Load()
	r1, _ := DeriveRegistry(c1)
	r2, err := DeriveRegistry(c2)
	if err != nil {
		t.Fatalf("reload DeriveRegistry: %v", err)
	}
	e1, _ := r1.Entry(5)
	e2, _ := r2.Entry(5)
	if e1.State != e2.State || e1.Submission.Solution != e2.Submission.Solution {
		t.Fatalf("state diverged after round-trip: %+v vs %+v", e1, e2)
	}
}

// helper: build a chain with level 5 submitted (broken) and return it.
func chainWithBrokenLevel5(t *testing.T) *Chain {
	t.Helper()
	c := newTestChain(t)
	sub := Submission{Solution: "36", CircuitHash: "sha256:abc", VerifiedAt: "t"}
	_, _ = c.Append(Event{Type: EventSubmit, Level: 5, Submission: &sub, Timestamp: "t"})
	return c
}

// TestDeriveReproductionIncrements: a positive reproduction on a broken level
// raises the reproductions counter to 1.
func TestDeriveReproductionIncrements(t *testing.T) {
	c := chainWithBrokenLevel5(t)
	rep := &Reproduction{Author: "lab-b", CircuitHash: "sha256:rep", Result: ReproductionReproduced}
	_, _ = c.Append(Event{Type: EventReproduce, Level: 5, Reproduction: rep, Timestamp: "t"})
	r, err := DeriveRegistry(c)
	if err != nil {
		t.Fatalf("DeriveRegistry: %v", err)
	}
	e, _ := r.Entry(5)
	if e.Reproductions != 1 {
		t.Fatalf("reproductions = %d, want 1", e.Reproductions)
	}
}

// TestDeriveReproductionAccumulates: multiple positive reproductions accumulate.
func TestDeriveReproductionAccumulates(t *testing.T) {
	c := chainWithBrokenLevel5(t)
	for _, author := range []string{"lab-b", "lab-c", "lab-d"} {
		rep := &Reproduction{Author: author, CircuitHash: "sha256:rep", Result: ReproductionReproduced}
		_, _ = c.Append(Event{Type: EventReproduce, Level: 5, Reproduction: rep, Timestamp: "t"})
	}
	r, err := DeriveRegistry(c)
	if err != nil {
		t.Fatalf("DeriveRegistry: %v", err)
	}
	e, _ := r.Entry(5)
	if e.Reproductions != 3 {
		t.Fatalf("reproductions = %d, want 3", e.Reproductions)
	}
}

// TestDeriveReproductionFailedDoesNotIncrement: a failed reproduction is
// recorded but does not add positive confidence.
func TestDeriveReproductionFailedDoesNotIncrement(t *testing.T) {
	c := chainWithBrokenLevel5(t)
	rep := &Reproduction{Author: "lab-x", CircuitHash: "sha256:rep", Result: ReproductionFailed}
	_, _ = c.Append(Event{Type: EventReproduce, Level: 5, Reproduction: rep, Timestamp: "t"})
	r, err := DeriveRegistry(c)
	if err != nil {
		t.Fatalf("DeriveRegistry: %v", err)
	}
	e, _ := r.Entry(5)
	if e.Reproductions != 0 {
		t.Fatalf("failed reproduction incremented counter to %d, want 0", e.Reproductions)
	}
}

// TestDeriveReproductionRejectsNotBroken: reproducing a level that was never
// broken is invalid and must fail derivation.
func TestDeriveReproductionRejectsNotBroken(t *testing.T) {
	c := newTestChain(t) // level 5 never touched
	rep := &Reproduction{Author: "lab-b", CircuitHash: "sha256:rep", Result: ReproductionReproduced}
	_, _ = c.Append(Event{Type: EventReproduce, Level: 5, Reproduction: rep, Timestamp: "t"})
	if _, err := DeriveRegistry(c); err == nil {
		t.Fatal("expected DeriveRegistry to reject a reproduction on a non-broken level")
	}
}

// TestDeriveReproductionPersistsAfterRoundTrip: the counter survives save/load.
func TestDeriveReproductionPersistsAfterRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/chain.json"
	c1 := NewChain(path)
	_ = c1.Load()
	sub := Submission{Solution: "36", CircuitHash: "sha256:abc", VerifiedAt: "t"}
	_, _ = c1.Append(Event{Type: EventSubmit, Level: 5, Submission: &sub, Timestamp: "t"})
	rep := &Reproduction{Author: "lab-b", CircuitHash: "sha256:rep", Result: ReproductionReproduced}
	_, _ = c1.Append(Event{Type: EventReproduce, Level: 5, Reproduction: rep, Timestamp: "t"})
	_ = c1.Save()

	c2 := NewChain(path)
	_ = c2.Load()
	r, err := DeriveRegistry(c2)
	if err != nil {
		t.Fatalf("reload DeriveRegistry: %v", err)
	}
	e, _ := r.Entry(5)
	if e.Reproductions != 1 {
		t.Fatalf("reproductions after round-trip = %d, want 1", e.Reproductions)
	}
}
