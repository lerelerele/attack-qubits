package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"

	"qlabcoin/internal/qlab"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "clock":
		clock(os.Args[2:])
	case "level":
		level(os.Args[2:])
	case "challenge":
		challenge(os.Args[2:])
	case "verify":
		verify(os.Args[2:])
	case "submit":
		submit(os.Args[2:])
	case "transition":
		transition(os.Args[2:])
	case "state":
		state(os.Args[2:])
	case "bitcoin":
		bitcoin()
	default:
		fmt.Fprintln(os.Stderr, "unknown command:", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Printf(`Qlabcoin %s

Commands:
  qlabcoin clock [-max 20]
  qlabcoin level <n>
  qlabcoin challenge <n>
  qlabcoin verify <n> -solution <k>
  qlabcoin submit <n> -solution <k> -circuit <sha256:...> [-backend <json>] [-registry <path>]
  qlabcoin transition <n> <state> [-registry <path>]
  qlabcoin state [-registry <path>]
  qlabcoin bitcoin

States: open, claimed, verified, broken, hardened, reopened
Registry: a local JSON file (default %s); not committed.
`, qlab.Version, qlab.DefaultRegistryPath)
}

func clock(args []string) {
	fs := flag.NewFlagSet("clock", flag.ExitOnError)
	max := fs.Int("max", 20, "maximum level to print")
	_ = fs.Parse(args)
	if *max < 1 {
		*max = 1
	}
	fmt.Printf("%-6s %-8s %-20s %-10s %-8s\n", "Level", "Qubits", "Family", "CurveBits", "BTC%")
	for i := 1; i <= *max; i++ {
		spec := qlab.LevelSpec(i)
		curve := "-"
		if spec.EstimatedCurveBits > 0 {
			curve = strconv.Itoa(spec.EstimatedCurveBits)
		}
		fmt.Printf("%-6d %-8d %-20s %-10s %6.2f\n", spec.Level, spec.RequiredLogicalQubits, spec.Family, curve, spec.BitcoinDistancePercent)
	}
}

func level(args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "level requires one number")
		os.Exit(1)
	}
	n := mustLevel(args[0])
	printJSON(qlab.LevelSpec(n))
}

func challenge(args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "challenge requires one number")
		os.Exit(1)
	}
	n := mustLevel(args[0])
	c := qlab.ChallengeForLevel(n)
	// For toy-order-finding levels, embed the deterministic group parameters so a
	// solver has everything needed to attempt the challenge.
	if qlab.IsToyOrderLevel(n) {
		toy := qlab.ToyOrderChallengeForLevel(n)
		c.Target["modulus"] = toy.Modulus
		c.Target["base"] = toy.Base
		c.Target["hint"] = toy.Hint
	}
	printJSON(c)
}

// reorderFlags moves flag tokens (and their values) before positional args so
// that stdlib flag parsing accepts the natural "cmd <level> -flag value" order.
// It assumes every flag takes a value (true for all qlabcoin flags: -max,
// -solution, -circuit, -backend, -registry). A "-x=v" token is self-contained.
func reorderFlags(args []string) []string {
	var flags, pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if len(a) > 0 && a[0] == '-' {
			flags = append(flags, a)
			if !containsEqual(a) && i+1 < len(args) && (len(args[i+1]) == 0 || args[i+1][0] != '-') {
				i++
				flags = append(flags, args[i])
			}
			continue
		}
		pos = append(pos, a)
	}
	return append(flags, pos...)
}

func containsEqual(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return true
		}
	}
	return false
}

// verify reports whether -solution is the multiplicative order for the level's
// deterministic toy group. Intended for inspection; submit() is the path that
// mutates state.
func verify(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	solution := fs.Int("solution", 0, "claimed multiplicative order to check")
	_ = fs.Parse(reorderFlags(args))
	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(os.Stderr, "verify requires a level number")
		os.Exit(1)
	}
	n := mustLevel(rest[0])
	if !qlab.IsToyOrderLevel(n) {
		fmt.Fprintf(os.Stderr, "level %d is not a toy-order-finding challenge; classical verification is not implemented for that family yet\n", n)
		os.Exit(2)
	}
	toy := qlab.ToyOrderChallengeForLevel(n)
	ok := qlab.VerifyOrder(n, toy.Modulus, toy.Base, *solution)
	printJSON(map[string]interface{}{
		"level":      n,
		"modulus":    toy.Modulus,
		"base":       toy.Base,
		"solution":   *solution,
		"verified":   ok,
		"true_order": qlab.SolveOrder(n, toy.Modulus, toy.Base),
	})
	if !ok {
		os.Exit(1)
	}
}

// submit records a submission against a level, verifies it classically, and on
// success advances the entry open→broken in one step.
func submit(args []string) {
	fs := flag.NewFlagSet("submit", flag.ExitOnError)
	solution := fs.Int("solution", 0, "claimed multiplicative order")
	circuit := fs.String("circuit", "", "circuit hash, e.g. sha256:...")
	backend := fs.String("backend", "", "backend metadata as JSON object")
	registry := fs.String("registry", qlab.DefaultRegistryPath, "registry file path")
	_ = fs.Parse(reorderFlags(args))
	rest := fs.Args()
	if len(rest) != 1 {
		fmt.Fprintln(os.Stderr, "submit requires a level number")
		os.Exit(1)
	}
	n := mustLevel(rest[0])
	if *circuit == "" {
		fmt.Fprintln(os.Stderr, "submit requires -circuit")
		os.Exit(1)
	}
	if !qlab.IsToyOrderLevel(n) {
		fmt.Fprintf(os.Stderr, "level %d is not a toy-order-finding challenge; classical verification is not implemented for that family yet\n", n)
		os.Exit(2)
	}
	var backendMap map[string]interface{}
	if *backend != "" {
		if err := json.Unmarshal([]byte(*backend), &backendMap); err != nil {
			fmt.Fprintf(os.Stderr, "invalid -backend JSON: %v\n", err)
			os.Exit(2)
		}
	}
	toy := qlab.ToyOrderChallengeForLevel(n)
	sub := qlab.Submission{
		Solution:    strconv.Itoa(*solution),
		CircuitHash: *circuit,
		Backend:     backendMap,
	}

	reg := qlab.NewRegistry(*registry)
	if err := reg.Load(); err != nil {
		fatal(err)
	}
	err := reg.Submit(n, sub, func() bool {
		return qlab.VerifyOrder(n, toy.Modulus, toy.Base, *solution)
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := reg.Save(); err != nil {
		fatal(err)
	}
	entry, _ := reg.Entry(n)
	printJSON(entry)
}

// transition moves a level to a new state via a validated lifecycle edge.
func transition(args []string) {
	fs := flag.NewFlagSet("transition", flag.ExitOnError)
	registry := fs.String("registry", qlab.DefaultRegistryPath, "registry file path")
	_ = fs.Parse(reorderFlags(args))
	rest := fs.Args()
	if len(rest) != 2 {
		fmt.Fprintln(os.Stderr, "transition requires <level> <state>")
		os.Exit(1)
	}
	n := mustLevel(rest[0])
	to := qlab.EntryState(rest[1])
	switch to {
	case qlab.StateOpen, qlab.StateClaimed, qlab.StateVerified, qlab.StateBroken, qlab.StateHardened, qlab.StateReopened:
		// ok
	default:
		fmt.Fprintf(os.Stderr, "unknown state %q\n", rest[1])
		os.Exit(2)
	}
	reg := qlab.NewRegistry(*registry)
	if err := reg.Load(); err != nil {
		fatal(err)
	}
	if err := reg.Transition(n, to); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := reg.Save(); err != nil {
		fatal(err)
	}
	entry, _ := reg.Entry(n)
	printJSON(entry)
}

// state dumps the full registry as JSON.
func state(args []string) {
	fs := flag.NewFlagSet("state", flag.ExitOnError)
	registry := fs.String("registry", qlab.DefaultRegistryPath, "registry file path")
	_ = fs.Parse(reorderFlags(args))
	reg := qlab.NewRegistry(*registry)
	if err := reg.Load(); err != nil {
		fatal(err)
	}
	printJSON(map[string]interface{}{"entries": reg.All()})
}

func bitcoin() {
	spec := qlab.LevelSpec(qlab.BitcoinLogicalThreshold)
	printJSON(map[string]interface{}{
		"label":                    "bitcoin-reference",
		"curve_bits":               qlab.BitcoinCurveBits,
		"logical_qubits":           qlab.LogicalQubitsForECDLP(qlab.BitcoinCurveBits),
		"toffoli_gates":            spec.EstimatedToffoliGates,
		"warning":                  "Logical-qubit threshold only; not a practical break claim without depth, runtime, and physical error-correction resources.",
		"qlabcoin_reference_level": qlab.BitcoinLogicalThreshold,
	})
}

func mustLevel(raw string) int {
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		fmt.Fprintln(os.Stderr, "level must be a positive integer")
		os.Exit(1)
	}
	return n
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func printJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
