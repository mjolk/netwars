package utils

import (
	"appengine"
	"appengine/blobstore"
	"encoding/json"
	"errors"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var (
	ImageTypes = []string{
		"image/bmp",
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/tiff",
		"image/x-icon",
	}
	NoAccess     = "No Access"
	certificates []*appengine.Certificate
)

type contextKey int

const (
	JWTSECRET            = "blalxdjbvvszkcyh56^b-=9if%=h1e%$ld=@4(js50t!$ld*a@5vcu(=2d0jxvxkbgtnhiuk"
	keyCtx    contextKey = iota
)

type JSONResult struct {
	Success    bool        `json:"-"`
	StatusCode int         `json:"-"`
	Error      string      `json:"error"`
	Result     interface{} `json:"result, omitempty"`
}

func (r *JSONResult) JSONf(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if !r.Success {
		w.WriteHeader(r.StatusCode)
	}
	if err := json.NewEncoder(w).Encode(r); err != nil {
		panic(err)
	}
}

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
	token.Claims["exp"] = time.Now().Add(time.Hour * 168).Unix()
	tokenString, err := token.SignedString([]byte(JWTSECRET))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func DecodeJsonBody(r *http.Request, v interface{}) error {
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		return err
	}
	if err := r.Body.Close(); err != nil {
		return err
	}
	if err := json.Unmarshal(body, v); err != nil {
		return err
	}
	return nil
}

func Pkey(r *http.Request) string {
	pkey := context.Get(r, keyCtx)
	return pkey.(string)
}

func Var(r *http.Request, vr string) string {
	vars := mux.Vars(r)
	if ret, ok := vars[vr]; ok {
		return ret
	}
	return ""
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

func noAccess(w http.ResponseWriter) {
	res := JSONResult{Success: false, Error: NoAccess}
	res.JSONf(w)
}

func Validator(inner http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ah := r.Header.Get("Authorization"); ah != "" {
			// Should be a netwars token
			if len(ah) > 7 && strings.ToUpper(ah[:7]) == "N3TWARS" {
				playerStr, err := ValidateToken(ah[7:])
				if err != nil {
					noAccess(w)
				} else {
					context.Set(r, keyCtx, playerStr)
					inner.ServeHTTP(w, r)
				}
			} else {
				noAccess(w)
			}
		} else {
			noAccess(w)
		}
	})
}

func IsNotImage(data *blobstore.BlobInfo) bool {
	for _, tpe := range ImageTypes {
		if data.ContentType == tpe {
			return false
		}
	}
	return true
}
