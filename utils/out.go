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

const JWTSECRET = "blalxdjbvvszkcyh56^b-=9if%=h1e%$ld=@4(js50t!$ld*a@5vcu(=2d0jxvxkbgtnhiuk"

type JSONResult struct {
	Success     bool        `json:"-"`
	EntityError bool        `json:"-"`
	Error       string      `json:"error"`
	Result      interface{} `json:"result, omitempty"`
}

func (r *JSONResult) JSONf(w http.ResponseWriter) {
	if err := json.NewEncoder(w).Encode(r); err != nil {
		panic(err)
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if !r.Success {
		w.WriteHeader(http.StatusInternalServerError)
		if r.EntityError {
			w.WriteHeader(422)
		}
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
	vars := mux.Vars(r)
	return vars["pkey"]
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

func Validator(inner http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var res JSONResult
		if ah := r.Header.Get("Authorization"); ah != "" {
			// Should be a netwars token
			if len(ah) > 7 && strings.ToUpper(ah[:7]) == "N3TWARS" {
				playerStr, err := ValidateToken(ah[7:])
				if err != nil {
					if rv := context.Get(r, 0); rv != nil {
						rvmap := rv.(map[string]string)
						rvmap["pkey"] = playerStr
						context.Set(r, 0, rv)
					} else {
						rv := make(map[string]string)
						rv["pkey"] = playerStr
						context.Set(r, 0, rv)
					}
					inner.ServeHTTP(w, r)
				} else {
					res = JSONResult{Success: false, Error: NoAccess}
				}

			} else {
				res = JSONResult{Success: false, Error: NoAccess}
			}
		} else {
			res = JSONResult{Success: false, Error: NoAccess}
		}
		res.JSONf(w)
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
