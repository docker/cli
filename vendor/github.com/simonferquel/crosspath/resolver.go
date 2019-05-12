package crosspath

// Preference represents a Comparer result
type Preference int

const (
	// PreferLeft indicates the comparer prefers the left value
	PreferLeft = Preference(-1)
	// PreferRight indicates the comparer prefers the right value
	PreferRight = Preference(1)
	// PreferNone indicates the comparer can't decide which one is better
	PreferNone = Preference(0)
)

// Comparer compares 2 paths and returns a value which indicates what Path it prefers
type Comparer func(lhs, rhs Path) Preference

// PreferOS returns a comparer that prefers an os target
func PreferOS(os TargetOS) Comparer {
	return func(lhs, rhs Path) Preference {
		if lhs.TargetOS() == rhs.TargetOS() {
			return PreferNone
		}
		if lhs.TargetOS() == os {
			return PreferLeft
		}
		if rhs.TargetOS() == os {
			return PreferRight
		}
		return PreferNone
	}
}

// PreferGreaterSegmentsLength returns a comparer that prefers path with more directory delimiters
func PreferGreaterSegmentsLength() Comparer {
	return func(lhs, rhs Path) Preference {
		l := len(lhs.segments())
		r := len(rhs.segments())
		if l == r {
			return PreferNone
		}
		if l > r {
			return PreferLeft
		}
		return PreferRight
	}
}

// PreferKinds returns a comparer that prefers path kinds in the specified order
func PreferKinds(kindsPreferenceOrder ...Kind) Comparer {
	return func(lhs, rhs Path) Preference {
		l := lhs.Kind()
		r := rhs.Kind()
		if l == r {
			return PreferNone
		}
		for _, k := range kindsPreferenceOrder {
			if l == k {
				return PreferLeft
			}
			if r == k {
				return PreferRight
			}
		}
		return PreferNone
	}
}

// PreferWithWindowsSpecificNamespacePrefix prefers paths with a win32 FileSystem or Device prefix
func PreferWithWindowsSpecificNamespacePrefix() Comparer {
	return func(lhs, rhs Path) Preference {
		if lhs.hasWindowsSpecificNamespacePrefix() {
			return PreferLeft
		}
		if rhs.hasWindowsSpecificNamespacePrefix() {
			return PreferRight
		}
		return PreferNone
	}
}

// PreferChain chain comparers to define the best Path candidate
func PreferChain(comparers ...Comparer) Comparer {
	return func(lhs, rhs Path) Preference {
		for _, c := range comparers {
			result := c(lhs, rhs)
			if result != PreferNone {
				return result
			}
		}
		return PreferNone
	}
}

// ParsePathWithPreference tries to parse the path both for windows and unix target OS, and returns the best match
// depending on the given comparer result
func ParsePathWithPreference(path string, comparer Comparer) (Path, error) {
	unixPath, err := NewUnixPath(path)
	if err != nil {
		// not valid for unix. so return a windows path
		return NewWindowsPath(path, true)
	}
	winPath, err := NewWindowsPath(path, true)
	if err != nil {
		return unixPath, nil
	}
	if comparer(unixPath, winPath) == PreferRight {
		return winPath, nil
	}
	return unixPath, nil
}

// ParsePathWithDefaults parse a path with default heuristics to define if a given path targets Windows or Linux
func ParsePathWithDefaults(path string) (Path, error) {
	p := PreferChain(
		PreferWithWindowsSpecificNamespacePrefix(),
		PreferKinds(Absolute, HomeRooted, AbsoluteFromCurrentDrive, RelativeFromDriveCurrentDir, WindowsDevice, UNC, Relative),
		PreferGreaterSegmentsLength(),
		PreferOS(Unix),
	)
	return ParsePathWithPreference(path, p)
}
