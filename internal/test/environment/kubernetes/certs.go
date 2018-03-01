package kubernetes

import (
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"math"
	"math/big"
	"net"
	"path/filepath"
	"time"
)

const (
	certValidityDuration = 365 * 24 * time.Hour
)

func generateCertificates(hostname string) (*certAuthority, map[string]cert, error) {
	ca, err := newCertAuthority("docker-kube", certValidityDuration)
	if err != nil {
		return nil, nil, err
	}

	hostnames := []string{"localhost", hostname, "kube-apiserver"}
	// FIXME(vdemeester) 10.96.0.1 is the apiserver ip
	ips := []net.IP{net.IPv4(127, 0, 0, 1), net.IPv4(10, 96, 0, 1)}
	apiServerCert, apiServerKey, err := ca.newSignedCert("kube-apiserver",
		nil,
		[]x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		hostnames,
		ips,
		certValidityDuration)
	if err != nil {
		return nil, nil, err
	}

	kubeletClientCert, kubeletClientKey, err := ca.newSignedCert("kube-apiserver-kubelet-client",
		[]string{"system:masters"},
		[]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		nil,
		nil,
		certValidityDuration)
	if err != nil {
		return nil, nil, err
	}

	frontCA, err := newCertAuthority("docker-kube-front", certValidityDuration)
	if err != nil {
		return nil, nil, err
	}

	frontClientCert, frontClientKey, err := frontCA.newSignedCert("kube-front-client",
		nil,
		[]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		nil,
		nil,
		certValidityDuration)
	if err != nil {
		return nil, nil, err
	}

	return ca, map[string]cert{
		"ca": {
			cert: ca.caCert,
			pvk:  ca.caPrivateKey(),
		},
		"apiserver": {
			cert: apiServerCert,
			pvk:  apiServerKey,
		},
		"kubelet-client": {
			cert: kubeletClientCert,
			pvk:  kubeletClientKey,
		},
		"front-ca": {
			cert: frontCA.caCert,
			pvk:  frontCA.caPrivateKey(),
		},
		"front-client": {
			cert: frontClientCert,
			pvk:  frontClientKey,
		},
	}, nil
}

type cert struct {
	cert *x509.Certificate
	pvk  *rsa.PrivateKey
}

type certAuthority struct {
	caCert *x509.Certificate
	caPvk  *rsa.PrivateKey
}

func newPVK() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(cryptorand.Reader, 2048)
}

func newCertAuthority(commonName string, validity time.Duration) (*certAuthority, error) {
	privateKey, err := newPVK()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	expires := now.Add(validity)
	template := x509.Certificate{
		SerialNumber: new(big.Int).SetInt64(0),
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore:             now.UTC(),
		NotAfter:              expires.UTC(),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA: true,
	}
	der, err := x509.CreateCertificate(cryptorand.Reader, &template, &template, privateKey.Public(), privateKey)
	if err != nil {
		return nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	return &certAuthority{caCert: cert, caPvk: privateKey}, nil
}

func (ca *certAuthority) caPrivateKey() *rsa.PrivateKey {
	return ca.caPvk
}

func (ca *certAuthority) newSignedCert(commonName string, organization []string,
	usages []x509.ExtKeyUsage,
	dnsNames []string,
	ips []net.IP,
	validity time.Duration,
) (*x509.Certificate, *rsa.PrivateKey, error) {

	expires := time.Now().Add(validity)
	serial, err := cryptorand.Int(cryptorand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, nil, err
	}
	privateKey, err := rsa.GenerateKey(cryptorand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	if len(commonName) == 0 {
		return nil, nil, errors.New("must specify a CommonName")
	}
	if len(usages) == 0 {
		return nil, nil, errors.New("must specify at least one ExtKeyUsage")
	}

	template := x509.Certificate{
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: organization,
		},
		DNSNames:     dnsNames,
		IPAddresses:  ips,
		SerialNumber: serial,
		NotBefore:    ca.caCert.NotBefore,
		NotAfter:     expires.UTC(),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  usages,
	}
	der, err := x509.CreateCertificate(cryptorand.Reader, &template, ca.caCert, privateKey.Public(), ca.caPvk)
	if err != nil {
		return nil, nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, nil, err
	}
	return cert, privateKey, nil
}

func writeCertAndKey(dir, baseName string, cert *x509.Certificate, key *rsa.PrivateKey) error {
	certPEM := encodeCertPEM(cert)
	keyPEM := encodePrivateKeyPEM(key)
	if err := ioutil.WriteFile(filepath.Join(dir, baseName+".crt"), certPEM, 0600); err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(dir, baseName+".key"), keyPEM, 0600)
}

func writePublicAndPrivateKey(dir, baseName string, key *rsa.PrivateKey) error {
	pubPEM, err := encodePublicKeyPEM(&key.PublicKey)
	if err != nil {
		return err
	}
	keyPEM := encodePrivateKeyPEM(key)
	if err := ioutil.WriteFile(filepath.Join(dir, baseName+".pub"), pubPEM, 0600); err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(dir, baseName+".key"), keyPEM, 0600)
}

func encodePublicKeyPEM(key *rsa.PublicKey) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return []byte{}, err
	}
	block := pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: der,
	}
	return pem.EncodeToMemory(&block), nil
}

func encodePrivateKeyPEM(key *rsa.PrivateKey) []byte {
	block := pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	return pem.EncodeToMemory(&block)
}

func encodeCertPEM(cert *x509.Certificate) []byte {
	block := pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(&block)
}
