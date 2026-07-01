package qlab

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func newTestRegistry(t *testing.T) *Registry {
	t.Helper()
	r := NewRegistry(filepath.Join(t.TempDir(), "registry.json"))
	if err := r.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	return r
}

func TestRegistryLoadMissingFileIsEmpty(t *testing.T) {
	r := newTestRegistry(t)
	if got := len(r.All()); got != 0 {
		t.Fatalf("expected empty registry, got %d entries", got)
	}
}

func TestRegistryEntryCreatedOpen(t *testing.T) {
	r := newTestRegistry(t)
	e, existed := r.Entry(5)
	if existed {
		t.Fatal("first Entry() should report not-existed")
	}
	if e.State != StateOpen {
		t.Fatalf("state = %s, want open", e.State)
	}
	if e.ChallengeID == "" {
		t.Fatal("expected non-empty challenge id")
	}
	if e.Level != 5 {
		t.Fatalf("level = %d, want 5", e.Level)
	}
}

func TestRegistryEntryInvalidLevel(t *testing.T) {
	r := newTestRegistry(t)
	if e, _ := r.Entry(0); e != nil {
		t.Fatal("Entry(0) should return nil")
	}
	if e, _ := r.Entry(-1); e != nil {
		t.Fatal("Entry(-1) should return nil")
	}
}

// TestSubmitSuccessAdvancesToBroken: a verified submission must collapse
// open→broken in one step and stamp VerifiedAt.
func TestSubmitSuccessAdvancesToBroken(t *testing.T) {
	r := newTestRegistry(t)
	s := Submission{Solution: "36", CircuitHash: "sha256:abc"}
	err := r.Submit(5, s, func() bool { return true })
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	e, _ := r.Entry(5)
	if e.State != StateBroken {
		t.Fatalf("state = %s, want broken", e.State)
	}
	if e.Submission == nil {
		t.Fatal("submission not stored")
	}
	if e.Submission.VerifiedAt == "" {
		t.Fatal("VerifiedAt not stamped")
	}
	if e.Submission.ChallengeID != e.ChallengeID {
		t.Fatalf("submission challenge id %q != entry %q", e.Submission.ChallengeID, e.ChallengeID)
	}
}

// TestSubmitFailureKeepsOpen: a failed verification must not mutate state.
func TestSubmitFailureKeepsOpen(t *testing.T) {
	r := newTestRegistry(t)
	err := r.Submit(5, Submission{Solution: "wrong"}, func() bool { return false })
	if err == nil {
		t.Fatal("expected verification error")
	}
	e, _ := r.Entry(5)
	if e.State != StateOpen {
		t.Fatalf("state = %s, want open after failed verify", e.State)
	}
	if e.Submission != nil {
		t.Fatal("submission must not be stored on failed verify")
	}
}

// TestSubmitOnNonOpenRejected: once broken, a level must reject new submissions.
func TestSubmitOnNonOpenRejected(t *testing.T) {
	r := newTestRegistry(t)
	if err := r.Submit(5, Submission{Solution: "36"}, func() bool { return true }); err != nil {
		t.Fatalf("first submit: %v", err)
	}
	err := r.Submit(5, Submission{Solution: "36"}, func() bool { return true })
	if err == nil {
		t.Fatal("expected error submitting to a broken level")
	}
}

// TestValidTransition covers every accepted and a few rejected edges.
func TestValidTransition(t *testing.T) {
	accepted := map[EntryState]EntryState{
		StateOpen:     StateClaimed,
		StateClaimed:  StateVerified,
		StateVerified: StateBroken,
		StateBroken:   StateHardened,
		StateHardened: StateReopened,
	}
	for from, to := range accepted {
		if !ValidTransition(from, to) {
			t.Fatalf("expected %s → %s to be valid", from, to)
		}
	}
	rejected := []struct{ from, to EntryState }{
		{StateOpen, StateHardened},   // skip too many steps
		{StateBroken, StateOpen},     // no going back
		{StateReopened, StateOpen},   // reopened is terminal here
		{StateHardened, StateBroken}, // no going back
	}
	for _, c := range rejected {
		if ValidTransition(c.from, c.to) {
			t.Fatalf("expected %s → %s to be invalid", c.from, c.to)
		}
	}
}

// TestReopenedAdvancesClock: reopening level N must open level N+1.
func TestReopenedAdvancesClock(t *testing.T) {
	r := newTestRegistry(t)
	if err := r.Submit(5, Submission{Solution: "36"}, func() bool { return true }); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if err := r.Transition(5, StateHardened); err != nil {
		t.Fatalf("transition to hardened: %v", err)
	}
	if err := r.Transition(5, StateReopened); err != nil {
		t.Fatalf("transition to reopened: %v", err)
	}
	next, existed := r.Entry(6)
	if next == nil {
		t.Fatal("reopened did not open level 6")
	}
	if next.State != StateOpen {
		t.Fatalf("level 6 state = %s, want open", next.State)
	}
	_ = existed
}

// TestSaveLoadRoundTrip: persisting and reloading preserves entries and states.
func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")

	r1 := NewRegistry(path)
	_ = r1.Load()
	if err := r1.Submit(5, Submission{Solution: "36", CircuitHash: "sha256:abc"}, func() bool { return true }); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if err := r1.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	r2 := NewRegistry(path)
	if err := r2.Load(); err != nil {
		t.Fatalf("reload: %v", err)
	}
	e, _ := r2.Entry(5)
	if e.State != StateBroken {
		t.Fatalf("after reload state = %s, want broken", e.State)
	}
	if e.Submission == nil || e.Submission.Solution != "36" {
		t.Fatalf("submission not preserved: %+v", e.Submission)
	}
}

// TestTransitionInvalidEdgeRejected: a bad edge must error without mutating.
func TestTransitionInvalidEdgeRejected(t *testing.T) {
	r := newTestRegistry(t)
	r.Entry(5)                            // open
	err := r.Transition(5, StateHardened) // open -> hardened skipped
	if err == nil {
		t.Fatal("expected invalid-transition error")
	}
	e, _ := r.Entry(5)
	if e.State != StateOpen {
		t.Fatalf("state mutated to %s on rejected transition", e.State)
	}
}

// TestSubmissionReportFieldsRoundTrip: the Phase 3 report fields (spec §6)
// survive JSON marshal/unmarshal.
func TestSubmissionReportFieldsRoundTrip(t *testing.T) {
	s := Submission{
		ChallengeID:                "qlab-L005-b6816f32eb",
		Level:                      5,
		ClaimedLogicalAttackQubits: 5,
		Solution:                   "36",
		CircuitHash:                "sha256:abc",
		CircuitDescription:         "3-qubit order-finding circuit",
		ReproducibilityNotes:       "run on simulator, 1024 shots",
		VerificationProof:          "2^36 mod 37 == 1, minimal",
		MeasuredOutputs:            map[string]interface{}{"peak_prob": 0.98},
	}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Submission
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.CircuitDescription != s.CircuitDescription ||
		got.ReproducibilityNotes != s.ReproducibilityNotes ||
		got.VerificationProof != s.VerificationProof ||
		got.MeasuredOutputs["peak_prob"] != s.MeasuredOutputs["peak_prob"] {
		t.Fatalf("report fields lost in round-trip: %+v", got)
	}
}

// TestSubmissionOmitsEmptyReportFields: empty report fields are omitted, so old
// chains/submissions keep their compact shape.
func TestSubmissionOmitsEmptyReportFields(t *testing.T) {
	b, err := json.Marshal(Submission{Solution: "36", CircuitHash: "sha256:abc"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	for _, key := range []string{"circuit_description", "measured_outputs", "reproducibility_notes", "verification_proof"} {
		if containsSubstring(s, key) {
			t.Fatalf("empty report field %q should be omitted, got %s", key, s)
		}
	}
}

func containsSubstring(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
