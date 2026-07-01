# Next Steps

## Immediate

1. Publish the repository as research-only.
2. Add a simple challenge registry file.
3. Implement classical verification for toy-order-finding challenges.
4. Add solver submission JSON files.
5. Add a state transition command:

```text
open -> claimed -> verified -> broken -> hardened -> reopened
```

## First Working Demo

The first real demo should not be an elliptic-curve challenge. It should be:

```text
Level 1:
  one useful logical qubit;
  repeatable circuit;
  circuit hash;
  backend report;
  classically verifiable result.
```

## First Cryptographic Demo

Start with tiny order-finding before ECDLP:

```text
Level 4-18:
  toy-order-finding;
  tiny modulus/group;
  cheap classical verification;
  reproducible circuit report.
```

## First ECDLP-Shaped Demo

Use level 19 and above only as ECDLP-shaped reference challenges under the Roetteler-Naehrig-Svore-Lauter resource model.

## Public Messaging

Use:

```text
Qlabcoin measures demonstrated logical attack qubits.
```

Avoid:

```text
This many physical qubits can break Bitcoin.
```
