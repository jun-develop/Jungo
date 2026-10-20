package console

import (
	"context"
	"fmt"
	"os"
	"sort"

	"go.uber.org/fx"

	"github.com/jun-develop/junkit/console"
)

// Params defines dependencies for command dispatch.
type Params struct {
	fx.In

	Commands   []Command `group:"commands"`
	Shutdowner fx.Shutdowner
}

// RunSelected registers an Fx OnStart hook to dispatch the CLI command and shut down. The command
// context outlives the OnStart deadline and is cancelled on stop (SIGINT/SIGTERM or Shutdown).
func RunSelected(p Params, lc fx.Lifecycle) {
	ctx, cancel := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go dispatch(ctx, p)
			return nil
		},
		OnStop: func(context.Context) error {
			cancel()
			return nil
		},
	})
}

// dispatch executes the selected command and shuts down the application.
func dispatch(ctx context.Context, p Params) {
	defer func() { _ = p.Shutdowner.Shutdown() }()

	if len(os.Args) < 2 {
		printAvailable(p.Commands)
		return
	}

	name, args := os.Args[1], os.Args[2:]

	for _, cmd := range p.Commands {
		if cmd.Signature() == name {
			if err := cmd.Run(ctx, args); err != nil {
				console.Fatalf("command %q failed: %v", name, err)
			}

			return
		}
	}

	console.Fatalf("unknown command %q (run without arguments to list available commands)", name)
}

// printAvailable lists every registered command's signature.
func printAvailable(cmds []Command) {
	signatures := make([]string, 0, len(cmds))
	for _, c := range cmds {
		signatures = append(signatures, c.Signature())
	}

	sort.Strings(signatures)

	fmt.Println("Available commands:")
	for _, s := range signatures {
		fmt.Printf("  %s\n", s)
	}
}
