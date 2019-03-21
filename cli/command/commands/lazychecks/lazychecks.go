package lazychecks

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/cli/cli/command"
	"github.com/spf13/pflag"
)

var lazyChecks []LazyCheck

const lazyCheckAnnotation = "lazy-checks"

// LazyCheck is a callback that is called lazily to know if a command / flag should be enabled
type LazyCheck func(clientInfo command.ClientInfo, serverInfo command.ServerInfo, clientVersion string) error

// AddLazyFlagCheck adds a LazyCheck on a flag
func AddLazyFlagCheck(flagset *pflag.FlagSet, name string, check LazyCheck) {
	index := len(lazyChecks)
	lazyChecks = append(lazyChecks, check)
	f := flagset.Lookup(name)
	if f == nil {
		return
	}
	if f.Annotations == nil {
		f.Annotations = map[string][]string{}
	}
	f.Annotations[lazyCheckAnnotation] = append(f.Annotations[lazyCheckAnnotation], fmt.Sprintf("%d", index))
}

// EvaluateFlagLazyChacks evaluates the lazy checks associated with a flag depending on client/server info
func EvaluateFlagLazyChacks(flag *pflag.Flag, clientInfo command.ClientInfo, serverInfo command.ServerInfo, clientVersion string) error {
	var errs []string
	for _, indexStr := range flag.Annotations[lazyCheckAnnotation] {
		index, err := strconv.ParseInt(indexStr, 10, 32)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		if err := lazyChecks[int(index)](clientInfo, serverInfo, clientVersion); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "\n"))
}
