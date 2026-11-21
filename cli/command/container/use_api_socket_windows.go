package container

// addSocketGroup is a no-op on Windows.
func addSocketGroup(_ *[]string, _ string) {}
