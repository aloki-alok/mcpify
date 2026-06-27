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

	s := ui.For(os.Stdout)
	title := doc.Title
	if title == "" {
		title = o.spec
	}
	fmt.Printf("%s  %s\n", s.Bold(title), s.Dim(fmt.Sprintf("%d tools", len(defs))))
	if len(doc.Servers) > 0 {
		fmt.Println(s.Dim("server: " + doc.Servers[0].URL))
	}
	fmt.Println()

	width := 0
	for _, d := range defs {
		if n := len(d.Name); n > width {
			width = n
		}
	}
	for _, d := range defs {
		verb := fmt.Sprintf("%-6s", d.Method())
		pad := strings.Repeat(" ", width-len(d.Name)+2)
		fmt.Printf("  %s%s%s %s\n",
			s.Accent(d.Name), pad, s.Dim(verb), d.Path())
		if line := firstLine(d.Description, d.Method()+" "+d.Path()); line != "" {
			fmt.Printf("  %s%s\n", strings.Repeat(" ", width+2), s.Dim(line))
		}
	}
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
