package errors

import "fmt"

// Wrapf takes an originating "cause" error and annotates
// it with text and the source file & line of the wrap point.
func Wrapf(err error, fields Fields, format string, args ...interface{}) *Wrapped {
	w := &Wrapped{
		Base:  NewBase(1, fmt.Sprintf(format, args...)),
		cause: err,
	}
	w.AddFields(fields)
	return w
}

// Wrap takes an originating "cause" error and annotates
// it just with the source file & line of the wrap point.
func Wrap(err error, fields Fields) *Wrapped {
	w := &Wrapped{
		Base:  NewBase(1, err.Error()),
		cause: err,
	}
	w.AddFields(fields)
	return w
}

// WithMessage takes an originating cause and text description,
// returning a wrapped error with the text and stack.
func WithMessage(err error, text string) *Wrapped {
	return &Wrapped{
		Base:  NewBase(1, text),
		cause: err,
	}
}

// WithStack takes an originating cause and returns
// a wrapped error that just records the stack.
func WithStack(err error) *Wrapped {
	return &Wrapped{
		Base:  NewBase(1, err.Error()),
		cause: err,
	}
}

// Wrapped provides a way to add additional context when passing back
// an error to a caller. It inherits the useful features from Base
// (call stacks, structured text & data).
type Wrapped struct {
	*Base
	cause error
}

func (w *Wrapped) Error() string {
	msg := textAndFields(w.Text, w.Fields)
	return msg + ": " + w.cause.Error()
}

// With allows adding additional structured data fields.
func (w *Wrapped) With(fields Fields) *Wrapped {
	w.AddFields(fields)
	return w
}

// Unwrap extracts any layered Wrapped errors inside of this one,
// returning the first non-Wrapped error found as the original cause.
func (w *Wrapped) Unwrap() (wraps []*Base, cause error) {
	for w != nil {
		cause = w.cause
		wraps = append(wraps, w.Base)
		w, _ = w.cause.(*Wrapped)
	}
	// For consistency, wraps should be in same order as stacks:
	// first element is from the innermost wrap
	// Hence we need to reverse the append operation above.
	for i := len(wraps)/2 - 1; i >= 0; i-- {
		j := len(wraps) - 1 - i
		wraps[i], wraps[j] = wraps[j], wraps[i]
	}
	return
}

// Cause checks if the passed in error is a Wrapped. If so, it will
// extract & return information about all the wrapped errors inside.
// It will return the CallStack of cause, if it supports the errors.Stack()
// interface, or the innermost wrap (which should be the closest wrap to cause.)
// If error is not a wrapped, cause is same as input err.
func Cause(err error) (stack CallStack, wraps []*Base, cause error) {
	cause = err
	if w, ok := err.(*Wrapped); ok {
		wraps, cause = w.Unwrap()
	}

	type stacker interface {
		Stack() CallStack
	}

	if s, ok := cause.(stacker); ok {
		stack = s.Stack()
	} else {
		if len(wraps) > 0 {
			stack = wraps[0].Stack()
		}
	}

	return
}
