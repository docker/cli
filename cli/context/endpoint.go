package context

// EndpointMetaBase contains fields we expect to be common for most context endpoints
type EndpointMetaBase struct {
	Host          string `json:"host,omitempty"`
	SkipTLSVerify bool   `json:"skip_tls_verify"`
}
