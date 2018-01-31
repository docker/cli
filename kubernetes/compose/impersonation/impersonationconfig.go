package impersonation

import "github.com/docker/cli/kubernetes/compose/clone"

// Config contains the data required to impersonate a user.
type Config struct {
	// UserName is the username to impersonate on each request.
	UserName string
	// Groups are the groups to impersonate on each request.
	Groups []string
	// Extra is a free-form field which can be used to link some authentication information
	// to authorization information.  This field allows you to impersonate it.
	Extra map[string][]string
}

// Clone clones the impersonation config
func (ic *Config) Clone() *Config {
	if ic == nil {
		return nil
	}
	result := new(Config)
	result.UserName = ic.UserName
	result.Groups = clone.SliceOfString(ic.Groups)
	result.Extra = clone.MapOfStringToSliceOfString(ic.Extra)
	return result
}
