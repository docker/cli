package opts

import (
	"fmt"
	"testing"

	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestParseHost(t *testing.T) {
	invalid := []string{
		"something with spaces",
		"://",
		"unknown://",
		"tcp://:port",
		"tcp://invalid:port",
	}

	valid := map[string]string{
		"":                         defaultHost,
		" ":                        defaultHost,
		"  ":                       defaultHost,
		"fd://":                    "fd://",
		"fd://something":           "fd://something",
		"tcp://host:":              "tcp://host:" + defaultHTTPPort,
		"tcp://":                   defaultTCPHost,
		"tcp://:2375":              fmt.Sprintf("tcp://%s:%s", defaultHTTPHost, defaultHTTPPort),
		"tcp://:2376":              fmt.Sprintf("tcp://%s:%s", defaultHTTPHost, defaultTLSHTTPPort),
		"tcp://0.0.0.0:8080":       "tcp://0.0.0.0:8080",
		"tcp://192.168.0.0:12000":  "tcp://192.168.0.0:12000",
		"tcp://192.168:8080":       "tcp://192.168:8080",
		"tcp://0.0.0.0:1234567890": "tcp://0.0.0.0:1234567890", // yeah it's valid :P
		" tcp://:7777/path ":       fmt.Sprintf("tcp://%s:7777/path", defaultHTTPHost),
		"tcp://docker.com:2375":    "tcp://docker.com:2375",
		"unix://":                  "unix://" + defaultUnixSocket,
		"unix://path/to/socket":    "unix://path/to/socket",
		"npipe://":                 "npipe://" + defaultNamedPipe,
		"npipe:////./pipe/foo":     "npipe:////./pipe/foo",
	}

	for _, value := range invalid {
		if _, err := ParseHost(false, value); err == nil {
			t.Errorf("Expected an error for %v, got [nil]", value)
		}
	}

	for value, expected := range valid {
		if actual, err := ParseHost(false, value); err != nil || actual != expected {
			t.Errorf("Expected for %v [%v], got [%v, %v]", value, expected, actual, err)
		}
	}
}

func TestParseDockerDaemonHost(t *testing.T) {
	invalids := map[string]string{
		"tcp:a.b.c.d":                   "",
		"tcp:a.b.c.d/path":              "",
		"udp://127.0.0.1":               "invalid bind address format: udp://127.0.0.1",
		"udp://127.0.0.1:2375":          "invalid bind address format: udp://127.0.0.1:2375",
		"tcp://unix:///run/docker.sock": "invalid proto, expected tcp: unix:///run/docker.sock",
		" tcp://:7777/path ":            "invalid bind address format:  tcp://:7777/path ", //nolint:gocritic // ignore mapKey: suspucious whitespace
		"":                              "invalid bind address format: ",
	}
	valids := map[string]string{
		"0.0.0.1:":                    "tcp://0.0.0.1:2375",
		"0.0.0.1:5555":                "tcp://0.0.0.1:5555",
		"0.0.0.1:5555/path":           "tcp://0.0.0.1:5555/path",
		"[::1]:":                      "tcp://[::1]:2375",
		"[::1]:5555/path":             "tcp://[::1]:5555/path",
		"[0:0:0:0:0:0:0:1]:":          "tcp://[0:0:0:0:0:0:0:1]:2375",
		"[0:0:0:0:0:0:0:1]:5555/path": "tcp://[0:0:0:0:0:0:0:1]:5555/path",
		":6666":                       fmt.Sprintf("tcp://%s:6666", defaultHTTPHost),
		":6666/path":                  fmt.Sprintf("tcp://%s:6666/path", defaultHTTPHost),
		"tcp://":                      defaultTCPHost,
		"tcp://:7777":                 fmt.Sprintf("tcp://%s:7777", defaultHTTPHost),
		"tcp://:7777/path":            fmt.Sprintf("tcp://%s:7777/path", defaultHTTPHost),
		"unix:///run/docker.sock":     "unix:///run/docker.sock",
		"unix://":                     "unix://" + defaultUnixSocket,
		"fd://":                       "fd://",
		"fd://something":              "fd://something",
		"localhost:":                  "tcp://localhost:2375",
		"localhost:5555":              "tcp://localhost:5555",
		"localhost:5555/path":         "tcp://localhost:5555/path",
	}
	for invalidAddr, expectedError := range invalids {
		if addr, err := parseDockerDaemonHost(invalidAddr); err == nil || expectedError != "" && err.Error() != expectedError {
			t.Errorf("tcp %v address expected error %q return, got %q and addr %v", invalidAddr, expectedError, err, addr)
		}
	}
	for validAddr, expectedAddr := range valids {
		if addr, err := parseDockerDaemonHost(validAddr); err != nil || addr != expectedAddr {
			t.Errorf("%v -> expected %v, got (%v) addr (%v)", validAddr, expectedAddr, err, addr)
		}
	}
}

func TestParseTCP(t *testing.T) {
	defaultHTTPHost := "tcp://127.0.0.1:2376"
	invalids := map[string]string{
		"tcp:a.b.c.d":          "",
		"tcp:a.b.c.d/path":     "",
		"udp://127.0.0.1":      "invalid proto, expected tcp: udp://127.0.0.1",
		"udp://127.0.0.1:2375": "invalid proto, expected tcp: udp://127.0.0.1:2375",
	}
	valids := map[string]string{
		"":                            defaultHTTPHost,
		"tcp://":                      defaultHTTPHost,
		"0.0.0.1:":                    "tcp://0.0.0.1:2376",
		"0.0.0.1:5555":                "tcp://0.0.0.1:5555",
		"0.0.0.1:5555/path":           "tcp://0.0.0.1:5555/path",
		":6666":                       "tcp://127.0.0.1:6666",
		":6666/path":                  "tcp://127.0.0.1:6666/path",
		"tcp://:7777":                 "tcp://127.0.0.1:7777",
		"tcp://:7777/path":            "tcp://127.0.0.1:7777/path",
		"[::1]:":                      "tcp://[::1]:2376",
		"[::1]:5555":                  "tcp://[::1]:5555",
		"[::1]:5555/path":             "tcp://[::1]:5555/path",
		"[0:0:0:0:0:0:0:1]:":          "tcp://[0:0:0:0:0:0:0:1]:2376",
		"[0:0:0:0:0:0:0:1]:5555":      "tcp://[0:0:0:0:0:0:0:1]:5555",
		"[0:0:0:0:0:0:0:1]:5555/path": "tcp://[0:0:0:0:0:0:0:1]:5555/path",
		"localhost:":                  "tcp://localhost:2376",
		"localhost:5555":              "tcp://localhost:5555",
		"localhost:5555/path":         "tcp://localhost:5555/path",
	}
	for invalidAddr, expectedError := range invalids {
		if addr, err := ParseTCPAddr(invalidAddr, defaultHTTPHost); err == nil || expectedError != "" && err.Error() != expectedError {
			t.Errorf("tcp %v address expected error %v return, got %s and addr %v", invalidAddr, expectedError, err, addr)
		}
	}
	for validAddr, expectedAddr := range valids {
		if addr, err := ParseTCPAddr(validAddr, defaultHTTPHost); err != nil || addr != expectedAddr {
			t.Errorf("%v -> expected %v, got %v and addr %v", validAddr, expectedAddr, err, addr)
		}
	}
}

func TestParseInvalidUnixAddrInvalid(t *testing.T) {
	if _, err := parseSimpleProtoAddr("unix", "tcp://127.0.0.1", "unix:///var/run/docker.sock"); err == nil || err.Error() != "invalid proto, expected unix: tcp://127.0.0.1" {
		t.Fatalf("Expected an error, got %v", err)
	}
	if _, err := parseSimpleProtoAddr("unix", "unix://tcp://127.0.0.1", "/var/run/docker.sock"); err == nil || err.Error() != "invalid proto, expected unix: tcp://127.0.0.1" {
		t.Fatalf("Expected an error, got %v", err)
	}
	if v, err := parseSimpleProtoAddr("unix", "", "/var/run/docker.sock"); err != nil || v != "unix:///var/run/docker.sock" {
		t.Fatalf("Expected an %v, got %v", v, "unix:///var/run/docker.sock")
	}
}

func TestValidateExtraHosts(t *testing.T) {
	tests := []struct {
		doc         string
		input       string
		expectedOut string // Expect output==input if not set.
		expectedErr string // Expect success if not set.
	}{
		{
			doc:   "IPv4, colon sep",
			input: `myhost:192.168.0.1`,
		},
		{
			doc:         "IPv4, eq sep",
			input:       `myhost=192.168.0.1`,
			expectedOut: `myhost:192.168.0.1`,
		},
		{
			doc:         "Weird but permitted, IPv4 with brackets",
			input:       `myhost=[192.168.0.1]`,
			expectedOut: `myhost:192.168.0.1`,
		},
		{
			doc:   "Host and domain",
			input: `host.and.domain.invalid:10.0.2.1`,
		},
		{
			doc:   "IPv6, colon sep",
			input: `anipv6host:2003:ab34:e::1`,
		},
		{
			doc:         "IPv6, colon sep, brackets",
			input:       `anipv6host:[2003:ab34:e::1]`,
			expectedOut: `anipv6host:2003:ab34:e::1`,
		},
		{
			doc:         "IPv6, eq sep, brackets",
			input:       `anipv6host=[2003:ab34:e::1]`,
			expectedOut: `anipv6host:2003:ab34:e::1`,
		},
		{
			doc:   "IPv6 localhost, colon sep",
			input: `ipv6local:::1`,
		},
		{
			doc:         "IPv6 localhost, eq sep",
			input:       `ipv6local=::1`,
			expectedOut: `ipv6local:::1`,
		},
		{
			doc:         "IPv6 localhost, eq sep, brackets",
			input:       `ipv6local=[::1]`,
			expectedOut: `ipv6local:::1`,
		},
		{
			doc:   "IPv6 localhost, non-canonical, colon sep",
			input: `ipv6local:0:0:0:0:0:0:0:1`,
		},
		{
			doc:         "IPv6 localhost, non-canonical, eq sep",
			input:       `ipv6local=0:0:0:0:0:0:0:1`,
			expectedOut: `ipv6local:0:0:0:0:0:0:0:1`,
		},
		{
			doc:         "IPv6 localhost, non-canonical, eq sep, brackets",
			input:       `ipv6local=[0:0:0:0:0:0:0:1]`,
			expectedOut: `ipv6local:0:0:0:0:0:0:0:1`,
		},
		{
			doc:   "host-gateway special case, colon sep",
			input: `host.docker.internal:host-gateway`,
		},
		{
			doc:         "host-gateway special case, eq sep",
			input:       `host.docker.internal=host-gateway`,
			expectedOut: `host.docker.internal:host-gateway`,
		},
		{
			doc:         "Bad address, colon sep",
			input:       `myhost:192.notanipaddress.1`,
			expectedErr: `invalid IP address in add-host: "192.notanipaddress.1"`,
		},
		{
			doc:         "Bad address, eq sep",
			input:       `myhost=192.notanipaddress.1`,
			expectedErr: `invalid IP address in add-host: "192.notanipaddress.1"`,
		},
		{
			doc:         "No sep",
			input:       `thathost-nosemicolon10.0.0.1`,
			expectedErr: `bad format for add-host: "thathost-nosemicolon10.0.0.1"`,
		},
		{
			doc:         "Bad IPv6",
			input:       `anipv6host:::::1`,
			expectedErr: `invalid IP address in add-host: "::::1"`,
		},
		{
			doc:         "Bad IPv6, trailing colons",
			input:       `ipv6local:::0::`,
			expectedErr: `invalid IP address in add-host: "::0::"`,
		},
		{
			doc:         "Bad IPv6, missing close bracket",
			input:       `ipv6addr=[::1`,
			expectedErr: `invalid IP address in add-host: "[::1"`,
		},
		{
			doc:         "Bad IPv6, missing open bracket",
			input:       `ipv6addr=::1]`,
			expectedErr: `invalid IP address in add-host: "::1]"`,
		},
		{
			doc:         "Missing address, colon sep",
			input:       `myhost.invalid:`,
			expectedErr: `invalid IP address in add-host: ""`,
		},
		{
			doc:         "Missing address, eq sep",
			input:       `myhost.invalid=`,
			expectedErr: `invalid IP address in add-host: ""`,
		},
		{
			doc:         "IPv6 localhost, bad name",
			input:       `:=::1`,
			expectedErr: `bad format for add-host: ":=::1"`,
		},
		{
			doc:         "No input",
			input:       ``,
			expectedErr: `bad format for add-host: ""`,
		},
	}

	for _, tc := range tests {
		if tc.expectedOut == "" {
			tc.expectedOut = tc.input
		}
		t.Run(tc.input, func(t *testing.T) {
			actualOut, actualErr := ValidateExtraHost(tc.input)
			if tc.expectedErr == "" {
				assert.Check(t, is.Equal(tc.expectedOut, actualOut))
				assert.NilError(t, actualErr)
			} else {
				assert.Check(t, actualOut == "")
				assert.Check(t, is.Error(actualErr, tc.expectedErr))
			}
		})
	}
}
