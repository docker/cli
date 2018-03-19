// Package errors provides error and error wrapping facilities that allow
// for the easy reporting of call stacks and structured error annotations.
package errors

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// New returns a base error that captures the call stack.
func New(text string) error {
	return NewBase(1, text)
}

// Base is an error type that supports capturing the call stack at creation
// time, and storing separate text & data to allow structured logging.
// While it could be used directly, it may make more sense as an
// anonymous inside a package/application specific error struct.
type Base struct {
	Text      string
	Fields    Fields
	CallStack CallStack
}

// Fields holds the annotations for an error.
type Fields map[string]interface{}

// NewBase creates a new Base, capturing a call trace starting
// at "skip" calls above.
func NewBase(skip int, text string) *Base {
	return &Base{
		Text:      text,
		CallStack: CurrentCallStack(skip + 1),
	}
}

func (e *Base) Error() string {
	return textAndFields(e.Text, e.Fields)
}

// AddFields allows a Error message to be further annotated with
// a set of key,values, to add more context when inspecting
// Error messages.
func (e *Base) AddFields(fields Fields) {
	e.Fields = combineFields(e.Fields, fields)
}

// Location returns the location info (file, line, ...) for the place
// where this Error error was created.
func (e *Base) Location() *Location {
	return e.CallStack[0].Location()
}

func (e *Base) String() string {
	var td string
	loc := e.Location()

	if e.Text != "" || len(e.Fields) > 0 {
		td = fmt.Sprintf(": %s", textAndFields(e.Text, e.Fields))
	}
	return fmt.Sprintf("%s:%d%s", loc.File, loc.Line, td)
}

// MarshalJSON creates a JSON representation of a Error.
func (e *Base) MarshalJSON() ([]byte, error) {
	m := make(Fields)
	loc := e.Location()
	m["file"] = loc.File
	m["line"] = loc.Line
	m["func"] = loc.Func
	if e.Text != "" {
		m["text"] = e.Text
	}
	if len(e.Fields) > 0 {
		m["fields"] = e.Fields
	}
	return json.Marshal(m)
}

// Stack returns the call stack from where this Error was created.
func (e *Base) Stack() CallStack {
	return e.CallStack
}

func combineFields(f1 Fields, f2 Fields) Fields {
	data := make(Fields, len(f1)+len(f2))
	for k, v := range f1 {
		data[k] = v
	}
	for k, v := range f2 {
		data[k] = v
	}
	return data
}

func textAndFields(text string, fields Fields) string {
	buf := &bytes.Buffer{}

	if text != "" {
		buf.WriteString(text)
	}

	for k, v := range fields {
		buf.WriteByte(' ')
		buf.WriteString(k)
		buf.WriteByte('=')
		fmt.Fprintf(buf, "%+v", v)
	}

	return string(buf.Bytes())
}
