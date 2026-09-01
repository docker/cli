package main

import "slices"

// stringSliceReplaceAt replaces the sub-slice find with the sub-slice replace in s,
// returning a new slice and a boolean indicating whether the replacement happened.
// requireIndex is the index at which find must be found, or -1 to disregard it.
func stringSliceReplaceAt(s, find, replace []string, requireIndex int) ([]string, bool) {
	if len(find) == 0 {
		return s, false
	}

	if requireIndex >= 0 {
		if requireIndex+len(find) > len(s) || !slices.Equal(s[requireIndex:requireIndex+len(find)], find) {
			return s, false
		}
		return slices.Concat(s[:requireIndex], replace, s[requireIndex+len(find):]), true
	}

	for i := range len(s) - len(find) + 1 {
		if slices.Equal(s[i:i+len(find)], find) {
			return slices.Concat(s[:i], replace, s[i+len(find):]), true
		}
	}
	return s, false
}
