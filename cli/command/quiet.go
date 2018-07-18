package command

import (
	"github.com/spf13/pflag"
)

// AddTrustSigningFlags Hides Progress Bar
func AddQuietFlag(fs *pflag.FlagSet, v *bool) {
	fs.BoolVar(v, "quiet", *v, "Supress Output")
}
