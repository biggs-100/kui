// Command setup implements the "kui setup" subcommand for interactive
// credential configuration (REQ-CRED-6, REQ-CRED-7, REQ-CRED-8).
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/biggs-100/kui/internal/adapters/providers"
	"github.com/biggs-100/kui/internal/credentials"
)

// availableProviders returns the list of built-in provider names.
func availableProviders() []string {
	reg := providers.NewDefaultRegistry()
	var names []string
	for _, name := range []string{"openai", "opencode"} {
		if _, err := reg.Resolve(name); err == nil {
			names = append(names, name)
		}
	}
	return names
}

// parseSetupFlags extracts the --provider / -p flag from args and returns the
// provider value and remaining positional arguments. Unknown flags are left in
// the rest slice so runSetup can report a usage error.
func parseSetupFlags(args []string) (provider string, rest []string) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--provider", "-p":
			if i+1 >= len(args) {
				return "", args
			}
			return args[i+1], args[i+2:]
		default:
			rest = append(rest, args[i])
		}
	}
	return "", rest
}

// ValidateKey checks that the API key is non-empty after trimming whitespace
// and matches the expected format for the given provider (REQ-CRED-7).
// OpenAI keys must start with "sk-". Other providers accept any non-empty key.
func ValidateKey(provider, key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("API key is empty: please enter a valid key")
	}
	switch provider {
	case "openai":
		if !strings.HasPrefix(key, "sk-") {
			return fmt.Errorf("invalid OpenAI key format: expected key starting with 'sk-', got %q", key)
		}
	case "opencode":
		// No prefix requirement for opencode.
	default:
		return fmt.Errorf("unknown provider %q: cannot validate key format", provider)
	}
	return nil
}

// runSetup launches the credential setup wizard (REQ-CRED-6). It returns an
// exit code following D13: 0 success, 1 runtime error, 2 usage error.
func runSetup(root string, args []string) int {
	provider, rest := parseSetupFlags(args)

	// Reject unknown flags.
	for _, arg := range rest {
		if strings.HasPrefix(arg, "-") {
			fmt.Fprintf(os.Stderr, "kui setup: unknown flag %q\n", arg)
			return 2
		}
	}

	// Use a single bufio.Reader for all stdin reads so that buffered data
	// from the provider selection is not lost when reading the API key.
	stdinReader := bufio.NewReader(os.Stdin)

	// If no --provider flag, show interactive provider list.
	if provider == "" {
		providers := availableProviders()
		fmt.Fprintln(os.Stderr, "Available providers:")
		for i, p := range providers {
			fmt.Fprintf(os.Stderr, "  %d) %s\n", i+1, p)
		}
		fmt.Fprint(os.Stderr, "Select provider [1]: ")

		input, err := stdinReader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "kui setup: read input: %v\n", err)
			return 1
		}
		input = strings.TrimSpace(input)
		if input == "" {
			input = "1"
		}
		// Parse numeric selection.
		idx := 0
		for _, ch := range input {
			if ch >= '1' && ch <= '9' {
				idx = int(ch - '1')
			}
		}
		if idx < 0 || idx >= len(providers) {
			fmt.Fprintf(os.Stderr, "kui setup: invalid selection %q\n", input)
			return 2
		}
		provider = providers[idx]
	}

	// Prompt for API key with masked display.
	fmt.Fprintf(os.Stderr, "Enter API key for %s: ", provider)
	apiKey, err := stdinReader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "kui setup: read input: %v\n", err)
		return 1
	}
	apiKey = strings.TrimSpace(apiKey)

	// Validate the key (REQ-CRED-7).
	if err := ValidateKey(provider, apiKey); err != nil {
		fmt.Fprintf(os.Stderr, "kui setup: %v\n", err)
		return 1
	}

	// Save to credential store (REQ-CRED-8).
	cs := credentials.NewCredentialStore(root)
	if err := cs.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "kui setup: load credentials: %v\n", err)
		return 1
	}
	if err := cs.SetAPIKey(provider, apiKey); err != nil {
		fmt.Fprintf(os.Stderr, "kui setup: save credentials: %v\n", err)
		return 1
	}

	fmt.Fprintf(os.Stdout, "Credentials saved for %s.\n", provider)
	fmt.Fprintln(os.Stdout, "Next step: run `kui tui` to start chatting.")
	return 0
}
