package message

import (
	"mj0lk.be/netwars/app"
	"net/http"
)

func CreateOrUpdateMessage(w http.ResponseWriter, r *http.Request, c app.Context) {
	message := Message{}
	if err := app.DecodeJsonBody(r, &message); err != nil {
		res := app.JSONResult{Success: false, StatusCode: 422, Error: err.Error()}
		res.JSONf(w)
		return
	}
	if err := CreateOrUpdate(c, c.User, message); err != nil {
		res := app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
		res.JSONf(w)
	}
}

func ListPublicBoards(w http.ResponseWriter, r *http.Request, c app.Context) {
	var res app.JSONResult
	boards, err := PublicBoards(c, c.User, c.Param("cursor_key"))
	if err != nil {
		res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
	} else {
		res = app.JSONResult{Success: true, StatusCode: http.StatusOK, Result: boards}
	}
	res.JSONf(w)
}

func ListClanBoards(w http.ResponseWriter, r *http.Request, c app.Context) {
	var res app.JSONResult
	boards, err := ClanBoards(c, c.User, c.Param("cursor_key"))
	if err != nil {
		res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
	} else {
		res = app.JSONResult{Success: true, StatusCode: http.StatusOK, Result: boards}
	}
	res.JSONf(w)
}

func ListThreads(w http.ResponseWriter, r *http.Request, c app.Context) {
	list(w, r, "threads", c)
}

func ListMessages(w http.ResponseWriter, r *http.Request, c app.Context) {
	list(w, r, "messages", c)
}

func list(w http.ResponseWriter, r *http.Request, tpe string, c app.Context) {
	var key string
	switch tpe {
	case "threads":
		key = c.Param("b_key")
	case "messages":
		key = c.Param("t_key")
	}
	var res app.JSONResult
	messages, err := Messages(c, tpe, c.User, key, c.Param("cursor_key"))
	if err != nil {
		res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
	} else {
		res = app.JSONResult{Success: true, StatusCode: http.StatusOK, Result: messages}
	}
	res.JSONf(w)
}
