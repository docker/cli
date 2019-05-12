package crosspath

import (
	"fmt"
	"runtime"
)

// TargetOS qualifies a path with a target OS
type TargetOS string

const (
	// Unix represents Unix-style file system
	Unix = TargetOS("unix")
	// Windows represents Windows-style file system
	Windows = TargetOS("windows")
)

// Kind qualifies the nature of a path (absolute, relative, home-rooted...)
type Kind string

const (
	// Absolute is an absolute path
	Absolute = Kind("absolute")
	// Relative is a relative path
	Relative = Kind("relative")
	// HomeRooted is a path relative to user's home path
	HomeRooted = Kind("home-rooted")
	// AbsoluteFromCurrentDrive is a windows only kind of style "\some\path"
	AbsoluteFromCurrentDrive = Kind("absolute-current-drive")
	// RelativeFromDriveCurrentDir is a wondows only of style "c:some\path"
	RelativeFromDriveCurrentDir = Kind("relative-drive-specified")
	// WindowsDevice represents a path to a Windows device or virtual device such as pipe
	WindowsDevice = Kind("windows-device")
	// UNC is a UNC share file path
	UNC = Kind("unc")
)

// Path represents a file Path
type Path interface {
	fmt.Stringer
	TargetOS() TargetOS
	Kind() Kind
	Separator() rune
	segments() []string
	Convert(os TargetOS) (Path, error)
	Normalize() Path
	Join(paths ...Path) (Path, error)
	hasWindowsSpecificNamespacePrefix() bool
	Raw() string
}

// RuntimeOS returns information about the running OS (in term of file paths semantic)
func RuntimeOS() TargetOS {
	if runtime.GOOS == "windows" {
		return Windows
	}
	return Unix
}
