package secure

import (
	"appengine"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
)

type SigningMethodAppengine struct {
	Context appengine.Context
}

func init() {
	jwt.RegisterSigningMethod("APPENGINE", func() jwt.SigningMethod {
		return new(SigningMethodAppengine)
	})
}

func (m *SigningMethodAppengine) Alg() string {
	return "APPENGINE"
}

func (m *SigningMethodAppengine) Verify(signingString, signature string, crt interface{}) error {
	// Key
	cert, ok := crt.(*x509.Certificate)
	if !ok {
		return errors.New("not a x509 certificate")
	}
	sig, err := jwt.DecodeSegment(signature)
	if err != nil {
		return err
	}
	if err := cert.CheckSignature(x509.SHA256WithRSA, []byte(signingString), sig); err != nil {
		return err
	}
	return nil
}

func (m *SigningMethodAppengine) Sign(signingString string, key interface{}) (sig string, err error) {
	keyName, sigbytes, err := appengine.SignBytes(m.Context, []byte(signingString))
	if err != nil {
		return "", err
	}
	sig = jwt.EncodeSegment(sigbytes)
	return fmt.Sprintf("%s.%s", sig, keyName), nil
}
