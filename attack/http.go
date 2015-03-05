package attack

import (
	"appengine"
	"encoding/json"
	"net/http"
	"netwars/utils"
)

func init() {
	r := utils.Router()
	r.HandleFunc("/attack", AttackPlayer).Methods("POST")
}

func AttackPlayer(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	cfg := AttackCfg{}
	var res utils.JSONResult
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		res = utils.JSONResult{Success: false, Error: err.Error()}
	} else {
		var response Response
		var err error
		if AttackType[cfg.AttackType]&BAL != 0 {
			response, err = Attack(c, cfg)
			c.Debugf("error attack %s\n", err)
		} else if AttackType[cfg.AttackType]&ICE != 0 {
			response, err = Ice(c, cfg)
		} else if AttackType[cfg.AttackType]&INT != 0 {
			response, err = Spy(c, cfg)
		}
		if err != nil {
			res = utils.JSONResult{Success: false, Error: err.Error()}
			c.Debugf("result switch %+v\n", res)
		} else {
			res = utils.JSONResult{Success: true, Result: response}
		}
	}
	res.JSONf(w)
}
