package qlab

import "testing"

func TestBitcoinThreshold(t *testing.T) {
	got := LogicalQubitsForECDLP(BitcoinCurveBits)
	if got != BitcoinLogicalThreshold {
		t.Fatalf("LogicalQubitsForECDLP(%d) = %d, want %d", BitcoinCurveBits, got, BitcoinLogicalThreshold)
	}
}

func TestLevelFamilies(t *testing.T) {
	tests := map[int]string{
		1:    "quantum-primitive",
		3:    "quantum-primitive",
		4:    "toy-order-finding",
		18:   "toy-order-finding",
		19:   "toy-ecdlp",
		2330: "bitcoin-reference",
	}
	for level, want := range tests {
		if got := LevelSpec(level).Family; got != want {
			t.Fatalf("LevelSpec(%d).Family = %q, want %q", level, got, want)
		}
	}
}

func TestFirstECDLPLevelMatchesOneBitCurve(t *testing.T) {
	// The ECDLP family starts exactly when the resource model can fit a one-bit
	// curve. If this fails, the level-family boundary has drifted from the model.
	if FirstECDLPLevel != LogicalQubitsForECDLP(1) {
		t.Fatalf("FirstECDLPLevel = %d, want LogicalQubitsForECDLP(1) = %d", FirstECDLPLevel, LogicalQubitsForECDLP(1))
	}
	if FirstECDLPLevel != 19 {
		t.Fatalf("FirstECDLPLevel = %d, want 19", FirstECDLPLevel)
	}
	if LevelSpec(FirstECDLPLevel).Family != "toy-ecdlp" {
		t.Fatalf("LevelSpec(%d).Family = %q, want toy-ecdlp", FirstECDLPLevel, LevelSpec(FirstECDLPLevel).Family)
	}
}

func TestChallengeIsDeterministic(t *testing.T) {
	a := ChallengeForLevel(32)
	b := ChallengeForLevel(32)
	if a.ID != b.ID {
		t.Fatalf("challenge id changed: %q != %q", a.ID, b.ID)
	}
	if a.RequiredLogicalQubits != 32 {
		t.Fatalf("required logical qubits = %d, want 32", a.RequiredLogicalQubits)
	}
}
