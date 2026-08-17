// Package main implements the kui CLI entry point.
package main

import (
	"fmt"
	"strings"
)

// Options holds all parsed CLI flag values. Fields default to zero values
// (empty string / false) when no flags are provided, preserving existing
// behavior (REQ-CLI-10).
type Options struct {
	// Model overrides the resolved model from the REQ-CLI-4 chain.
	Model string
	// Tools is a comma-separated list of tool names to include (empty = all).
	Tools string
	// ExcludeTools is a comma-separated list of tool names to exclude.
	ExcludeTools string
	// NoTools disables all tools (empty registry).
	NoTools bool
	// NoExtensions disables extension loading.
	NoExtensions bool
	// NoSkills disables skill index building.
	NoSkills bool
	// NoSession is a no-op placeholder for future session persistence control.
	NoSession bool
	// Verbose enables debug output to stderr.
	Verbose bool
	// Mode selects output format: "text" (default) or "json".
	Mode string
	// Approve bypasses permission prompts (permissive ruleset).
	Approve bool
	// Print writes the answer to stdout regardless of mode.
	Print bool
	// Thinking selects the reasoning effort level: off, low, medium, high.
	Thinking string
}

// stringFlags maps long flag names to whether they take a value argument.
// Boolean flags are handled separately via boolFlags.
var stringFlags = map[string]bool{
	"model":         true,  // --model <value>
	"tools":         true,  // --tools <value>
	"exclude-tools": true,  // --exclude-tools <value>
	"mode":          true,  // --mode <value>
	"thinking":      true,  // --thinking <value>
}

// shortMap maps single-character short flags to their long equivalents.
// Boolean short flags map to empty string (no value needed).
var shortMap = map[string]string{
	"m":  "model",
	"a":  "", // approve (bool)
	"p":  "", // print (bool)
	"t":  "tools",
	"xt": "exclude-tools",
	"nt": "", // no-tools (bool)
	"ne": "", // no-extensions (bool)
	"ns": "", // no-skills (bool)
}

// boolFlags lists the long flag names that are pure booleans (no value).
var boolFlags = map[string]bool{
	"verbose":       true,
	"no-tools":      true,
	"no-extensions": true,
	"no-skills":     true,
	"no-session":    true,
	"approve":       true,
	"print":         true,
}

// parseFlags processes a slice of CLI arguments and returns the parsed Options,
// remaining positional arguments (the prompt), and any error encountered.
// The special "--" separator stops flag processing; everything after it is
// returned as remaining args (REQ-CLI-7, REQ-CLI-8).
func parseFlags(args []string) (Options, []string, error) {
	var opts Options
	var remaining []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// "--" separates flags from positional arguments.
		if arg == "--" {
			remaining = append(remaining, args[i+1:]...)
			break
		}

		// Non-flag arguments are positional (the prompt).
		if len(arg) < 2 || arg[0] != '-' {
			remaining = append(remaining, arg)
			continue
		}

		// Parse flag name and optional value from the argument.
		name, value, hasEq := parseFlagArg(arg)

		// Check if this is a short flag (single dash, not double dash).
		if arg[1] != '-' {
			longName, ok := shortMap[name]
			if !ok {
				return Options{}, nil, fmt.Errorf("unknown flag: %s", arg)
			}
			// Boolean short flag.
			if longName == "" {
				switch name {
				case "a":
					opts.Approve = true
				case "p":
					opts.Print = true
				case "ne":
					opts.NoExtensions = true
				case "ns":
					opts.NoSkills = true
				default:
					return Options{}, nil, fmt.Errorf("unknown flag: %s", arg)
				}
				continue
			}
			// String short flag — needs a value.
			if hasEq {
				// -m=value not supported, treat as unknown.
				return Options{}, nil, fmt.Errorf("unknown flag: %s", arg)
			}
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return Options{}, nil, fmt.Errorf("flag %s requires a value", arg)
			}
			i++
			value = args[i]
			setStringOption(&opts, longName, value)
			continue
		}

		// Double-dash long flags.
		if hasEq {
			// --flag=value
			if boolFlags[name] {
				// Bool flags don't accept =value.
				switch name {
				case "verbose":
					opts.Verbose = true
				case "no-tools":
					opts.NoTools = true
				case "no-extensions":
					opts.NoExtensions = true
				case "no-skills":
					opts.NoSkills = true
				case "no-session":
					opts.NoSession = true
				case "approve":
					opts.Approve = true
				case "print":
					opts.Print = true
				default:
					return Options{}, nil, fmt.Errorf("unknown flag: %s", arg)
				}
				continue
			}
			if !stringFlags[name] {
				return Options{}, nil, fmt.Errorf("unknown flag: %s", arg)
			}
			setStringOption(&opts, name, value)
			continue
		}

		// --flag (no equals) — check if it's a bool or needs next arg.
		if boolFlags[name] {
			switch name {
			case "verbose":
				opts.Verbose = true
			case "no-tools":
				opts.NoTools = true
			case "no-extensions":
				opts.NoExtensions = true
			case "no-skills":
				opts.NoSkills = true
			case "no-session":
				opts.NoSession = true
			case "approve":
				opts.Approve = true
			case "print":
				opts.Print = true
			default:
				return Options{}, nil, fmt.Errorf("unknown flag: %s", arg)
			}
			continue
		}
		if stringFlags[name] {
			// Needs a value from next arg.
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return Options{}, nil, fmt.Errorf("flag --%s requires a value", name)
			}
			i++
			setStringOption(&opts, name, args[i])
			continue
		}

		return Options{}, nil, fmt.Errorf("unknown flag: %s", arg)
	}

	return opts, remaining, nil
}

// parseFlagArg splits a CLI argument into flag name, optional value, and whether
// an equals sign was present. For "-m value" this returns ("m", "", false).
// For "--model=gpt-4o" this returns ("model", "gpt-4o", true).
func parseFlagArg(arg string) (name, value string, hasEq bool) {
	if arg[1] == '-' {
		// Long flag: --name or --name=value
		body := arg[2:]
		if idx := strings.IndexByte(body, '='); idx >= 0 {
			return body[:idx], body[idx+1:], true
		}
		return body, "", false
	}
	// Short flag: -x or -x=value (value form not used but handled)
	body := arg[1:]
	if idx := strings.IndexByte(body, '='); idx >= 0 {
		return body[:idx], body[idx+1:], true
	}
	return body, "", false
}

// setStringOption assigns a value to the appropriate Options string field.
func setStringOption(opts *Options, name, value string) {
	switch name {
	case "model":
		opts.Model = value
	case "tools":
		opts.Tools = value
	case "exclude-tools":
		opts.ExcludeTools = value
	case "mode":
		opts.Mode = value
	case "thinking":
		opts.Thinking = value
	}
}
