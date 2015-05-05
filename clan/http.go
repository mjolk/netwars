package clan

import (
	"appengine/blobstore"
	"mj0lk.be/netwars/app"
	"net/http"
)

func Invites(w http.ResponseWriter, r *http.Request, c app.Context) {
	var res app.JSONResult
	invites, err := InvitesForPlayer(c, c.User)
	if err != nil {
		res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError,
			Error: err.Error()}
	}
	res = app.JSONResult{Success: true, StatusCode: http.StatusOK, Result: invites}
	res.JSONf(w)
}

func ClanStatus(w http.ResponseWriter, r *http.Request, c app.Context) {
	team := new(Clan)
	var res app.JSONResult
	if err := Status(c, c.User, team); err != nil {
		res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
	}
	res = app.JSONResult{Success: true, StatusCode: http.StatusOK, Result: team}
	res.JSONf(w)
}

func PublicClanStatus(w http.ResponseWriter, r *http.Request, c app.Context) {
	team := new(Clan)
	var res app.JSONResult
	if err := PublicStatus(c, c.User, c.Param("clan_id"), team); err != nil {
		res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
	}
	res = app.JSONResult{Success: true, StatusCode: http.StatusOK, Result: team}
	res.JSONf(w)
}

func GetClanList(w http.ResponseWriter, r *http.Request, c app.Context) {
	list, err := List(c, c.User, c.Param("range_bool"), c.Param("cursor_key"))
	var res app.JSONResult
	if err != nil {
		res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
	} else {
		res = app.JSONResult{Success: true, StatusCode: http.StatusOK, Result: list}
	}
	res.JSONf(w)
}

func CancelPlayerInvite(w http.ResponseWriter, r *http.Request, c app.Context) {
	sk := SendKey{}
	var res app.JSONResult
	if err := app.DecodeJsonBody(r, &sk); err != nil {
		res = app.JSONResult{Success: false, StatusCode: 422, Error: err.Error()}
		res.JSONf(w)
		return
	}
	if err := CancelInvite(c, c.User, sk.Key); err != nil {
		res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
		res.JSONf(w)
	}
}

func EditLeaderShip(w http.ResponseWriter, r *http.Request, c app.Context) {
	var res app.JSONResult
	prom := Promotion{}
	if err := app.DecodeJsonBody(r, &prom); err != nil {
		res = app.JSONResult{Success: false, StatusCode: 422, Error: err.Error()}
		res.JSONf(w)
		return
	}
	if err := PromoteOrDemote(c, c.User, prom.PlayerID, prom.Rank); err != nil {
		res := app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
		res.JSONf(w)
	}
}

func KickPlayer(w http.ResponseWriter, r *http.Request, c app.Context) {
	p := SendID{}
	if err := app.DecodeJsonBody(r, &p); err != nil {
		res := app.JSONResult{Success: false, StatusCode: 422, Error: err.Error()}
		res.JSONf(w)
		return
	}
	if err := Kick(c, c.User, p.ID); err != nil {
		res := app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
		res.JSONf(w)
	}
}

func EditMessage(w http.ResponseWriter, r *http.Request, c app.Context) {
	messageUpdate := new(MessageUpdate)
	if err := app.DecodeJsonBody(r, &messageUpdate); err != nil {
		res := app.JSONResult{Success: false, StatusCode: 422, Error: err.Error()}
		res.JSONf(w)
		return
	}
	if err := UpdateMessage(c, c.User, messageUpdate); err != nil {
		res := app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
		res.JSONf(w)
	}
}

func ClanConnect(w http.ResponseWriter, r *http.Request, c app.Context) {
	clc := SendID{}
	if err := app.DecodeJsonBody(r, &clc); err != nil {
		res := app.JSONResult{Success: false, StatusCode: 422, Error: err.Error()}
		res.JSONf(w)
		return
	}
	if err := Connect(c, c.User, clc.ID); err != nil {
		res := app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
		res.JSONf(w)
	}
}

func ClanDisConnect(w http.ResponseWriter, r *http.Request, c app.Context) {
	clc := SendID{}
	if err := app.DecodeJsonBody(r, &clc); err != nil {
		res := app.JSONResult{Success: false, StatusCode: 422, Error: err.Error()}
		res.JSONf(w)
		return
	}
	if err := DisConnect(c, c.User, clc.ID); err != nil {
		res := app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
		res.JSONf(w)
	}
}

func LeaveClan(w http.ResponseWriter, r *http.Request, c app.Context) {
	if err := Leave(c, c.User); err != nil {
		res := app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
		res.JSONf(w)
	}
}

func JoinClan(w http.ResponseWriter, r *http.Request, c app.Context) {
	b := SendKey{}
	if err := app.DecodeJsonBody(r, &b); err != nil {
		res := app.JSONResult{Success: false, StatusCode: 422, Error: err.Error()}
		res.JSONf(w)
		return
	}
	if err := Join(c, c.User, b.Key); err != nil {
		res := app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
		res.JSONf(w)
	}
}

func ClanInvite(w http.ResponseWriter, r *http.Request, c app.Context) {
	p := SendID{}
	if err := app.DecodeJsonBody(r, &p); err != nil {
		res := app.JSONResult{Success: false, StatusCode: 422, Error: err.Error()}
		res.JSONf(w)
		return
	}
	if err := InvitePlayer(c, c.User, p.ID); err != nil {
		res := app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
		res.JSONf(w)
	}
}

func CreateClan(w http.ResponseWriter, r *http.Request, c app.Context) {
	cr := Creation{}
	res := app.JSONResult{}
	if err := app.DecodeJsonBody(r, &cr); err != nil {
		res = app.JSONResult{Success: false, StatusCode: 422, Error: err.Error()}
		res.JSONf(w)
		return
	}
	_, errmap, err := Create(c, c.User, cr.Name, cr.Tag)
	if err != nil {
		res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
		res.JSONf(w)
	} else if errmap["clan_name"]+errmap["clan_tag"] > 0 {
		if errmap["clan_name"] > 0 {
			res = app.JSONResult{Success: false, StatusCode: http.StatusConflict, Result: "name error"}
		} else {
			res = app.JSONResult{Success: false, StatusCode: http.StatusConflict, Result: "tag error"}
		}
		res.JSONf(w)
	}
}

func UploadAvatar(w http.ResponseWriter, r *http.Request, c app.Context) {
	res := app.JSONResult{}
	blobs, _, err := blobstore.ParseUpload(r)
	if err != nil {
		res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
	} else {
		file := blobs["avatar"]
		if len(file) == 0 {
			res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: "No Image Uploaded"}
		} else {
			img := file[0]
			if app.IsNotImage(img) {
				res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: "No Image Uploaded"}
			} else {
				if err := UpdateAvatar(c, c.User, img); err != nil {
					res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
				}
			}
		}

	}
	if len(res.Error) > 0 {
		res.JSONf(w)
	}

}

func EditAvatar(w http.ResponseWriter, r *http.Request, c app.Context) {
	uploadURL, err := blobstore.UploadURL(c, "/clans/avatar", nil)
	var res app.JSONResult
	if err != nil {
		res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
	} else {
		res = app.JSONResult{Success: true, StatusCode: http.StatusOK, Result: uploadURL}
	}
	res.JSONf(w)
}
