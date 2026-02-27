package repl

import (
	"strings"

	"github.com/barun-bash/human/internal/cmdutil"
)

func cmdExplain(r *REPL, args []string) {
	topic := ""
	if len(args) > 0 {
		topic = strings.Join(args, " ")
	}
	cmdutil.RunExplain(r.out, topic)
}
