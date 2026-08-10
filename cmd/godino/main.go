package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printRootHelp()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "new-app":
		if err := runNewApp(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "version", "-v", "--version":
		fmt.Println(version)
	case "help", "-h", "--help":
		printRootHelp()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		printRootHelp()
		os.Exit(1)
	}
}

func printRootHelp() {
	fmt.Fprintf(os.Stderr, `kaizengo %s — project tooling

Usage:
  kaizengo <command> [flags]

Commands:
  new-app    Bootstrap a new app under apps/
  version    Print CLI version
  help       Show this help

Examples:
  kaizengo new-app notes
  kaizengo new-app notes --type svelte
  kaizengo new-app notes --type svelte --with-graphql

See docs/ for full guides.
`, version)
}

func runNewApp(args []string) error {
	fs := flag.NewFlagSet("new-app", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	typ := fs.String("type", "vanilla", "app UI type: vanilla | svelte")
	title := fs.String("title", "", "Apps menu title (default: name)")
	summary := fs.String("summary", "", "manifest summary")
	withGQL := fs.Bool("with-graphql", false, "register sample GraphQL query/mutation")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage:
  kaizengo new-app <name> [flags]
  kaizengo new-app [flags] <name>

Flags:
`)
		fs.PrintDefaults()
	}

	var name string
	var flagArgs []string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		name = args[0]
		flagArgs = args[1:]
	} else {
		flagArgs = args
	}

	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	switch {
	case name == "" && fs.NArg() == 1:
		name = fs.Arg(0)
	case name != "" && fs.NArg() == 0:
		// ok
	default:
		fs.Usage()
		return fmt.Errorf("expected app name")
	}

	return createApp(AppOptions{
		Name:        name,
		Type:        *typ,
		Title:       *title,
		Summary:     *summary,
		WithGraphQL: *withGQL,
	})
}
