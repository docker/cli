package command

import (
	"github.com/spf13/pflag"
)

//AddQuietFlag Hides Output
func AddQuietFlag(fs *pflag.FlagSet, v *bool) {
	fs.BoolVar(v, "quiet", *v, "Suppress Output")
}
