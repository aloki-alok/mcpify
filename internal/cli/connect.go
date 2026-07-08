package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/aloki-alok/mcpify/internal/server"
	"github.com/aloki-alok/mcpify/internal/tool"
	"github.com/aloki-alok/mcpify/internal/ui"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// interact is the human-facing flow when someone runs mcpify against a spec in a
// terminal (rather than a client spawning it over stdio). It offers the few
// things a person actually wants, with running a server as the one-keypress
// default, so they do not have to reason about transports or ports.
func interact(o options, srv *sdk.Server, name, base string, defs []tool.Def) int {
	s := ui.For(os.Stdout)
	header(s, name, base, len(defs))

	in := bufio.NewReader(os.Stdin)
	for {
		fmt.Println()
		fmt.Println(s.Accent("how do you want to use it?"))
		fmt.Println("  " + s.Bold("1") + ") run a local server and get a connect URL  " + s.Dim("(recommended)"))
		fmt.Println("  " + s.Bold("2") + ") print a client config to paste")
		fmt.Println("  " + s.Bold("3") + ") list the tools")
		fmt.Println("  " + s.Bold("4") + ") quit")
		fmt.Print("\n" + s.Dim("> [1] "))

		line, err := in.ReadString('\n')
		if err != nil { // EOF (ctrl-D): treat as quit
			fmt.Println()
			return 0
		}
		switch strings.TrimSpace(line) {
		case "", "1":
			return serveHTTP(srv, freePort(), name, base, len(defs))
		case "2":
			printClientConfig(s, name, o)
			return 0
		case "3":
			renderTools(s, os.Stdout, name, base, defs)
		case "4", "q", "quit":
			return 0
		default:
			fmt.Println(s.Dim("  pick 1-4"))
		}
	}
}

// header prints the one-line identity shared by every entry point.
func header(s ui.Style, name, base string, n int) {
	fmt.Printf("%s %s\n", s.Accent("mcpify ·"), s.Bold(name))
	fmt.Printf("%s\n", s.Dim(fmt.Sprintf("%d tools → %s", n, base)))
}

// serveHTTP starts the streamable-HTTP server at addr, printing the endpoint and
// ready-to-run connect commands first. It blocks until interrupted.
func serveHTTP(srv *sdk.Server, addr, name, base string, n int) int {
	s := ui.For(os.Stdout)
	endpoint := endpointURL(addr)

	fmt.Println()
	fmt.Printf("  %s   %s\n", s.Dim("MCP endpoint"), s.Bold(endpoint))
	fmt.Println()
	fmt.Println("  " + s.Dim("connect a client:"))
	fmt.Printf("    %s %s\n", s.Dim("mctop"), endpoint)
	fmt.Printf("    %s\n", s.Dim("claude mcp add --transport http "+slug(name)+" "+endpoint))
	fmt.Printf("    %s\n", s.Dim("Cursor / Claude Desktop: add a streamable-HTTP server at that URL"))
	fmt.Printf("\n  %s\n", s.Dim("serving · Ctrl-C to stop"))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := server.RunHTTP(ctx, srv, addr, infoPage(name, base, n, endpoint)); err != nil && err != context.Canceled {
		fmt.Fprintln(os.Stderr, "mcpify:", err)
		return 1
	}
	return 0
}

// printClientConfig prints a copy-paste MCP client config that launches mcpify
// over stdio with the same spec and flags the user supplied.
func printClientConfig(s ui.Style, name string, o options) {
	args := configArgs(o)
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = `"` + a + `"`
	}
	fmt.Println()
	fmt.Println(s.Dim("add this to your MCP client config:"))
	fmt.Println()
	fmt.Printf(`{
  "mcpServers": {
    %q: {
      "command": "mcpify",
      "args": [%s]
    }
  }
}
`, slug(name), strings.Join(quoted, ", "))
}

// configArgs reconstructs the argument list a client should launch mcpify with,
// so the printed config behaves exactly like the current invocation.
func configArgs(o options) []string {
	var args []string
	if o.base != "" {
		args = append(args, "--base", o.base)
	}
	for k, v := range o.headers {
		args = append(args, "-H", k+": "+v)
	}
	if o.readOnly {
		args = append(args, "--read-only")
	}
	return append(args, o.spec)
}

// renderTools prints the readable tool table shared by `ls` and the interactive
// menu: each tool's name, method, path, and first description line.
func renderTools(s ui.Style, w io.Writer, title, base string, defs []tool.Def) {
	fmt.Fprintf(w, "%s  %s\n", s.Bold(title), s.Dim(fmt.Sprintf("%d tools", len(defs))))
	if base != "" {
		fmt.Fprintln(w, s.Dim("server: "+base))
	}
	fmt.Fprintln(w)

	width := 0
	for _, d := range defs {
		if n := len(d.Name); n > width {
			width = n
		}
	}
	for _, d := range defs {
		verb := fmt.Sprintf("%-6s", d.Method())
		pad := strings.Repeat(" ", width-len(d.Name)+2)
		fmt.Fprintf(w, "  %s%s%s %s\n", s.Accent(d.Name), pad, s.Dim(verb), d.Path())
		if line := firstLine(d.Description, d.Method()+" "+d.Path()); line != "" {
			fmt.Fprintf(w, "  %s%s\n", strings.Repeat(" ", width+2), s.Dim(line))
		}
	}
}

// infoPage is the plain-text page served to a browser that opens the server
// root, so a curious human gets an explanation instead of a protocol error.
func infoPage(name, base string, n int, endpoint string) string {
	return fmt.Sprintf(`mcpify · %s
%d tools, proxying %s

MCP endpoint (streamable HTTP): %s

connect:
  mctop %s
  claude mcp add --transport http %s %s
`, name, n, base, endpoint, endpoint, slug(name), endpoint)
}

// endpointURL turns a listen address into the URL a client connects to,
// normalising an empty or wildcard host to localhost and adding the /mcp path.
func endpointURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		host, port = "", strings.TrimPrefix(addr, ":")
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, port) + "/mcp"
}

// freePort asks the OS for an open port on the loopback interface.
func freePort() string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "127.0.0.1:8080"
	}
	addr := ln.Addr().String()
	ln.Close()
	return addr
}

// isPortFree reports whether addr can be bound right now.
func isPortFree(addr string) bool {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// nextFreePort returns the first bindable address at or after addr's port,
// keeping the same host, so a busy --http port has an obvious alternative.
func nextFreePort(addr string) string {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return freePort()
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return freePort()
	}
	for p := port + 1; p < port+50; p++ {
		cand := net.JoinHostPort(host, strconv.Itoa(p))
		if isPortFree(cand) {
			return cand
		}
	}
	return freePort()
}

// slug turns a spec title into a short identifier usable as an MCP server name.
func slug(title string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "mcpify"
	}
	return out
}
