package repl

import (
	"strings"

	"github.com/barun-bash/human/internal/cmdutil"
)

func cmdSyntax(r *REPL, args []string) {
	section := ""
	search := ""

	for i, arg := range args {
		if arg == "--search" && i+1 < len(args) {
			search = strings.Join(args[i+1:], " ")
			break
		}
		if section == "" {
			section = arg
		}
	}

	cmdutil.RunSyntax(r.out, section, search)
}
