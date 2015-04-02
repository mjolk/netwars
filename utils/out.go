package utils

import (
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
	NoAccess = "No Access"
)

type contextKey int

const (
	JWTSECRET            = "blalxdjbvvszkcyh56^b-=9if%=h1e%$ld=@4(js50t!$ld*a@5vcu(=2d0jxvxkbgtnhiuk"
	keyCtx    contextKey = iota
)

type JSONResult struct {
	Success     bool        `json:"-"`
	EntityError bool        `json:"-"`
	Error       string      `json:"error"`
	Result      interface{} `json:"result, omitempty"`
}

func (r *JSONResult) JSONf(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if !r.Success {
		if r.EntityError {
			w.WriteHeader(422)
		} else {
			switch r.Error {
			case NoAccess:
				w.WriteHeader(http.StatusUnauthorized)
			default:
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	}
	if err := json.NewEncoder(w).Encode(r); err != nil {
		panic(err)
	}
}

func CreateTokenString(playerKey string) (string, error) {
	token := jwt.New(jwt.GetSigningMethod("HS256"))
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
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(JWTSECRET), nil
	})
	if err == nil && token.Valid {
		playerStr = token.Claims["pkey"].(string)
	} else if err != nil {
		verr := err.(jwt.ValidationError)
		if jwt.ValidationErrorExpired == jwt.ValidationErrorExpired&verr.Errors {
			//TODO expired : regenerate???
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
