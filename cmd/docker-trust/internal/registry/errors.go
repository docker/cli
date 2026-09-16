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
