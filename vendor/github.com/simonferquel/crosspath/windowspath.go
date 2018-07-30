package crosspath

import (
	"errors"
	"fmt"
	"strings"
)

const (
	windowsForbiddenRunes           = "<>:\"/\\|?*\r\n\t"
	windowsFirstTokenForbiddenRunes = "<>\"/\\|?*\r\n\t"
	win32FileSystemNamespacePrefix  = `\\?\`
	win32DeviceNamespacePrefix      = `\\.\`
	uncPrefix                       = `\\`
)

func tokenizeWindowsPath(path string, isUnc bool) ([]string, error) {
	tokens := strings.Split(path, `\`)
	for ix, token := range tokens {
		forbidden := windowsForbiddenRunes
		if ix == 0 && !isUnc {
			forbidden = windowsFirstTokenForbiddenRunes
		}
		if strings.ContainsAny(token, forbidden) {
			return nil, fmt.Errorf("invalid charcter in token %q", token)
		}
		if strings.HasSuffix(token, " ") {
			return nil, fmt.Errorf("token %q should not end with a space", token)
		}
		if token != "." && token != ".." && strings.HasSuffix(token, ".") {
			return nil, fmt.Errorf("token %q should not end with a dot", token)
		}
	}
	if !isUnc && len(tokens) > 0 {
		// on non-unc path, ':' can only occur at the second character position
		idx := strings.IndexRune(tokens[0], ':')
		if idx != -1 && idx != 1 {
			return nil, fmt.Errorf("token %q should not contain ':' at this position", tokens[0])
		}
	}
	return tokens, nil
}

// NewWindowsPath parses a Windows file path
func NewWindowsPath(path string, convertSlashes bool) (Path, error) {
	if convertSlashes {
		path = strings.Replace(path, "/", `\`, -1)
	}
	namespacePrefix := ""
	prefix := ""
	isUnc := false
	if strings.HasPrefix(path, win32FileSystemNamespacePrefix) {
		path = strings.TrimPrefix(path, win32FileSystemNamespacePrefix)
		namespacePrefix = win32FileSystemNamespacePrefix
		if strings.HasPrefix(path, `UNC\`) {
			isUnc = true
			prefix = `UNC\`
			path = strings.TrimPrefix(path, `UNC\`)
		}
	} else if strings.HasPrefix(path, win32DeviceNamespacePrefix) {
		path = strings.TrimPrefix(path, win32DeviceNamespacePrefix)
		namespacePrefix = win32DeviceNamespacePrefix
	} else if strings.HasPrefix(path, uncPrefix) {
		isUnc = true
		path = strings.TrimPrefix(path, uncPrefix)
		prefix = uncPrefix
	}
	tokens, err := tokenizeWindowsPath(path, isUnc)
	if err != nil {
		return nil, err
	}
	// validation rules

	if len(tokens) == 0 {
		return nil, errors.New("unsupported empty path")
	}

	// if unc, first token cannot be empty, . or ..
	if isUnc &&
		(tokens[0] == "" || tokens[0] == "." || tokens[0] == "..") {
		return nil, errors.New("invalid unc path")
	}

	// device namespace prefix forbid unc paths
	if isUnc && namespacePrefix == win32DeviceNamespacePrefix {
		return nil, errors.New("cannot express UNC paths after a windows device namespace prefix")
	}
	return &windowsPath{
		namespacePrefix: namespacePrefix,
		prefix:          prefix,
		unc:             isUnc,
		tokens:          tokens,
	}, nil
}

type windowsPath struct {
	namespacePrefix string
	prefix          string
	unc             bool
	tokens          []string
}

func (p *windowsPath) Raw() string {
	return p.namespacePrefix + p.prefix + strings.Join(p.tokens, `\`)
}

func (p *windowsPath) String() string {
	return p.Normalize().Raw()
}

func (p *windowsPath) TargetOS() TargetOS {
	return Windows
}

func isTokenWindowsDriveRoot(token string) bool {
	return len(token) == 2 && token[1] == ':'
}
func (p *windowsPath) Kind() Kind {
	if p.namespacePrefix == win32DeviceNamespacePrefix {
		return WindowsDevice
	}
	if p.unc {
		return UNC
	}
	switch {
	case p.unc:
		return UNC
	case p.tokens[0] == `~`:
		return HomeRooted
	case isTokenWindowsDriveRoot(p.tokens[0]):
		return Absolute
	case (len(p.tokens[0]) > 2 && p.tokens[0][1] == ':'):
		return RelativeFromDriveCurrentDir
	case p.tokens[0] == "":
		return AbsoluteFromCurrentDrive
	default:
		return Relative
	}
}

func (p *windowsPath) Separator() rune {
	return '\\'
}

func (p *windowsPath) segments() []string {
	// clone
	result := make([]string, len(p.tokens))
	copy(result, p.tokens)
	return result
}

func (p *windowsPath) Normalize() Path {
	if p.namespacePrefix == win32FileSystemNamespacePrefix {
		// using this namespace bypasses all path resolution mechanism
		// and allows underlying file system drivers to interpret paths names like "." or ".." themselves
		// so, do nothing
		return p
	}
	kind := p.Kind()
	var result []string
	if kind == AbsoluteFromCurrentDrive {
		result = []string{""}
	}
	for _, token := range p.tokens {
		switch token {
		case "":
			continue
		case ".":
			continue
		case "..":
			if (kind == Absolute || kind == UNC || kind == AbsoluteFromCurrentDrive) && len(result) <= 1 {
				continue
			}
			if len(result) == 0 ||
				result[len(result)-1] == ".." ||
				(len(result) == 1 && result[0] == "~") {
				result = append(result, "..")
			} else {
				result = result[:len(result)-1]
			}
		default:
			result = append(result, token)
		}
	}
	if len(result) == 0 {
		result = []string{"."}
	}
	if len(result) == 1 && (kind == Absolute || kind == AbsoluteFromCurrentDrive) {
		result = append(result, "")
	}
	return &windowsPath{namespacePrefix: p.namespacePrefix, prefix: p.prefix, tokens: result, unc: p.unc}
}

func (p *windowsPath) Join(paths ...Path) (Path, error) {
	if len(paths) == 0 {
		return p, nil
	}
	head := paths[0]
	tail := paths[1:]
	if head.Kind() != Relative && head.Kind() != HomeRooted {
		return nil, errors.New("can only join relative paths")
	}
	var err error
	if head, err = head.Convert(Unix); err != nil {
		return nil, err
	}
	segs := head.segments()
	if head.Kind() == HomeRooted {
		segs = segs[1:]
	}
	current := &windowsPath{tokens: append(p.tokens, segs...), namespacePrefix: p.namespacePrefix, prefix: p.prefix, unc: p.unc}
	return current.Join(tail...)
}

func (p *windowsPath) Convert(os TargetOS) (Path, error) {
	if os == Windows {
		return p, nil
	}
	switch p.Kind() {
	case Relative, HomeRooted:
		return NewUnixPath(strings.Join(p.tokens, `/`))
	default:
		return nil, errors.New("only relative and home rooted paths can be converted")
	}
}

func (p *windowsPath) hasWindowsSpecificNamespacePrefix() bool {
	return p.namespacePrefix != ""
}
