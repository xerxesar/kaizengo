package shell

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// split tokenizes one command. It supports 'single' and "double" quotes,
// bash-like backslash escapes and line continuation, newlines inside open
// quotes, and one heredoc when the first line ends with <<TAG:
//
//	domain apply --migrate="UPDATE …" <<EOF
//	name: crm
//	EOF
//
// The heredoc body (without the terminator) becomes a single token. When the
// first line opens a heredoc, later newlines belong to that body — so flag
// values like --migrate= must be fully closed on the first line.
func split(input string) ([]string, error) {
	first, rest, _ := strings.Cut(input, "\n")
	if tag, ok := heredocTag(first); ok {
		body, err := heredocBody(rest, tag)
		if err != nil {
			return nil, err
		}
		idx := strings.LastIndex(first, "<<")
		tokens, err := tokenize(first[:idx])
		if err != nil {
			return nil, err
		}
		return append(tokens, body), nil
	}
	return tokenize(input)
}

// HeredocPending reports whether input opens a heredoc on its first line that
// no later line has closed yet.
func HeredocPending(input string) bool {
	first, rest, _ := strings.Cut(input, "\n")
	tag, ok := heredocTag(first)
	if !ok {
		return false
	}
	_, err := heredocBody(rest, tag)
	return err != nil
}

// InputPending reports whether interactive clients should keep reading:
// open heredoc, unclosed quote, or a line ending in an unescaped backslash.
func InputPending(input string) bool {
	if HeredocPending(input) {
		return true
	}
	first, _, _ := strings.Cut(input, "\n")
	if _, ok := heredocTag(first); ok {
		return false // heredoc is closed (otherwise HeredocPending)
	}
	return tokenizeIncomplete(input)
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

func tokenize(s string) ([]string, error) {
	tokens, quote, escaped, err := scan(s, true)
	if err != nil {
		return nil, err
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated %c quote", quote)
	}
	if escaped {
		return nil, fmt.Errorf("trailing backslash")
	}
	return tokens, nil
}

func tokenizeIncomplete(s string) bool {
	_, quote, escaped, err := scan(s, false)
	if err != nil {
		return false
	}
	return quote != 0 || escaped
}

// scan tokenizes s. If collect is false, tokens are discarded (pending check).
// Returns the final quote state and whether a backslash escape is open.
func scan(s string, collect bool) (tokens []string, quote rune, escaped bool, err error) {
	var (
		cur     strings.Builder
		inToken bool
	)
	flush := func() {
		if inToken {
			if collect {
				tokens = append(tokens, cur.String())
			}
			cur.Reset()
			inToken = false
		}
	}
	write := func(r rune) {
		if collect {
			cur.WriteRune(r)
		}
		inToken = true
	}
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size
		switch {
		case escaped:
			escaped = false
			if r == '\n' {
				// bash: \ + newline is line continuation (elided), in or out of "
				continue
			}
			write(r)
		case quote == '\'':
			if r == '\'' {
				quote = 0
			} else {
				write(r)
			}
		case quote == '"':
			switch r {
			case '"':
				quote = 0
			case '\\':
				escaped = true
			default:
				write(r)
			}
		case r == '\\':
			escaped = true
		case r == '\'' || r == '"':
			quote = r
			inToken = true
		case r == ' ' || r == '\t' || r == '\r' || r == '\n':
			flush()
		default:
			write(r)
		}
	}
	flush()
	return tokens, quote, escaped, nil
}
