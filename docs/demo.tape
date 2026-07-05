# Regenerates docs/demo.gif with charm vhs (https://github.com/charmbracelet/vhs).
#
#   go build -o /tmp/mcpify .
#   alias mcpify=/tmp/mcpify
#   vhs docs/demo.tape
#
# On a headless or root environment set VHS_NO_SANDBOX=true first.

Output "docs/demo.gif"

Set Shell "bash"
Set FontSize 20
Set Width 1000
Set Height 380
Set Padding 24
Set Theme "Rose Pine"

Hide
Type "export PS1='$ ' && alias mcpify=/tmp/mcpify && clear"
Enter
Show

Sleep 800ms
Type "mcpify ls examples/petstore.yaml"
Sleep 700ms
Enter
Sleep 3.5s
