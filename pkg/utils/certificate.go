package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"strings"
)

type KymaCertificates interface {
	GetCertificateKey() *rsa.PrivateKey
	GetCertificateRequest() []byte
	GetCertificates() ([]*x509.Certificate, error)
	AppendCertificates(base64encodedCertificate string)
}

func NewKymaCertificates(rawSubject string) KymaCertificates {
	return &kymaCerts{
		certKey:     nil,
		certRequest: nil,
		certificate: []*x509.Certificate{},
		rawSubject:  rawSubject,
	}
}

type kymaCerts struct {
	certKey     *rsa.PrivateKey
	certRequest []byte
	certificate []*x509.Certificate
	rawSubject  string
}

func (k *kymaCerts) GetCertificateKey() *rsa.PrivateKey {
	if k.certKey != nil {
		return k.certKey
	}
	certKey, err := createCertificateKey()
	if err != nil {
		panic("unable to create certificate key")
	}
	k.certKey = certKey
	return k.certKey
}

func (k *kymaCerts) GetCertificateRequest() []byte {
	if k.certRequest != nil {
		return k.certRequest
	}
	k.certRequest = k.createCertificateRequest()
	return k.certRequest
}

func (k *kymaCerts) GetCertificates() ([]*x509.Certificate, error) {
	if len(k.certificate) < 1 {
		return nil, errors.New("no certificates available")
	}
	return k.certificate, nil
}

func (k *kymaCerts) AppendCertificates(base64encodedCertificate string) {
	crtBytes, err := base64.StdEncoding.DecodeString(base64encodedCertificate)
	if err != nil {
		panic("Unable to decode base64 String")
	}

	var certificates []*x509.Certificate

	for block, rest := pem.Decode(crtBytes); block != nil && rest != nil; {
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			panic(err)
		}

		certificates = append(certificates, cert)

		block, rest = pem.Decode(rest)
	}

	k.certificate = append(k.certificate, certificates...)
}

func (k *kymaCerts) createCertificateRequest() []byte {
	subjectInfo := extractSubject(k.rawSubject)

	csrTemplate := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:         subjectInfo["CN"],
			Country:            []string{subjectInfo["C"]},
			Organization:       []string{subjectInfo["O"]},
			OrganizationalUnit: []string{subjectInfo["OU"]},
			Locality:           []string{subjectInfo["L"]},
			Province:           []string{subjectInfo["ST"]},
		},
	}
	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, csrTemplate, k.GetCertificateKey())
	if err != nil {
		panic("unable to create Certificate Request")
	}

	csr := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrBytes,
	})
	return csr
}

func createCertificateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 2048)
}

func extractSubject(subject string) map[string]string {
	result := map[string]string{}

	segments := strings.Split(subject, ",")

	for _, segment := range segments {
		parts := strings.Split(segment, "=")
		result[parts[0]] = parts[1]
	}

	return result
}
