package tlsconf

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"time"
)

func GenTlsConfig(certFile, keyFile string) (*tls.Config, error) {
	var cert tls.Certificate
	if certFile == "" || keyFile == "" {
		certBytes, keyBytes, err := GenerateSelfSignedCertKey()
		if err != nil {
			log.Fatalf("failed to generate self-signed certificate: %v", err)
		}
		cert, err = tls.X509KeyPair(certBytes, keyBytes)
		if err != nil {
			return nil, fmt.Errorf("X509KeyPair err:%w", err)
		}
	} else {
		cert = loadTLSCertificate(certFile, keyFile)
	}
	tlsConf := &tls.Config{
		Certificates: []tls.Certificate{
			cert,
		},
	}
	return tlsConf, nil
}
func loadTLSCertificate(certFile, keyFile string) tls.Certificate {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Fatalf("failed to load TLS certificate: %v", err)
	}
	return cert
}

func GenerateSelfSignedCertKey() ([]byte, []byte, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Country:            []string{"CN"},
			Organization:       []string{"Easy"},
			OrganizationalUnit: []string{"Easy"},
			Province:           []string{"ShenZhen"},
			CommonName:         "xxxx",
			Locality:           []string{"xxxxx"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(10, 0, 0),

		KeyUsage: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature |
			x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		EmailAddresses:        []string{"xxxx@qq.com"},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template,
		&priv.PublicKey, priv)
	if err != nil {
		return nil, nil, err
	}

	// Generate cert
	certBuffer := bytes.Buffer{}
	if err := pem.Encode(&certBuffer, &pem.Block{Type: "CERTIFICATE",
		Bytes: derBytes}); err != nil {
		return nil, nil, err
	}

	// Generate key
	keyBuffer := bytes.Buffer{}
	if err := pem.Encode(&keyBuffer, &pem.Block{Type: "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv)}); err != nil {
		return nil, nil, err
	}

	return certBuffer.Bytes(), keyBuffer.Bytes(), nil
}
