// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.24

package registry

import (
	"fmt"
)

func invalidParam(err error) error {
	return invalidParameterErr{err}
}

func invalidParamf(format string, args ...any) error {
	return invalidParameterErr{fmt.Errorf(format, args...)}
}

type invalidParameterErr struct{ error }

func (invalidParameterErr) InvalidParameter() {}

func (e invalidParameterErr) Unwrap() error {
	return e.error
}
