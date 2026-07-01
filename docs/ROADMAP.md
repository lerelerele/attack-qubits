# Roadmap

## Phase 0: Project Shell

- Define level model.
- Define challenge JSON.
- Create CLI for clock and challenge generation.
- Publish assumptions.

## Phase 1: Toy Challenge Engine (done)

- Deterministic toy order-finding groups for levels 4-18.
- Classical verification (multiplicative order, strict minimality).
- Solver submission via `submit` (verifies and advances to broken).
- Challenge state transitions via `transition`; local JSON registry via `state`.
- Remaining: classical verification for quantum-primitive (1-3) and toy-ecdlp
  (19+) families.

## Phase 2: Local Chain

- Blocks.
- Challenge registry.
- Broken-level history.
- Hardening events.
- Reopen events.

## Phase 3: Academic Clock

- Level 1 starts with one useful logical attack qubit.
- Require reports and circuit hashes.
- Track independent reproductions.

## Phase 4: Mitigation Lab

- Exposed-pubkey mode.
- Hash-only-address mode.
- No-live-UTXO-after-exposure rule.
- Hybrid signature mode.
- Post-quantum signature mode.

## Phase 5: Bitcoin Distance Model

- Reference secp256k1 threshold.
- Multiple hardware/error-correction assumptions.
- Public dashboard.

## Phase 6: External Review

- University challenge submissions.
- Reproducibility review.
- Published results.
