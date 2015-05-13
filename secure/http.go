package secure

/*import (
	"appengine"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"mj0lk.be/netwars/app"
	"net/http"
)

const (
	signingString = "3A391e3c70409fc63b80d38ff74e6da9e3lksnlkjngknsklfvlkasbnjlljnkl49u858946u94u9uhkldfnn4j5ihj4jjofbo5"
)

func VerifySecure(w http.ResponseWriter, r *http.Request, c app.Context) {
	var res app.JSONResult
	var err error
	var keyName string
	var sigBytes []byte
	var usedCert appengine.Certificate
	var parsedCert *x509.Certificate
	token := c.Param("token")
	keyName = c.Param("keyname")
	sigBytes, err = base64.URLEncoding.DecodeString(token)
	loadCertificates(c)
	usedCert, err = getCertificate(keyName)
	block, _ := pem.Decode(usedCert.Data)
	if block == nil {
		err = errors.New("no block")
	}
	parsedCert, err = x509.ParseCertificate(block.Bytes)
	err = parsedCert.CheckSignature(x509.SHA256WithRSA, []byte(signingString), sigBytes)
	if err != nil {
		res = app.JSONResult{Success: false, Error: err.Error()}
	} else {
		res = app.JSONResult{Success: true}
	}
	res.JSONf(w)
}

func GetToken(w http.ResponseWriter, r *http.Request, c app.Context) {
	var res app.JSONResult
	keyName, sigbytes, err := appengine.SignBytes(c, []byte(signingString))
	if err != nil {
		res = app.JSONResult{Success: false, Error: err.Error()}
	} else {
		tokenString := base64.URLEncoding.EncodeToString(sigbytes)
		url := fmt.Sprintf("https://n3twars.appspot.com/load/token/%s/%s", tokenString, keyName)
		res = app.JSONResult{Success: true, Error: keyName, Result: url}
	}
	res.JSONf(w)
}*/
