package program

import (
	"encoding/json"
	"mj0lk.be/netwars/app"
	"net/http"
)

func GetAllPrograms(w http.ResponseWriter, r *http.Request, c app.Context) {
	programs := make(map[string][]Program)
	var res app.JSONResult
	if err := GetAll(c, programs); err != nil {
		res = app.JSONResult{Success: false, Error: err.Error()}
	} else {
		res = app.JSONResult{Success: true, Result: programs}
	}
	res.JSONf(w)
}

func GetProgram(w http.ResponseWriter, r *http.Request, c app.Context) {
	var res app.JSONResult
	program, err := Get(c, c.User)
	if err != nil {
		res = app.JSONResult{Success: false, Error: err.Error()}
	} else {
		res = app.JSONResult{Success: true, Result: program}
	}
	res.JSONf(w)
}

func CreateOrUpdateProgram(w http.ResponseWriter, r *http.Request, c app.Context) {
	program := new(Program)
	json.NewDecoder(r.Body).Decode(program)
	if err := CreateOrUpdate(c, program); err != nil {
		res := app.JSONResult{Success: false, Error: err.Error()}
		res.JSONf(w)
	}
}

func LoadPrograms(w http.ResponseWriter, r *http.Request, c app.Context) {
	var res app.JSONResult
	if err := LoadFromFile(c); err != nil {
		res = app.JSONResult{Success: false, Error: err.Error()}
	} else {
		res = app.JSONResult{Success: true}
	}
	res.JSONf(w)
}
