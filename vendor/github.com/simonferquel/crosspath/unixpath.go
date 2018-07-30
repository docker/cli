package crosspath

import (
	"errors"
	"strings"
)

var _ Path = &unixPath{}

// NewUnixPath parse a string and make it a unix-style path
func NewUnixPath(path string) (Path, error) {
	if len(path) == 0 {
		return nil, errors.New("path is empty")
	}
	return &unixPath{tokens: strings.Split(path, "/")}, nil
}

type unixPath struct {
	tokens []string
}

func (p *unixPath) Raw() string {
	return strings.Join(p.tokens, "/")
}

func (p *unixPath) String() string {
	return p.Normalize().Raw()
}

func (p *unixPath) TargetOS() TargetOS {
	return Unix
}

func (p *unixPath) Kind() Kind {
	switch p.tokens[0] {
	case "":
		return Absolute
	case "~":
		return HomeRooted
	default:
		return Relative
	}
}

func (p *unixPath) Separator() rune {
	return '/'
}

func (p *unixPath) segments() []string {
	// clone
	result := make([]string, len(p.tokens))
	copy(result, p.tokens)
	return result
}

func (p *unixPath) Normalize() Path {
	var result []string
	if p.Kind() == Absolute {
		result = []string{""}
	}
	for _, s := range p.tokens {
		switch s {
		case "":
			continue
		case ".":
			continue
		case "..":
			if p.Kind() == Absolute && len(result) <= 1 {
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
			result = append(result, s)
		}
	}
	if len(result) == 0 {
		result = []string{"."}
	}
	return &unixPath{tokens: result}
}

func (p *unixPath) Join(paths ...Path) (Path, error) {
	if len(paths) == 0 {
		return p, nil
	}
	head := paths[0]
	tail := paths[1:]
	if head.Kind() != Relative && head.Kind() != HomeRooted {
		return nil, errors.New("can only join relative or home rooted paths")
	}
	var err error
	if head, err = head.Convert(Unix); err != nil {
		return nil, err
	}

	segs := head.segments()
	if head.Kind() == HomeRooted {
		segs = segs[1:]
	}
	current := &unixPath{tokens: append(p.tokens, segs...)}
	return current.Join(tail...)
}

func (p *unixPath) Convert(os TargetOS) (Path, error) {
	if os == Unix {
		return p, nil
	}
	switch p.Kind() {
	case Relative, HomeRooted:
		return NewWindowsPath(strings.Join(p.tokens, `\`), false)
	default:
		return nil, errors.New("only relative and home rooted paths can be converted")
	}
}

func (p *unixPath) hasWindowsSpecificNamespacePrefix() bool {
	return false
}
