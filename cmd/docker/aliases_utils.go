package main

func stringSliceIndex(s, subs []string) int {
	j := 0
	if len(subs) > 0 {
		for i, x := range s {
			if j < len(subs) && subs[j] == x {
				j++
			} else {
				j = 0
			}
			if len(subs) == j {
				return i + 1 - j
			}
		}
	}
	return -1
}

// stringSliceReplaceAt replaces the sub-slice find, with the sub-slice replace, in the string
// slice s, returning a new slice and a boolean indicating if the replacement happened.
// requireIdx is the index at which old needs to be found at (or -1 to disregard that).
func stringSliceReplaceAt(s, find, replace []string, requireIndex int) ([]string, bool) {
	idx := stringSliceIndex(s, find)
	if (requireIndex != -1 && requireIndex != idx) || idx == -1 {
		return s, false
	}
	out := append([]string{}, s[:idx]...)
	out = append(out, replace...)
	out = append(out, s[idx+len(find):]...)
	return out, true
}
