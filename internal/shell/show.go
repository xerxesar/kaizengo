package shell

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// domainShowNamed prints one keyed list item from the tip as YAML.
func domainShowNamed(s *session, args []string, listKey, nameKey, kind string) (string, any, error) {
	usage := fmt.Errorf("usage: domain show-%s <app> <%s>", kind, nameKey)
	if len(args) != 2 {
		return "", nil, usage
	}
	d, err := fetchDomain(s, args[0])
	if err != nil {
		return "", nil, err
	}
	body, err := extractNamedYAML(fmt.Sprint(d["spec"]), listKey, nameKey, args[1])
	if err != nil {
		return "", nil, err
	}
	return strings.TrimRight(body, "\n"), map[string]any{"app": args[0], kind: args[1], "yaml": body}, nil
}

func domainShowPolicy(s *session, args []string) (string, any, error) {
	if len(args) != 2 {
		return "", nil, fmt.Errorf("usage: domain show-policy <app> <name>")
	}
	d, err := fetchDomain(s, args[0])
	if err != nil {
		return "", nil, err
	}
	body, err := extractPolicyYAML(fmt.Sprint(d["spec"]), args[1])
	if err != nil {
		return "", nil, err
	}
	return strings.TrimRight(body, "\n"), map[string]any{"app": args[0], "policy": args[1], "yaml": body}, nil
}

// extractNamedYAML returns the YAML for one item in a top-level sequence (views, menus, …).
func extractNamedYAML(doc, listKey, nameKey, name string) (string, error) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(doc), &root); err != nil {
		return "", fmt.Errorf("parse domain: %w", err)
	}
	if len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		return "", fmt.Errorf("domain document is not a mapping")
	}
	list := mapValue(root.Content[0], listKey)
	if list == nil || list.Kind != yaml.SequenceNode {
		return "", fmt.Errorf("domain has no %s", listKey)
	}
	for _, item := range list.Content {
		if n := mapValue(item, nameKey); n != nil && n.Value == name {
			return encodeNode(item)
		}
	}
	return "", fmt.Errorf("%s %q not found", strings.TrimSuffix(listKey, "s"), name)
}

func extractPolicyYAML(doc, name string) (string, error) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(doc), &root); err != nil {
		return "", fmt.Errorf("parse domain: %w", err)
	}
	if len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		return "", fmt.Errorf("domain document is not a mapping")
	}
	sec := mapValue(root.Content[0], "security")
	if sec == nil || sec.Kind != yaml.MappingNode {
		return "", fmt.Errorf("domain has no security block")
	}
	entries := mapValue(sec, "entries")
	if entries == nil || entries.Kind != yaml.SequenceNode {
		return "", fmt.Errorf("domain has no security.entries")
	}
	for _, item := range entries.Content {
		if n := mapValue(item, "name"); n != nil && n.Value == name {
			return encodeNode(item)
		}
	}
	return "", fmt.Errorf("policy %q not found", name)
}

func encodeNode(n *yaml.Node) (string, error) {
	var buf strings.Builder
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(n); err != nil {
		_ = enc.Close()
		return "", err
	}
	_ = enc.Close()
	return buf.String(), nil
}

// listNamedKeys returns name/id values under a tip list for completions.
func listNamedKeys(doc, listKey, nameKey string) ([]string, error) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(doc), &root); err != nil {
		return nil, err
	}
	if len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("domain document is not a mapping")
	}
	list := mapValue(root.Content[0], listKey)
	if list == nil || list.Kind != yaml.SequenceNode {
		return nil, nil
	}
	out := make([]string, 0, len(list.Content))
	for _, item := range list.Content {
		if n := mapValue(item, nameKey); n != nil && n.Value != "" {
			out = append(out, n.Value)
		}
	}
	return out, nil
}

func tedSoft(command, seed string) (string, any, error) {
	payload := map[string]any{
		"ted":      true,
		"command":  command,
		"seed":     seed,
		"language": "yaml",
	}
	var b strings.Builder
	b.WriteString("open terminal editor (-ted) — edit YAML then save to apply\n")
	b.WriteString("(native CLI: paste a heredoc, or use the web terminal)\n")
	if strings.TrimSpace(seed) != "" {
		b.WriteString("---\n")
		b.WriteString(strings.TrimRight(seed, "\n"))
		b.WriteString("\n---")
	}
	return b.String(), payload, fmt.Errorf("ted required")
}

func takeTedFlag(args []string) ([]string, bool) {
	args, a := dropFlag(args, "-ted")
	args, b := dropFlag(args, "--ted")
	return args, a || b
}
