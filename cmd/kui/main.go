// Command kui runs the agent loop once for a prompt given as command-line
// arguments (REQ-CLI-1). Exit codes follow D13: 0 success, 1 runtime failure,
// 2 usage error. The final answer goes to stdout; errors and usage go to
// stderr (REQ-CLI-2).
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/biggs-100/kui/internal/adapters/providers/openai"
	"github.com/biggs-100/kui/internal/adapters/tools"
	"github.com/biggs-100/kui/internal/core"
)

// maxIterations bounds the provider calls per run so a misbehaving provider
// cannot loop forever (D7).
const maxIterations = 10

const usage = `kui PROMPT...

Runs the agent loop once and prints the final answer to stdout.

Environment:
  OPENAI_API_KEY   required; API key for the chat-completions endpoint
  OPENAI_BASE_URL  optional; defaults to https://api.openai.com/v1
  OPENAI_MODEL     optional; defaults to gpt-4o-mini
`

func main() {
	os.Exit(run(os.Args[1:]))
}

// run wires the provider, the default tool set, and the agent loop, and
// returns the process exit code (D13).
func run(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}

	client, err := openai.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "kui: %v\n", err)
		return 1
	}

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "kui: determine working directory: %v\n", err)
		return 1
	}

	agent := &core.Agent{
		Provider:      client,
		Tools:         core.NewRegistry(),
		MaxIterations: maxIterations,
	}
	for _, tool := range tools.Default(root, 0) {
		if err := agent.Tools.Register(tool); err != nil {
			fmt.Fprintf(os.Stderr, "kui: register tool: %v\n", err)
			return 1
		}
	}

	answer, err := agent.Run(context.Background(), strings.Join(args, " "))
	if err != nil {
		fmt.Fprintf(os.Stderr, "kui: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintln(os.Stdout, answer)
	return 0
}
