package secure

import (
	"appengine"
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/pem"
	"errors"
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

func (m *SigningMethodAppengine) Verify(signingString, signature string, crt interface{}) (err error) {
	// Key
	var sig []byte
	key, ok := crt.([]byte)
	if !ok {
		return errors.New("not a valid bye array")
	}
	if sig, err = jwt.DecodeSegment(signature); err == nil {
		var block *pem.Block
		if block, _ = pem.Decode(key); block != nil {
			var parsedKey interface{}
			if parsedKey, err = x509.ParsePKIXPublicKey(block.Bytes); err != nil {
				parsedKey, err = x509.ParseCertificate(block.Bytes)
			}
			if err == nil {
				if rsaKey, ok := parsedKey.(*rsa.PublicKey); ok {
					hasher := sha1.New()
					hasher.Write([]byte(signingString))

					err = rsa.VerifyPKCS1v15(rsaKey, crypto.SHA1, hasher.Sum(nil), sig)
				} else if cert, ok := parsedKey.(*x509.Certificate); ok {
					err = cert.CheckSignature(x509.SHA1WithRSA, []byte(signingString), sig)
				} else {
					err = errors.New("Key is not a valid RSA public key")
				}
			}
		} else {
			err = errors.New("Could not parse key data")
		}
	}
	return
}

func (m *SigningMethodAppengine) Sign(signingString string, key interface{}) (sig string, err error) {
	keyName, sigbytes, err := appengine.SignBytes(m.Context, []byte(signingString))
	if err != nil {
		return "", err
	}
	sig = jwt.EncodeSegment(sigbytes)
	l := len(certificates) - 2
	for k, cert := range certificates {
		if k == 0 && cert.KeyName == keyName {
			break
		} else if cert.KeyName == keyName {
			if k < l {
				certificates = append([]*appengine.Certificate{cert}, append(certificates[:k], certificates[k+1:]...)...)
			} else {
				certificates = append([]*appengine.Certificate{cert}, certificates[:k]...)
			}
		}
	}
	return
}
