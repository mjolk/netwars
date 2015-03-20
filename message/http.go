package message

import (
	"appengine"
	"encoding/json"
	"mj0lk.be/netwars/utils"
	"net/http"
)

func CreateOrUpdateMessage(w http.ResponseWriter, r *http.Request) {
	message := new(Message)
	c := appengine.NewContext(r)
	json.NewDecoder(r.Body).Decode(message)
	if err := CreateOrUpdate(c, message); err != nil {
		res := utils.JSONResult{Success: false, Error: err.Error()}
		res.JSONf(w)
	}
}

func ListPublicBoards(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	pkey := r.FormValue("pkey")
	cursor := r.FormValue("cursor")
	var res utils.JSONResult
	boards, err := PublicBoards(c, pkey, cursor)
	if err != nil {
		res = utils.JSONResult{Success: false, Error: err.Error()}
	} else {
		res = utils.JSONResult{Success: true, Result: boards}
	}
	res.JSONf(w)
}

func ListClanBoards(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	pkey := r.FormValue("pkey")
	cursor := r.FormValue("cursor")
	var res utils.JSONResult
	boards, err := ClanBoards(c, pkey, cursor)
	if err != nil {
		res = utils.JSONResult{Success: false, Error: err.Error()}
	} else {
		res = utils.JSONResult{Success: true, Result: boards}
	}
	res.JSONf(w)
}

func ListThreads(w http.ResponseWriter, r *http.Request) {
	list(w, r, "threads")
}

func ListMessages(w http.ResponseWriter, r *http.Request) {
	list(w, r, "messages")
}

func list(w http.ResponseWriter, r *http.Request, tpe string) {
	c := appengine.NewContext(r)
	pkey := r.FormValue("pkey")
	tkey := r.FormValue("tkey")
	ckey := r.FormValue("ckey")
	var res utils.JSONResult
	messages, err := Messages(c, tpe, pkey, tkey, ckey)
	if err != nil {
		res = utils.JSONResult{Success: false, Error: err.Error()}
	} else {
		res = utils.JSONResult{Success: true, Result: messages}
	}
	res.JSONf(w)
}
