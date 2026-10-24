package shell

import (
	"fmt"
	"strings"
)

// split tokenizes one command line. It supports 'single' and "double" quotes,
// backslash escapes outside single quotes, and one heredoc per line:
//
//	domain apply <<EOF
//	name: crm
//	EOF
//
// The heredoc body (without the terminator) becomes a single token.
func split(input string) ([]string, error) {
	first, rest, _ := strings.Cut(input, "\n")
	if tag, ok := heredocTag(first); ok {
		body, err := heredocBody(rest, tag)
		if err != nil {
			return nil, err
		}
		idx := strings.LastIndex(first, "<<")
		tokens, err := splitLine(first[:idx])
		if err != nil {
			return nil, err
		}
		return append(tokens, body), nil
	}
	if strings.TrimSpace(rest) != "" {
		return nil, fmt.Errorf("multi-line input needs a heredoc (<<EOF … EOF)")
	}
	return splitLine(first)
}

// HeredocPending reports whether input opens a heredoc on its first line that
// no later line has closed yet. Interactive clients keep reading while true.
func HeredocPending(input string) bool {
	first, rest, _ := strings.Cut(input, "\n")
	tag, ok := heredocTag(first)
	if !ok {
		return false
	}
	_, err := heredocBody(rest, tag)
	return err != nil
}

// Quote returns arg as a single token for split (used when rebuilding a line
// from already-split argv).
func Quote(arg string) string {
	if arg != "" && !strings.ContainsAny(arg, " \t\r\n'\"\\<") {
		return arg
	}
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return `"` + r.Replace(arg) + `"`
}

func heredocTag(line string) (string, bool) {
	idx := strings.LastIndex(line, "<<")
	if idx < 0 {
		return "", false
	}
	tag := strings.Trim(strings.TrimSpace(line[idx+2:]), `'"`)
	if tag == "" || strings.ContainsAny(tag, " \t") {
		return "", false
	}
	return tag, true
}

func heredocBody(rest, tag string) (string, error) {
	lines := strings.Split(rest, "\n")
	for i, l := range lines {
		if strings.TrimSpace(l) == tag {
			return strings.Join(lines[:i], "\n") + "\n", nil
		}
	}
	return "", fmt.Errorf("heredoc: missing terminator %q", tag)
}

func splitLine(line string) ([]string, error) {
	var (
		tokens  []string
		cur     strings.Builder
		inToken bool
		quote   rune
		escaped bool
	)
	flush := func() {
		if inToken {
			tokens = append(tokens, cur.String())
			cur.Reset()
			inToken = false
		}
	}
	for _, r := range line {
		switch {
		case escaped:
			cur.WriteRune(r)
			escaped = false
		case quote == '\'':
			if r == '\'' {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case quote == '"':
			switch r {
			case '"':
				quote = 0
			case '\\':
				escaped = true
			default:
				cur.WriteRune(r)
			}
		case r == '\\':
			escaped = true
			inToken = true
		case r == '\'' || r == '"':
			quote = r
			inToken = true
		case r == ' ' || r == '\t' || r == '\r':
			flush()
		default:
			cur.WriteRune(r)
			inToken = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated %c quote", quote)
	}
	flush()
	return tokens, nil
}
