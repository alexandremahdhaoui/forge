package main

import "testing"

// The front door: four verbs that are not forge's own but a companion
// binary's, handed the rest of the argv untouched. The whole feature had no
// test, so deleting the switch would have broken nothing in CI and
// everything on a desk.
func TestTheFrontDoorKnowsWhichVerbsAreNotItsOwn(t *testing.T) {
	for _, verb := range []string{"factory", "ci", "register", "cache"} {
		t.Run(verb, func(t *testing.T) {
			// The error is whatever the resolution answered; what this
			// pins is that the verb was claimed at all. An unclaimed one
			// falls through to forge's own parser and comes back as
			// "Unknown command", which is what a user saw before this
			// existed.
			if _, _, claimed := delegation(verb, nil); !claimed {
				t.Fatalf("%q fell through to forge's own commands", verb)
			}
		})
	}
}

func TestForgesOwnVerbsAreNotDelegated(t *testing.T) {
	// build and test are forge's. Claiming one would hand it to a binary
	// that does not have it, and the failure would name the wrong tool.
	for _, verb := range []string{"build", "test", "run", "list", "clone", ""} {
		t.Run(verb, func(t *testing.T) {
			if tool, _, claimed := delegation(verb, nil); claimed {
				t.Fatalf("%q was delegated to %q; it is forge's own", verb, tool.name)
			}
		})
	}
}

// cache is forge-factory's verb under forge's name, so the verb has to
// travel with the rest of the argv. Dropping it turns "forge cache clean"
// into a bare "forge-factory clean", which is not a command.
func TestTheCacheVerbTravelsWithItsArguments(t *testing.T) {
	tool, args, ok := delegation("cache", []string{"clean", "--cache", "x"})
	if !ok {
		t.Fatal("cache was not claimed")
	}

	if tool.name != "forge-factory" {
		t.Fatalf("cache went to %q", tool.name)
	}

	want := []string{"cache", "clean", "--cache", "x"}
	if len(args) != len(want) {
		t.Fatalf("got %v, want %v", args, want)
	}

	for i, w := range want {
		if args[i] != w {
			t.Fatalf("got %v, want %v", args, want)
		}
	}
}

// Each verb goes to the binary that has it. Sending "ci" to forge-factory
// fails naming the wrong tool, which is the hardest kind of error to read.
func TestEachVerbGoesToItsOwnBinary(t *testing.T) {
	for verb, want := range map[string]string{
		"factory":  "forge-factory",
		"cache":    "forge-factory",
		"ci":       "forge-ci",
		"register": "forge-register",
	} {
		t.Run(verb, func(t *testing.T) {
			tool, _, ok := delegation(verb, nil)
			if !ok {
				t.Fatalf("%q was not claimed", verb)
			}

			if tool.name != want {
				t.Fatalf("%q went to %q, want %q", verb, tool.name, want)
			}

			// Every companion names a module, so a machine with nothing
			// provisioned is told what provisions it rather than getting a
			// go run at a guessed version.
			if tool.module == "" {
				t.Fatalf("%q names no module", verb)
			}
		})
	}
}

// A global flag before the verb is forge's; everything from the verb on
// belongs to whoever owns the verb. Reading past the verb would eat a
// --config the child needed.
func TestOnlyTheFlagsBeforeTheVerbAreForges(t *testing.T) {
	defer func(c, w string) { configPath, cwdOverride = c, w }(configPath, cwdOverride)

	for name, tc := range map[string]struct {
		argv       []string
		wantVerb   string
		wantRest   []string
		wantCwd    string
		wantConfig string
	}{
		"no flags":      {[]string{"build"}, "build", []string{}, "", ""},
		"cwd then verb": {[]string{"--cwd", "/w", "ci", "apply"}, "ci", []string{"apply"}, "/w", ""},
		"joined form":   {[]string{"--cwd=/w", "register", "status"}, "register", []string{"status"}, "/w", ""},
		"config then verb": {
			[]string{"--config", "f.yaml", "build"}, "build", []string{}, "", "f.yaml",
		},
		// The child's flags reach it verbatim, including ones spelled the
		// same as forge's.
		"the child's own --config": {
			[]string{"ci", "apply", "--config", "theirs.yaml"},
			"ci",
			[]string{"apply", "--config", "theirs.yaml"},
			"", "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			configPath, cwdOverride = "", ""

			verb, rest := splitLeadingFlags(tc.argv)
			if verb != tc.wantVerb {
				t.Fatalf("verb %q, want %q", verb, tc.wantVerb)
			}

			if len(rest) != len(tc.wantRest) {
				t.Fatalf("rest %v, want %v", rest, tc.wantRest)
			}

			for i, w := range tc.wantRest {
				if rest[i] != w {
					t.Fatalf("rest %v, want %v", rest, tc.wantRest)
				}
			}

			if cwdOverride != tc.wantCwd {
				t.Fatalf("--cwd %q, want %q", cwdOverride, tc.wantCwd)
			}

			if configPath != tc.wantConfig {
				t.Fatalf("--config %q, want %q", configPath, tc.wantConfig)
			}
		})
	}
}

// An unknown leading flag is not swallowed. It becomes the verb, and the
// usual parser reports it, rather than being silently dropped.
func TestAnUnknownLeadingFlagIsNotEaten(t *testing.T) {
	verb, _ := splitLeadingFlags([]string{"--nope", "build"})
	if verb != "--nope" {
		t.Fatalf("verb %q; an unknown flag must reach the usual error", verb)
	}
}
