package message

import (
	"appengine"
	"mj0lk.be/netwars/utils"
	"net/http"
)

func CreateOrUpdateMessage(w http.ResponseWriter, r *http.Request) {
	message := Message{}
	c := appengine.NewContext(r)
	if err := utils.DecodeJsonBody(r, &message); err != nil {
		res := utils.JSONResult{Success: false, EntityError: true, Error: err.Error()}
		res.JSONf(w)
		return
	}
	playerStr := utils.Pkey(r)
	if err := CreateOrUpdate(c, playerStr, message); err != nil {
		res := utils.JSONResult{Success: false, Error: err.Error()}
		res.JSONf(w)
	}
}

func ListPublicBoards(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	pkey := utils.Pkey(r)
	cursor := utils.Var(r, "cursor")
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
	pkey := utils.Pkey(r)
	cursor := utils.Var(r, "cursor")
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
	pkey := utils.Pkey(r)
	var key string
	switch tpe {
	case "threads":
		key = utils.Var(r, "bkey")
	case "messages":
		key = utils.Var(r, "tkey")
	}
	ckey := utils.Var(r, "cursor")
	var res utils.JSONResult
	messages, err := Messages(c, tpe, pkey, key, ckey)
	if err != nil {
		res = utils.JSONResult{Success: false, Error: err.Error()}
	} else {
		res = utils.JSONResult{Success: true, Result: messages}
	}
	res.JSONf(w)
}
