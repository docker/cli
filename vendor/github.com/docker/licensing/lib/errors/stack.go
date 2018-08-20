package errors

import (
	"encoding/json"
	"fmt"
	"runtime"
)

// Call holds a call frame
type Call runtime.Frame

// Location is the parsed file, line, and other info we can determine from
// a specific gostack.Call.
type Location struct {
	File string
	Line int
	Func string
}

// Location uses the gostack package to construct file, line, and other
// info about this call.
func (c Call) Location() *Location {
	return &Location{
		File: c.File,
		Line: c.Line,
		Func: c.Function,
	}
}

// MarshalJSON returns the JSON representation.
func (c Call) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	loc := c.Location()
	m["file"] = loc.File
	m["line"] = loc.Line
	m["func"] = loc.Func
	return json.Marshal(m)
}

func (c Call) String() string {
	return fmt.Sprintf("%v:%v", c.File, c.Line)
}

// CallStack is a convenience alias for a call stack.
type CallStack []Call

// CurrentCallStack returns the call stack, skipping the specified
// depth of calls.
func CurrentCallStack(skip int) CallStack {
	var pcs [128]uintptr
	n := runtime.Callers(skip+2, pcs[:])

	callersFrames := runtime.CallersFrames(pcs[:n])
	cs := make([]Call, 0, n)

	for {
		frame, more := callersFrames.Next()
		cs = append(cs, Call(frame))
		if !more {
			break
		}
	}

	return cs
}
