# Challenge Format

Challenges are JSON documents. They are produced by the CLI and have a
deterministic id of the form `qlab-L<NNN>-<hash10>`, where `<hash10>` is the
first 10 hex chars of `sha256("qlabcoin:<level>:<family>")`.

```bash
go run ./cmd/qlabcoin challenge 1
```

```json
{
  "id": "qlab-L001-f845323307",
  "level": 1,
  "required_logical_qubits": 1,
  "family": "quantum-primitive",
  "status": "open",
  "target": {
    "description": "Demonstrate useful logical attack qubits in a repeatable quantum subroutine.",
    "type": "quantum-primitive",
    "win_condition": "submit measured output, circuit hash, and reproducible verification notes"
  },
  "verification": {
    "classical": true,
    "requires_backend_report": true,
    "requires_circuit_hash": true
  },
  "mitigation_after_break": "publish result; open next level"
}
```

## Families

```text
quantum-primitive
  Levels 1-3. Demonstrates useful logical qubits and repeatable circuits.

toy-order-finding
  Levels 4 up to FirstECDLPLevel-1. Early Shor-like period-finding and
  small discrete-log-shaped challenges over tiny groups. Cheap to verify
  classically.

toy-ecdlp
  FirstECDLPLevel (19) and above. Tiny elliptic-curve-discrete-log-shaped
  tasks under the reference resource model, starting when a one-bit curve fits.

bitcoin-reference
  Level 2330. A non-spendable reference marker for secp256k1-level estimates.
```

The boundary between `toy-order-finding` and `toy-ecdlp` is
`FirstECDLPLevel`, derived from the resource model
(`LogicalQubitsForECDLP(1) = 19`) rather than hard-coded.

## Toy order-finding targets

For levels in the `toy-order-finding` band (4 .. `FirstECDLPLevel-1`), the
challenge `target` carries a deterministic group: a base `g` and a prime modulus
`m`, both derived from the level so the same level always yields the same target.

```bash
go run ./cmd/qlabcoin challenge 5
```

```json
{
  "target": {
    "base": 2,
    "modulus": 37,
    "type": "toy-order-finding",
    "hint": "find the multiplicative order of 2 modulo 37 (least k>=1 with 2^k ≡ 1 mod 37)"
  }
}
```

The solution is the multiplicative order: the least `k >= 1` with `g^k ≡ 1 (mod m)`.
For level 5 that order is 36. Classical verification checks that the claim holds
*and* is minimal (no proper divisor of the claim already reaches 1).

## Submission and state

`submit` verifies the solution classically and, on success, advances the level
`open → claimed → verified → broken` in one step:

```bash
go run ./cmd/qlabcoin submit 5 -solution 36 -circuit sha256:example
```

State is persisted in a local JSON registry (default `qlabcoin-registry.json`).
The remaining manual steps use `transition`:

```bash
go run ./cmd/qlabcoin transition 5 hardened
go run ./cmd/qlabcoin transition 5 reopened   # opens level 6
```

See `examples/submission-005.json` for a full winning entry.

## Solver Proof

```json
{
  "challenge_id": "qlab-L019-89834f043f",
  "claimed_logical_attack_qubits": 19,
  "solution": "1",
  "circuit_hash": "sha256:...",
  "backend": {
    "hardware": "example university lab",
    "physical_qubits": 12,
    "logical_qubits": 1,
    "notes": "demonstration run"
  }
}
```

Qlabcoin must verify the solution classically before advancing the clock.
