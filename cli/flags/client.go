package flags

// ClientOptions are the options used to configure the client cli
type ClientOptions struct {
	Common    *CommonOptions
	ConfigDir string
}

// NewClientOptions returns a new ClientOptions
//构造函数
func NewClientOptions() *ClientOptions {
	return &ClientOptions{Common: NewCommonOptions()}
}
