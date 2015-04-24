package attack

import (
	"mj0lk.be/netwars/app"
	"net/http"
)

func AttackPlayer(w http.ResponseWriter, r *http.Request, c app.Context) {
	cfg := AttackCfg{}
	var res app.JSONResult
	if err := app.DecodeJsonBody(r, &cfg); err != nil {
		res = app.JSONResult{Success: false, StatusCode: 422, Error: err.Error()}
	} else {
		var response AttackEvent
		var err error
		switch cfg.AttackType {
		case BAL:
			response, err = Attack(c, c.User, cfg)
		case ICE:
			response, err = Ice(c, c.User, cfg)
		case INT:
			response, err = Spy(c, c.User, cfg)
		}
		if err != nil {
			res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
			c.Debugf("result switch %+v\n", res)
		} else {
			res = app.JSONResult{Success: true, StatusCode: http.StatusOK, Result: response}
		}
	}
	res.JSONf(w)
}
