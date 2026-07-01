package qlab

import "fmt"

// Toy-order-finding challenges (levels QuantumPrimitiveMaxLevel+1 .. FirstECDLPLevel-1).
//
// This is the smallest faithful Shor sub-product: find the multiplicative order
// of a base g modulo a small prime m, i.e. the least k>=1 with g^k ≡ 1 (mod m).
// The modulus and base are derived deterministically from the level, so a given
// level always produces the same challenge (reproducible examples/registry).

// ToyOrderMaxBase caps the search space for a non-trivial base. Small enough that
// the challenge is solvable by hand yet the order is non-trivial.
const toyOrderMinModulus = 31

// ToyOrderChallenge describes a level's toy order-finding target.
type ToyOrderChallenge struct {
	Level   int    `json:"level"`
	Modulus int    `json:"modulus"`
	Base    int    `json:"base"`
	Family  string `json:"family"` // always "toy-order-finding"
	Hint    string `json:"hint"`
}

// IsToyOrderLevel reports whether level is in the order-finding band.
func IsToyOrderLevel(level int) bool {
	return level > QuantumPrimitiveMaxLevel && level < FirstECDLPLevel
}

// ToyOrderChallengeForLevel returns the deterministic toy order-finding
// challenge for level. Callers should only invoke this for levels in the
// toy-order-finding band (see IsToyOrderLevel); for other levels the modulus
// and base are 0 and the hint explains the family is not order-finding.
func ToyOrderChallengeForLevel(level int) ToyOrderChallenge {
	c := ToyOrderChallenge{
		Level:  level,
		Family: "toy-order-finding",
	}
	if !IsToyOrderLevel(level) {
		c.Hint = fmt.Sprintf("level %d is not a toy-order-finding challenge", level)
		return c
	}
	m := toyModulus(level)
	g := toyBase(level, m)
	c.Modulus = m
	c.Base = g
	c.Hint = fmt.Sprintf("find the multiplicative order of %d modulo %d (least k>=1 with %d^k ≡ 1 mod %d)", g, m, g, m)
	return c
}

// VerifyOrder checks a claimed multiplicative order classically. It is strict:
// the claim must hold (base^claim ≡ 1 mod modulus) AND be minimal (dividing the
// claim by any prime factor must stop being congruent to 1). modulus and base
// must match the deterministic challenge for level; otherwise the result is
// false. Out-of-band levels are rejected.
func VerifyOrder(level, modulus, base, claimedOrder int) bool {
	if !IsToyOrderLevel(level) {
		return false
	}
	want := ToyOrderChallengeForLevel(level)
	if modulus != want.Modulus || base != want.Base {
		return false
	}
	if claimedOrder < 1 {
		return false
	}
	// (a) the claim must hold.
	if modPow(base, claimedOrder, modulus) != 1 {
		return false
	}
	// (b) minimal: no proper divisor d = claim/p may already give 1.
	for _, p := range primeFactors(claimedOrder) {
		if modPow(base, claimedOrder/p, modulus) == 1 {
			return false // a smaller exponent already reaches 1, so claim is not minimal
		}
	}
	return true
}

// SolveOrder returns the true multiplicative order of base modulo modulus for
// the deterministic challenge at level. Used by tests and as a reference; not
// needed by a solver. Returns 0 for out-of-band levels or non-coprime pairs.
func SolveOrder(level, modulus, base int) int {
	if !IsToyOrderLevel(level) {
		return 0
	}
	if modulus <= 0 {
		return 0
	}
	// Brute force up to modulus: order always divides phi(modulus) < modulus.
	for k := 1; k < modulus; k++ {
		if modPow(base, k, modulus) == 1 {
			return k
		}
	}
	return 0
}

// toyModulus picks a small prime deterministically from the level. It scans
// primes starting at toyOrderMinModulus and selects the (levelOffset+1)-th,
// where levelOffset is the position of level within the order-finding band.
func toyModulus(level int) int {
	offset := level - (QuantumPrimitiveMaxLevel + 1) // 0-based index in the band
	p := toyOrderMinModulus
	for i := 0; i < offset; i++ {
		p = nextPrime(p + 1)
	}
	return p
}

// toyBase picks a deterministic non-trivial base coprime to modulus. We avoid
// 1 and modulus-1 (orders 1 and 2, trivially guessable) and ensure gcd==1.
func toyBase(level, modulus int) int {
	for g := 2 + (level % 5); g < modulus; g++ {
		if g == modulus-1 {
			continue
		}
		if gcd(g, modulus) != 1 {
			continue
		}
		// skip trivially small orders so the challenge is non-trivial
		if modPow(g, 2, modulus) == 1 {
			continue
		}
		return g
	}
	// fallback: smallest valid base > 1
	for g := 2; g < modulus; g++ {
		if gcd(g, modulus) == 1 {
			return g
		}
	}
	return 2
}

// nextPrime returns the smallest prime >= n (n >= 2).
func nextPrime(n int) int {
	for {
		if isPrime(n) {
			return n
		}
		n++
	}
}

func isPrime(n int) bool {
	if n < 2 {
		return false
	}
	for d := 2; d*d <= n; d++ {
		if n%d == 0 {
			return false
		}
	}
	return true
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	if a < 0 {
		return -a
	}
	return a
}

// modPow computes (base^exp) % mod for non-negative exp, mod > 0.
func modPow(base, exp, mod int) int {
	if mod == 1 {
		return 0
	}
	result := 1
	base %= mod
	if base < 0 {
		base += mod
	}
	for exp > 0 {
		if exp&1 == 1 {
			result = (result * base) % mod
		}
		exp >>= 1
		base = (base * base) % mod
	}
	return result
}

// primeFactors returns the distinct prime factors of n (n >= 1).
func primeFactors(n int) []int {
	if n < 2 {
		return nil
	}
	var factors []int
	d := 2
	for d*d <= n {
		if n%d == 0 {
			factors = append(factors, d)
			for n%d == 0 {
				n /= d
			}
		}
		d++
	}
	if n > 1 {
		factors = append(factors, n)
	}
	return factors
}
