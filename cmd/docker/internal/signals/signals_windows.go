package signals

import "os"

// TerminationSignals represents the list of signals we
// want to special-case handle, on this platform.
var TerminationSignals = []os.Signal{os.Interrupt}
