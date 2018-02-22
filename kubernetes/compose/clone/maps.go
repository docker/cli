package clone

// MapOfStringToSliceOfString deep copy a map[string][]string
func MapOfStringToSliceOfString(source map[string][]string) map[string][]string {
	if source == nil {
		return nil
	}
	res := make(map[string][]string, len(source))
	for k, v := range source {
		res[k] = SliceOfString(v)
	}
	return res
}

// MapOfStringToInt deep copy a map[string]int
func MapOfStringToInt(source map[string]int) map[string]int {
	if source == nil {
		return nil
	}
	res := make(map[string]int, len(source))
	for k, v := range source {
		res[k] = v
	}
	return res
}
