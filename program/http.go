package program

import (
	"appengine"
	"encoding/json"
	"mj0lk.be/netwars/utils"
	"net/http"
)

func GetAllPrograms(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	programs := make(map[string][]Program)
	var res utils.JSONResult
	if err := GetAll(c, programs); err != nil {
		res = utils.JSONResult{Success: false, Error: err.Error()}
	} else {
		res = utils.JSONResult{Success: true, Result: programs}
	}
	res.JSONf(w)
}

func GetProgram(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	pKey := utils.Pkey(r)
	var res utils.JSONResult
	program, err := Get(c, pKey)
	if err != nil {
		res = utils.JSONResult{Success: false, Error: err.Error()}
	} else {
		res = utils.JSONResult{Success: true, Result: program}
	}
	res.JSONf(w)
}

func CreateOrUpdateProgram(w http.ResponseWriter, r *http.Request) {
	program := new(Program)
	c := appengine.NewContext(r)
	json.NewDecoder(r.Body).Decode(program)
	if err := CreateOrUpdate(c, program); err != nil {
		res := utils.JSONResult{Success: false, Error: err.Error()}
		res.JSONf(w)
	}
}

func LoadPrograms(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	var res utils.JSONResult
	if err := LoadFromFile(c); err != nil {
		res = utils.JSONResult{Success: false, Error: err.Error()}
	} else {
		res = utils.JSONResult{Success: true}
	}
	res.JSONf(w)
}
