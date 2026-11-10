package shell

import (
	"fmt"
	"strings"
)

func cmdMan(_ *session, args []string) (string, any, error) {
	if len(args) == 0 {
		return "", nil, fmt.Errorf("usage: man <command> [sub]\nTry: man help | man domain | man domain set-view")
	}
	topic := strings.Join(args, " ")
	if page, ok := manPages[topic]; ok {
		return page, nil, nil
	}
	if page, ok := manPages[args[0]]; ok && len(args) == 1 {
		return page, nil, nil
	}
	if c, ok := commands[args[0]]; ok && len(args) == 1 {
		return fmt.Sprintf("SYNOPSIS\n  %s\n\nDESCRIPTION\n  %s\n\nSee also: man %s (if documented), help %s", c.usage, c.summary, args[0], args[0]), nil, nil
	}
	return "", nil, fmt.Errorf("no manual entry for %q (try man help)", topic)
}

var manPages = map[string]string{
	"help": `SYNOPSIS
  help [command]

DESCRIPTION
  Lists runtime CLI commands in groups, or prints one command's usage line.
  For longer docs use man <command>.

EXAMPLES
  help
  help domain
  man domain set-view`,

	"man": `SYNOPSIS
  man <command> [domain-sub]

DESCRIPTION
  Prints a short manual page for a shell command or domain subcommand.

EXAMPLES
  man domain
  man domain set-view
  man query`,

	"domain": `SYNOPSIS
  domain <subcommand> …

DESCRIPTION
  Inspect and change app domain tips (git-like revisions). Most mutators append
  a new tip snapshot and hot-mount the app. Data-touching field changes may
  require a migrate (Settings → Domains or --migrate-pre / --migrate / --no-migrate).

SUBCOMMANDS
  list, show, show-view, show-menu, show-query, show-command, show-model, show-policy
  history, conflicts, rebase, diff, apply
  add|set|remove -field|-index|-view|-menu|-query|-command
  revert, delete, export, import

FLAGS (common)
  --dry-run          Validate without committing
  --force            Destructive drops / take-code on rebase
  -ted, --ted        Open the web terminal vim editor with the current YAML seed
  --migrate-pre=SQL  Run before additive DDL (e.g. RENAME)
  --migrate=SQL      Post-phase SQL (alias --migrate-post)
  --migrate-go name  Compiled Go post migration
  --no-migrate       Accept empty/zero new columns without SQL

EXAMPLES
  domain show hellospec
  domain show-view hellospec GreetingList
  domain set-view hellospec GreetingList -ted
  domain add-field hellospec.greeting scratch string --no-migrate

See also: man domain set-view, man domain show-view, Settings → Domains.`,

	"domain show": `SYNOPSIS
  domain show <app> [--rev N]

DESCRIPTION
  Print the tip domain YAML (or a stored revision).`,

	"domain show-view": `SYNOPSIS
  domain show-view <app> <name>

DESCRIPTION
  Print one view mapping from the tip as YAML.

EXAMPLES
  domain show-view hellospec GreetingList`,

	"domain show-menu": `SYNOPSIS
  domain show-menu <app> <id>

DESCRIPTION
  Print one menu entry from the tip.`,

	"domain show-query": `SYNOPSIS
  domain show-query <app> <name>

DESCRIPTION
  Print one query definition from the tip.`,

	"domain show-command": `SYNOPSIS
  domain show-command <app> <name>

DESCRIPTION
  Print one command definition from the tip.`,

	"domain show-model": `SYNOPSIS
  domain show-model <app> <name>

DESCRIPTION
  Print one model (fields, indexes) from the tip.`,

	"domain show-policy": `SYNOPSIS
  domain show-policy <app> <name>

DESCRIPTION
  Print one security.entries policy by name.`,

	"domain set-view": `SYNOPSIS
  domain set-view <app> <name> [-ted | <<EOF … EOF]

DESCRIPTION
  Replace or create a view in the tip. With -ted (web terminal), opens a vim
  YAML editor seeded with the current view (or a stub). Save submits a heredoc apply.

EXAMPLES
  domain set-view hellospec GreetingList -ted
  domain set-view hellospec GreetingList <<EOF
  layout: column
  components:
    - id: list
      kind: collection
      model: greeting
      defaultView: table
  EOF`,

	"domain set-menu": `SYNOPSIS
  domain set-menu <app> <id> [-ted | flags | <<EOF … EOF]

DESCRIPTION
  Replace a menu. -ted opens the web terminal editor with the current menu YAML.`,

	"domain set-query": `SYNOPSIS
  domain set-query <app> <name> [-ted | <<EOF … EOF]

DESCRIPTION
  Replace a query definition. -ted opens the vim editor with the current seed.`,

	"domain set-command": `SYNOPSIS
  domain set-command <app> <name> [-ted | <<EOF … EOF]

DESCRIPTION
  Replace a command definition. -ted opens the vim editor with the current seed.`,

	"domain apply": `SYNOPSIS
  domain apply [--dry-run] [--force] [migrate flags] <<EOF … EOF
  domain apply <app> -ted

DESCRIPTION
  Validate and commit a full tip document. -ted seeds the editor from <app>'s tip.

FLAGS
  See man domain for migrate flags.`,

	"query": `SYNOPSIS
  query <app.op> [key=value …] [--fields a,b] [--all] [--json]

DESCRIPTION
  Run a GraphQL query from the platform catalog.

EXAMPLES
  query hellospec.greetings
  query crm.leads q=acme --fields title,amount`,

	"run": `SYNOPSIS
  run <app.op> [key=value …] [--fields a,b] [--all] [--json]

DESCRIPTION
  Run a GraphQL command (mutation) from the platform catalog.`,

	"describe": `SYNOPSIS
  describe <app.op>

DESCRIPTION
  Show GraphQL argument types and return fields for an operation.`,
}

func cmdHelp(_ *session, args []string) (string, any, error) {
	if len(args) > 0 {
		c, ok := commands[args[0]]
		if !ok {
			return "", nil, fmt.Errorf("unknown command %q (try man help)", args[0])
		}
		var b strings.Builder
		b.WriteString(c.usage)
		b.WriteString("\n  ")
		b.WriteString(c.summary)
		b.WriteString("\n\nFor details: man ")
		b.WriteString(args[0])
		return b.String(), nil, nil
	}
	groups := []struct {
		title string
		names []string
	}{
		{"Inspection", []string{"help", "man", "whoami", "apps", "ls", "describe", "plugins"}},
		{"Invoke", []string{"query", "run", "gql"}},
		{"Domain", []string{"domain"}},
	}
	var b strings.Builder
	for _, g := range groups {
		b.WriteString(g.title)
		b.WriteString("\n")
		rows := [][]string{}
		for _, name := range g.names {
			c, ok := commands[name]
			if !ok {
				continue
			}
			rows = append(rows, []string{name, c.summary})
		}
		b.WriteString(table([]string{"command", "summary"}, rows))
		b.WriteString("\n")
	}
	b.WriteString("Values: key=value (quote spaces: title=\"Big deal\"); lists: tags=a,b or tags='[\"a\",\"b\"]'.\n")
	b.WriteString("Multi-line: <<EOF … EOF. Web editor: domain set-view <app> <name> -ted\n")
	b.WriteString("More: man <command>  |  man domain  |  man domain set-view")
	return b.String(), nil, nil
}
