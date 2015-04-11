package attack

import (
	"appengine"
	"mj0lk.be/netwars/utils"
	"net/http"
)

func AttackPlayer(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	cfg := AttackCfg{}
	playerStr := utils.Pkey(r)
	var res utils.JSONResult
	if err := utils.DecodeJsonBody(r, &cfg); err != nil {
		res = utils.JSONResult{Success: false, StatusCode: 422, Error: err.Error()}
	} else {
		var response AttackEvent
		var err error
		switch cfg.AttackType {
		case BAL:
			response, err = Attack(c, playerStr, cfg)
		case ICE:
			response, err = Ice(c, playerStr, cfg)
		case INT:
			response, err = Spy(c, playerStr, cfg)
		}
		if err != nil {
			res = utils.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
			c.Debugf("result switch %+v\n", res)
		} else {
			res = utils.JSONResult{Success: true, StatusCode: http.StatusOK, Result: response}
		}
	}
	res.JSONf(w)
}
