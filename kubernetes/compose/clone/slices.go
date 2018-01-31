package clone

// SliceOfString deep copy a slice of strings
func SliceOfString(source []string) []string {
	if source == nil {
		return nil
	}
	res := make([]string, len(source))
	copy(res, source)
	return res
}
