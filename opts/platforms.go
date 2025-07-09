package opts

import (
	"strings"

	"github.com/containerd/platforms"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func NewPlatformSlice(val *[]ocispec.Platform) *PlatformSlice {
	return &PlatformSlice{values: val}
}

// PlatformSlice is a Value type for passing multiple platforms.
type PlatformSlice struct {
	values *[]ocispec.Platform
}

func (m *PlatformSlice) Set(value string) error {
	vals := strings.Split(value, ",")
	for _, val := range vals {
		p, err := platforms.Parse(val)
		if err != nil {
			return err
		}
		*m.values = append(*m.values, p)
	}
	return nil
}

// Type returns the type of this option
func (*PlatformSlice) Type() string {
	return "platforms"
}

// String returns a string representation of this option.
func (m *PlatformSlice) String() string {
	return strings.Join(m.GetSlice(), ", ")
}

// GetSlice returns the platforms as a string-slice.
func (m *PlatformSlice) GetSlice() []string {
	values := make([]string, 0, len(*m.values))
	for _, v := range *m.values {
		values = append(values, platforms.FormatAll(v))
	}
	return values
}

// Value returns the platforms
func (m *PlatformSlice) Value() []ocispec.Platform {
	return *m.values
}
