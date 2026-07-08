package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/aloki-alok/mcpify/internal/tool"
	"github.com/aloki-alok/mcpify/internal/ui"
)

// ls previews the tools a spec would expose, without starting a server. It is
// the "show me what I'd get" command and prints a readable table to stdout.
func ls(args []string) int {
	o, err := parse(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mcpify:", err)
		return 2
	}
	doc, _, err := loadSpec(o.spec)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mcpify:", err)
		return 1
	}
	defs := tool.Defs(doc, o.readOnly)

	title := doc.Title
	if title == "" {
		title = o.spec
	}
	server := ""
	if len(doc.Servers) > 0 {
		server = doc.Servers[0].URL
	}
	renderTools(ui.For(os.Stdout), os.Stdout, title, server, defs)
	return 0
}

// firstLine returns the first description line, skipping the leading
// "METHOD /path" anchor line that the tool builder prepends.
func firstLine(desc, anchor string) string {
	for _, line := range strings.Split(desc, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == anchor {
			continue
		}
		return line
	}
	return ""
}
