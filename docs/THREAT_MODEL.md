# Threat Model

## Threats Studied

- Shor-style period finding.
- Discrete logarithm recovery in tiny groups.
- Elliptic-curve public-key exposure.
- Unsafe reuse of addresses after public-key exposure.
- Migration from vulnerable signatures to quantum-resistant signatures.

## Threats Not Claimed

- Current hardware breaking Bitcoin.
- Physical qubit count directly implying logical qubit count.
- A toy challenge being financially meaningful.

## Honest Language

Use:

```text
This level was broken with N demonstrated logical attack qubits.
```

Do not use:

```text
This number of physical qubits can break Bitcoin.
```

## Defensive Lessons

- Hide public keys until spend where possible.
- Avoid live funds on exposed public keys.
- Prefer short migration windows once exposure occurs.
- Prepare hybrid and post-quantum signature paths before crisis.
