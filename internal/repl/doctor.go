package repl

import (
	"github.com/barun-bash/human/internal/cmdutil"
)

func cmdDoctor(r *REPL, args []string) {
	cmdutil.RunDoctor(r.out)
}
