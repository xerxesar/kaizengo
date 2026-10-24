// Command kaizen is the native runtime CLI: a thin GraphQL client that signs in
// with email/password, keeps the session cookie, and sends lines to the
// server's platformShell mutation (same grammar as the web terminal).
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"kaizengo/internal/shell"
)

const usage = `kaizen — KaizenGo runtime CLI (GraphQL)

Usage:
  kaizen login [--email E] [--password P]   Sign in and store the session
  kaizen logout                             Sign out and forget the session
  kaizen <command> [args…]                  Run one shell line, e.g. kaizen query crm.leads
  kaizen                                    Interactive prompt (reads lines from stdin)

Flags (before the command):
  --url URL     Server base URL (env KAIZEN_URL, default http://localhost:8080)
  --db NAME     Database to select (env KAIZEN_DB)
  -f FILE       Append FILE as a heredoc body (- for stdin), e.g. kaizen -f crm.yaml domain apply
  --json        Print the structured data payload instead of the text output

Run "kaizen help" for the server's command list.
`

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(argv []string) int {
	fs := flag.NewFlagSet("kaizen", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	url := fs.String("url", envOr("KAIZEN_URL", "http://localhost:8080"), "server base URL")
	db := fs.String("db", os.Getenv("KAIZEN_DB"), "database to select")
	file := fs.String("f", "", "heredoc body file (- for stdin)")
	asJSON := fs.Bool("json", false, "print data payload as JSON")
	if err := fs.Parse(argv); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	args := fs.Args()

	c, err := newClient(strings.TrimRight(*url, "/"), *db)
	if err != nil {
		fmt.Fprintln(os.Stderr, "kaizen:", err)
		return 1
	}

	if len(args) > 0 {
		switch args[0] {
		case "login":
			return cmdLogin(c, args[1:])
		case "logout":
			if err := c.logout(); err != nil {
				fmt.Fprintln(os.Stderr, "kaizen:", err)
				return 1
			}
			fmt.Println("signed out")
			return 0
		}
	}

	if len(args) == 0 && *file == "" {
		return repl(c, *asJSON)
	}

	line := joinArgs(args)
	if *file != "" {
		body, err := readBody(*file)
		if err != nil {
			fmt.Fprintln(os.Stderr, "kaizen:", err)
			return 1
		}
		line = withHeredoc(line, body)
	}
	return exec(c, line, *asJSON)
}

func cmdLogin(c *client, argv []string) int {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	email := fs.String("email", os.Getenv("KAIZEN_EMAIL"), "account email")
	password := fs.String("password", os.Getenv("KAIZEN_PASSWORD"), "account password (prompted when empty)")
	if err := fs.Parse(argv); err != nil {
		return 2
	}
	in := bufio.NewReader(os.Stdin)
	if *email == "" {
		fmt.Fprint(os.Stderr, "email: ")
		line, _ := in.ReadString('\n')
		*email = strings.TrimSpace(line)
	}
	if *password == "" {
		fmt.Fprint(os.Stderr, "password: ")
		if term.IsTerminal(int(os.Stdin.Fd())) {
			raw, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Fprintln(os.Stderr)
			if err != nil {
				fmt.Fprintln(os.Stderr, "kaizen:", err)
				return 1
			}
			*password = string(raw)
		} else {
			line, _ := in.ReadString('\n')
			*password = strings.TrimRight(line, "\r\n")
		}
	}
	name, err := c.login(*email, *password)
	if err != nil {
		fmt.Fprintln(os.Stderr, "kaizen:", err)
		return 1
	}
	fmt.Printf("signed in as %s (%s)\n", name, c.base)
	return 0
}

func exec(c *client, line string, asJSON bool) int {
	res, err := c.shell(line)
	if err != nil {
		fmt.Fprintln(os.Stderr, "kaizen:", err)
		return 1
	}
	out := res.Output
	if asJSON && res.Data != "" {
		out = res.Data
	}
	if res.OK {
		if out != "" {
			fmt.Println(out)
		}
		return 0
	}
	fmt.Fprintln(os.Stderr, out)
	return 1
}

func repl(c *client, asJSON bool) int {
	interactive := term.IsTerminal(int(os.Stdin.Fd()))
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	status := 0
	var buf strings.Builder
	prompt := func() {
		if !interactive {
			return
		}
		if buf.Len() > 0 {
			fmt.Fprint(os.Stderr, "… ")
		} else {
			fmt.Fprint(os.Stderr, "kaizen› ")
		}
	}
	prompt()
	for sc.Scan() {
		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(sc.Text())
		input := buf.String()
		if shell.HeredocPending(input) {
			prompt()
			continue
		}
		buf.Reset()
		line := strings.TrimSpace(input)
		switch line {
		case "":
		case "exit", "quit":
			return status
		default:
			if exec(c, line, asJSON) != 0 {
				status = 1
			}
		}
		prompt()
	}
	if interactive {
		fmt.Fprintln(os.Stderr)
	}
	if buf.Len() > 0 {
		fmt.Fprintln(os.Stderr, "kaizen: unterminated heredoc")
		return 1
	}
	return status
}

func joinArgs(args []string) string {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = shell.Quote(a)
	}
	return strings.Join(parts, " ")
}

func readBody(path string) (string, error) {
	if path == "-" {
		raw, err := io.ReadAll(os.Stdin)
		return string(raw), err
	}
	raw, err := os.ReadFile(path)
	return string(raw), err
}

// withHeredoc appends body as a heredoc with a terminator that does not occur in it.
func withHeredoc(line, body string) string {
	tag := "EOF"
	for i := 0; strings.Contains(body, tag); i++ {
		tag = fmt.Sprintf("EOF%d", i)
	}
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return line + " <<" + tag + "\n" + body + tag
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
