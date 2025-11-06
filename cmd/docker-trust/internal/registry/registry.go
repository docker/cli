// Package registry contains client primitives to interact with a remote Docker registry.
package registry

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/docker/distribution/registry/client/transport"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/sirupsen/logrus"
)

func hasFile(files []os.DirEntry, name string) bool {
	for _, f := range files {
		if f.Name() == name {
			return true
		}
	}
	return false
}

// ReadCertsDirectory reads the directory for TLS certificates
// including roots and certificate pairs and updates the
// provided TLS configuration.
func ReadCertsDirectory(tlsConfig *tls.Config, directory string) error {
	return loadTLSConfig(context.TODO(), directory, tlsConfig)
}

// loadTLSConfig reads the directory for TLS certificates including roots and
// certificate pairs, and updates the provided TLS configuration.
func loadTLSConfig(ctx context.Context, directory string, tlsConfig *tls.Config) error {
	fs, err := os.ReadDir(directory)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return invalidParam(err)
	}

	for _, f := range fs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		switch filepath.Ext(f.Name()) {
		case ".crt":
			if tlsConfig.RootCAs == nil {
				systemPool, err := tlsconfig.SystemCertPool()
				if err != nil {
					return invalidParam(fmt.Errorf("unable to get system cert pool: %w", err))
				}
				tlsConfig.RootCAs = systemPool
			}
			fileName := filepath.Join(directory, f.Name())
			logrus.Debugf("crt: %s", fileName)
			data, err := os.ReadFile(fileName)
			if err != nil {
				return err
			}
			tlsConfig.RootCAs.AppendCertsFromPEM(data)
		case ".cert":
			certName := f.Name()
			keyName := certName[:len(certName)-5] + ".key"
			logrus.Debugf("cert: %s", filepath.Join(directory, certName))
			if !hasFile(fs, keyName) {
				return invalidParamf("missing key %s for client certificate %s. CA certificates must use the extension .crt", keyName, certName)
			}
			cert, err := tls.LoadX509KeyPair(filepath.Join(directory, certName), filepath.Join(directory, keyName))
			if err != nil {
				return err
			}
			tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
		case ".key":
			keyName := f.Name()
			certName := keyName[:len(keyName)-4] + ".cert"
			logrus.Debugf("key: %s", filepath.Join(directory, keyName))
			if !hasFile(fs, certName) {
				return invalidParamf("missing client certificate %s for key %s", certName, keyName)
			}
		}
	}

	return nil
}

// Headers returns request modifiers with a User-Agent and metaHeaders
func Headers(userAgent string, metaHeaders http.Header) []transport.RequestModifier {
	modifiers := []transport.RequestModifier{}
	if userAgent != "" {
		modifiers = append(modifiers, transport.NewHeaderRequestModifier(http.Header{
			"User-Agent": []string{userAgent},
		}))
	}
	if metaHeaders != nil {
		modifiers = append(modifiers, transport.NewHeaderRequestModifier(metaHeaders))
	}
	return modifiers
}
