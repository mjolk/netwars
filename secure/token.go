package secure

import (
	"appengine"
	"errors"
	"github.com/dgrijalva/jwt-go"
	"mj0lk.be/netwars/app"
	"net/http"
	"strings"
	"time"
)

var (
	certificates []*appengine.Certificate
)

const (
	JWTSECRET = "blalxdjbvvszkcyh56^b-=9if%=h1e%$ld=@4(js50t!$ld*a@5vcu(=2d0jxvxkbgtnhiuk"
	TTL       = 168
)

func loadCertificates(c appengine.Context) error {
	certs, err := appengine.PublicCertificates(c)
	if err != nil {
		return err
	}
	ln := len(certs)
	certificates = make([]*appengine.Certificate, ln, ln)
	for k, cert := range certs {
		certificates[k] = &cert
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
	var vErr *jwt.ValidationError
	var token *jwt.Token
	if !appengine.IsDevAppServer() {
		maxTries := len(certificates)
		for i := 0; i < maxTries; {
			var err error
			token, err = jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				cert := certificates[i]
				return cert.Data, nil
			})
			if err != nil {
				vErr = err.(*jwt.ValidationError)
				if jwt.ValidationErrorSignatureInvalid == vErr.Errors&jwt.ValidationErrorSignatureInvalid {
					i++
					continue
				}
			}
		}
	} else {
		var err error
		token, err = jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(JWTSECRET), nil
		})
		if err != nil {
			vErr = err.(*jwt.ValidationError)

		}
	}

	if vErr == nil && token.Valid {
		playerStr = token.Claims["pkey"].(string)
	} else if vErr != nil {
		if jwt.ValidationErrorExpired == jwt.ValidationErrorExpired&vErr.Errors {
			//TODO expired : regenerate???
			//playerStr = token.Claims["pkey"].(string)

		}
		return "", errors.New("Invalid token")
	}
	return playerStr, nil
}

func Validator(inner app.EngineHandler) app.EngineHandler {
	return func(w http.ResponseWriter, r *http.Request, c app.Context) {
		if ah := r.Header.Get("Authorization"); ah != "" {
			// Should be a netwars token
			if len(ah) > 7 && strings.ToUpper(ah[:7]) == "N3TWARS" {
				playerStr, err := ValidateToken(ah[7:])
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
