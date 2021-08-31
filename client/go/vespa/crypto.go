package vespa

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"time"
)

const (
	defaultCommonName = "cloud.vespa.example"
	certificateExpiry = 3650 * 24 * time.Hour // Approximately 10 years
	tempFilePattern   = "vespa"
)

// PemKeyPair represents a PEM-encoded private key and X509 certificate.
type PemKeyPair struct {
	Certificate []byte
	PrivateKey  []byte
}

// WriteCertificateFile writes the certificate contained in this key pair to certificateFile.
func (kp *PemKeyPair) WriteCertificateFile(certificateFile string, overwrite bool) error {
	return atomicWriteFile(certificateFile, kp.Certificate, overwrite)
}

// WritePrivateKeyFile writes the private key contained in this key pair to privateKeyFile.
func (kp *PemKeyPair) WritePrivateKeyFile(privateKeyFile string, overwrite bool) error {
	return atomicWriteFile(privateKeyFile, kp.PrivateKey, overwrite)
}

func atomicWriteFile(filename string, data []byte, overwrite bool) error {
	tmpFile, err := os.CreateTemp("", tempFilePattern)
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	if err := os.WriteFile(tmpFile.Name(), data, 0600); err != nil {
		return err
	}
	_, err = os.Stat(filename)
	if errors.Is(err, os.ErrNotExist) || overwrite {
		return os.Rename(tmpFile.Name(), filename)
	}
	return fmt.Errorf("cannot overwrite existing file: %s", filename)
}

// CreateKeyPair creates a key pair containing a private key and self-signed X509 certificate.
func CreateKeyPair() (PemKeyPair, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return PemKeyPair{}, fmt.Errorf("failed to generate private key: %w", err)
	}
	serialNumber, err := randomSerialNumber()
	if err != nil {
		return PemKeyPair{}, fmt.Errorf("failed to create serial number: %w", err)
	}
	notBefore := time.Now()
	notAfter := notBefore.Add(certificateExpiry)
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      pkix.Name{CommonName: defaultCommonName},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
	}
	certificateDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return PemKeyPair{}, err
	}
	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return PemKeyPair{}, err
	}
	pemPrivateKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyDER})
	pemCertificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificateDER})
	return PemKeyPair{Certificate: pemCertificate, PrivateKey: pemPrivateKey}, nil
}

type RequestSigner struct {
	now           func() time.Time
	rnd           io.Reader
	KeyID         string
	PemPrivateKey []byte
}

// NewRequestSigner creates a new signer using the EC pemPrivateKey. keyID names the key used to sign requests.
func NewRequestSigner(keyID string, pemPrivateKey []byte) *RequestSigner {
	return &RequestSigner{
		now:           time.Now,
		rnd:           rand.Reader,
		KeyID:         keyID,
		PemPrivateKey: pemPrivateKey,
	}
}

// SignRequest signs the given HTTP request using the private key in rs
func (rs *RequestSigner) SignRequest(request *http.Request) error {
	timestamp := rs.now().UTC().Format(time.RFC3339)
	contentHash, body, err := contentHash(request.Body)
	if err != nil {
		return err
	}
	privateKey, err := ecPrivateKeyFrom(rs.PemPrivateKey)
	if err != nil {
		return err
	}
	pemPublicKey, err := pemPublicKeyFrom(privateKey)
	if err != nil {
		return err
	}
	base64PemPublicKey := base64.StdEncoding.EncodeToString(pemPublicKey)
	signature, err := rs.hashAndSign(privateKey, request, timestamp, contentHash)
	if err != nil {
		return err
	}
	base64Signature := base64.StdEncoding.EncodeToString(signature)
	request.Body = io.NopCloser(body)
	request.Header.Set("X-Timestamp", timestamp)
	request.Header.Set("X-Content-Hash", contentHash)
	request.Header.Set("X-Key-Id", rs.KeyID)
	request.Header.Set("X-Key", base64PemPublicKey)
	request.Header.Set("X-Authorization", base64Signature)
	return nil
}

func (rs *RequestSigner) hashAndSign(privateKey *ecdsa.PrivateKey, request *http.Request, timestamp, contentHash string) ([]byte, error) {
	msg := []byte(request.Method + "\n" + request.URL.String() + "\n" + timestamp + "\n" + contentHash)
	hasher := sha256.New()
	hasher.Write(msg)
	hash := hasher.Sum(nil)
	return ecdsa.SignASN1(rs.rnd, privateKey, hash)
}

func ecPrivateKeyFrom(pemPrivateKey []byte) (*ecdsa.PrivateKey, error) {
	privateKeyBlock, _ := pem.Decode(pemPrivateKey)
	if privateKeyBlock == nil {
		return nil, fmt.Errorf("invalid pem private key")
	}
	return x509.ParseECPrivateKey(privateKeyBlock.Bytes)
}

func pemPublicKeyFrom(privateKey *ecdsa.PrivateKey) ([]byte, error) {
	publicKeyDER, err := x509.MarshalPKIXPublicKey(privateKey.Public())
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyDER}), nil
}

func contentHash(r io.Reader) (string, io.Reader, error) {
	var copy bytes.Buffer
	teeReader := io.TeeReader(r, &copy) // Copy reader contents while we hash it
	hasher := sha256.New()
	if _, err := io.Copy(hasher, teeReader); err != nil {
		return "", nil, err
	}
	hashSum := hasher.Sum(nil)
	return base64.StdEncoding.EncodeToString(hashSum), &copy, nil
}

func randomSerialNumber() (*big.Int, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, serialNumberLimit)
}
