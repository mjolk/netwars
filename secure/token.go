package secure

import (
	"appengine"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"github.com/dgrijalva/jwt-go"
	"mj0lk.be/netwars/app"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	certificates map[string]*x509.Certificate
	certMutex    sync.RWMutex
)

const (
	JWTSECRET = "blalxdjbvvszkcyh56^b-=9if%=h1e%$ld=@4(js50t!$ld*a@5vcu(=2d0jxvxkbgtnhiuk"
	TTL       = 168
)

func getCertificate(key string) (*x509.Certificate, error) {
	certMutex.RLock()
	defer certMutex.RUnlock()
	cert, ok := certificates[key]
	if !ok {
		return nil, errors.New("No certificate")
	}
	return cert, nil
}

func loadCertificates(c appengine.Context) error {
	appCerts, err := appengine.PublicCertificates(c)
	if err != nil {
		return err
	}
	certMutex.Lock()
	defer certMutex.Unlock()
	certificates = make(map[string]*x509.Certificate)
	for _, cert := range appCerts {
		block, _ := pem.Decode(cert.Data)
		if block == nil {
			return errors.New("no block")
		}
		xcert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return err
		}
		certificates[cert.KeyName] = xcert
	}
	return nil
}

func CreateTokenString(c appengine.Context, playerKey string) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	if !appengine.IsDevAppServer() {
		if len(certificates) == 0 {
			if err := loadCertificates(c); err != nil {
				return "", err
			}
		}
		token = jwt.New(&SigningMethodAppengine{c})
	}
	token.Claims["pkey"] = playerKey
	// Expire in a week
	token.Claims["exp"] = time.Now().Add(time.Hour * TTL).Unix()
	tokenString, err := token.SignedString([]byte(JWTSECRET))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func ValidateToken(tokenString string) (string, error) {
	var playerStr string
	var err error
	var token *jwt.Token
	if appengine.IsDevAppServer() {
		token, err = jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(JWTSECRET), nil
		})
	} else {
		keyIndex := strings.LastIndex(tokenString, ".")
		keyString := tokenString[keyIndex+1:]
		tokenString := tokenString[7:keyIndex]
		token, err = jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			cert, err := getCertificate(keyString)
			if err != nil {
				return nil, errors.New("no certificate")
			}
			return cert, nil
		})
	}
	if err == nil && token.Valid {
		playerStr = token.Claims["pkey"].(string)
	} else if err != nil {
		vErr := err.(*jwt.ValidationError)
		if jwt.ValidationErrorExpired == jwt.ValidationErrorExpired&vErr.Errors {
			//TODO expired : regenerate???
			//playerStr = token.Claims["pkey"].(string)

		}
		return "", vErr //errors.New("Invalid token")
	}
	return playerStr, nil
}

func Validator(inner app.EngineHandler) app.EngineHandler {
	return func(w http.ResponseWriter, r *http.Request, c app.Context) {
		if ah := r.Header.Get("Authorization"); ah != "" {
			// Should be a netwars token
			if len(ah) > 7 && strings.ToUpper(ah[:7]) == "N3TWARS" {
				playerStr, err := ValidateToken(ah)
				if err != nil {
					app.NoAccess(w)
				} else {
					c.User = playerStr
					inner(w, r, c)
				}
			} else {
				app.NoAccess(w)
			}
		} else {
			app.NoAccess(w)
		}
	}
}
