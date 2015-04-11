package clan

import (
	"appengine"
	"appengine/blobstore"
	"mj0lk.be/netwars/utils"
	"net/http"
)

func Invites(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	player := utils.Pkey(r)
	var res utils.JSONResult
	invites, err := InvitesForPlayer(c, player)
	if err != nil {
		res = utils.JSONResult{Success: false, StatusCode: http.StatusInternalServerError,
			Error: err.Error()}
	}
	res = utils.JSONResult{Success: true, StatusCode: http.StatusOK, Result: invites}
	res.JSONf(w)
}

func ClanStatus(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	team := new(Clan)
	pkeyStr := utils.Pkey(r)
	var res utils.JSONResult
	if err := Status(c, pkeyStr, team); err != nil {
		res = utils.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
	}
	res = utils.JSONResult{Success: true, StatusCode: http.StatusOK, Result: team}
	res.JSONf(w)
}

func PublicClanStatus(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	team := new(Clan)
	pkeyStr := utils.Pkey(r)
	clanStr := utils.Var(r, "clanid")
	var res utils.JSONResult
	if err := PublicStatus(c, pkeyStr, clanStr, team); err != nil {
		res = utils.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
	}
	res = utils.JSONResult{Success: true, StatusCode: http.StatusOK, Result: team}
	res.JSONf(w)
}

func GetClanList(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	cur := utils.Var(r, "cursor")
	playerKey := utils.Pkey(r)
	attackRange := utils.Var(r, "rangebool")
	list, err := List(c, playerKey, attackRange, cur)
	var res utils.JSONResult
	if err != nil {
		res = utils.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
	} else {
		res = utils.JSONResult{Success: true, StatusCode: http.StatusOK, Result: list}
	}
	res.JSONf(w)
}

func CancelPlayerInvite(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	sk := SendKey{}
	var res utils.JSONResult
	if err := utils.DecodeJsonBody(r, &sk); err != nil {
		res = utils.JSONResult{Success: false, StatusCode: 422, Error: err.Error()}
		res.JSONf(w)
		return
	}
	playerStr := utils.Pkey(r)
	inviteKeyStr := sk.Key
	if err := CancelInvite(c, playerStr, inviteKeyStr); err != nil {
		res = utils.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
		res.JSONf(w)
	}
}

func EditLeaderShip(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	var res utils.JSONResult
	prom := Promotion{}
	if err := utils.DecodeJsonBody(r, &prom); err != nil {
		res = utils.JSONResult{Success: false, StatusCode: 422, Error: err.Error()}
		res.JSONf(w)
		return
	}
	playerStr := utils.Pkey(r)
	subject := prom.PlayerID
	rank := prom.Rank
	if err := PromoteOrDemote(c, playerStr, subject, rank); err != nil {
		res := utils.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
		res.JSONf(w)
	}
}

func KickPlayer(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	p := SendID{}
	if err := utils.DecodeJsonBody(r, &p); err != nil {
		res := utils.JSONResult{Success: false, StatusCode: 422, Error: err.Error()}
		res.JSONf(w)
		return
	}
	playerStr := utils.Pkey(r)
	subject := p.ID
	if err := Kick(c, playerStr, subject); err != nil {
		res := utils.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
		res.JSONf(w)
	}
}

func EditMessage(w http.ResponseWriter, r *http.Request) {
	messageUpdate := new(MessageUpdate)
	c := appengine.NewContext(r)
	if err := utils.DecodeJsonBody(r, &messageUpdate); err != nil {
		res := utils.JSONResult{Success: false, StatusCode: 422, Error: err.Error()}
		res.JSONf(w)
		return
	}
	playerStr := utils.Pkey(r)
	if err := UpdateMessage(c, playerStr, messageUpdate); err != nil {
		res := utils.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
		res.JSONf(w)
	}
}

func ClanConnect(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	clc := SendID{}
	if err := utils.DecodeJsonBody(r, &clc); err != nil {
		res := utils.JSONResult{Success: false, StatusCode: 422, Error: err.Error()}
		res.JSONf(w)
		return
	}
	playerStr := utils.Pkey(r)
	subject := clc.ID
	if err := Connect(c, playerStr, subject); err != nil {
		res := utils.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
		res.JSONf(w)
	}
}

func ClanDisConnect(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	clc := SendID{}
	if err := utils.DecodeJsonBody(r, &clc); err != nil {
		res := utils.JSONResult{Success: false, StatusCode: 422, Error: err.Error()}
		res.JSONf(w)
		return
	}
	playerStr := utils.Pkey(r)
	subject := clc.ID
	if err := DisConnect(c, playerStr, subject); err != nil {
		res := utils.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
		res.JSONf(w)
	}
}

func LeaveClan(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	playerStr := utils.Pkey(r)
	if err := Leave(c, playerStr); err != nil {
		res := utils.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
		res.JSONf(w)
	}
}

func JoinClan(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	b := SendKey{}
	if err := utils.DecodeJsonBody(r, &b); err != nil {
		res := utils.JSONResult{Success: false, StatusCode: 422, Error: err.Error()}
		res.JSONf(w)
		return
	}
	invite := b.Key
	player := utils.Pkey(r)
	if err := Join(c, player, invite); err != nil {
		res := utils.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
		res.JSONf(w)
	}
}

func ClanInvite(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	p := SendID{}
	if err := utils.DecodeJsonBody(r, &p); err != nil {
		res := utils.JSONResult{Success: false, StatusCode: 422, Error: err.Error()}
		res.JSONf(w)
		return
	}
	player := utils.Pkey(r)
	invitee := p.ID
	if err := InvitePlayer(c, player, invitee); err != nil {
		res := utils.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
		res.JSONf(w)
	}
}

func CreateClan(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	cr := Creation{}
	if err := utils.DecodeJsonBody(r, &cr); err != nil {
		res := utils.JSONResult{Success: false, StatusCode: 422, Error: err.Error()}
		res.JSONf(w)
		return
	}
	clanName := cr.Name
	player := utils.Pkey(r)
	tag := cr.Tag
	_, errmap, err := Create(c, player, clanName, tag)
	if err != nil {
		res := utils.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
		res.JSONf(w)
	} else if errmap["clan_name"]+errmap["clan_tag"] > 0 {
		res := utils.JSONResult{Success: false, StatusCode: http.StatusConflict, Result: errmap}
		res.JSONf(w)
	}
}

func UploadAvatar(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	res := utils.JSONResult{}
	blobs, _, err := blobstore.ParseUpload(r)
	if err != nil {
		res = utils.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
	} else {
		file := blobs["avatar"]
		if len(file) == 0 {
			res = utils.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: "No Image Uploaded"}
		} else {
			img := file[0]
			if utils.IsNotImage(img) {
				res = utils.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: "No Image Uploaded"}
			} else {
				playerStr := utils.Pkey(r)
				if err := UpdateAvatar(c, playerStr, img); err != nil {
					res = utils.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
				}
			}
		}

	}
	if len(res.Error) > 0 {
		res.JSONf(w)
	}

}

func EditAvatar(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	uploadURL, err := blobstore.UploadURL(c, "/clans/avatar", nil)
	var res utils.JSONResult
	if err != nil {
		res = utils.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
	} else {
		res = utils.JSONResult{Success: true, StatusCode: http.StatusOK, Result: uploadURL}
	}
	res.JSONf(w)
}
