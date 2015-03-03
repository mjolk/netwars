package clan

import (
	"appengine"
	"appengine/blobstore"
	"encoding/json"
	"mj0lk.be/netwars/utils"
	"net/http"
)

func Invites(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	player := r.FormValue("pkey")
	var res utils.JSONResult
	invites, err := InvitesForPlayer(c, player)
	if err != nil {
		res = utils.JSONResult{Success: false, Error: err.Error()}
	}
	res = utils.JSONResult{Success: true, Result: invites}
	res.JSONf(w)
}

func ClanStatus(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	team := new(Clan)
	pkeyStr := r.FormValue("pkey")
	var res utils.JSONResult
	if err := Status(c, pkeyStr, team); err != nil {
		res = utils.JSONResult{Success: false, Error: err.Error()}
	}
	res = utils.JSONResult{Success: true, Result: team}
	res.JSONf(w)
}

func EditLeaderShip(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	player := r.FormValue("pkey")
	subject := r.FormValue("target")
	rank := r.FormValue("rank")
	if err := PromoteOrDemote(c, player, subject, rank); err != nil {
		res := utils.JSONResult{Success: false, Error: err.Error()}
		res.JSONf(w)
	}
}

func KickPlayer(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	player := r.FormValue("pkey")
	subject := r.FormValue("target")
	if err := Kick(c, player, subject); err != nil {
		res := utils.JSONResult{Success: false, Error: err.Error()}
		res.JSONf(w)
	}
}

func EditMessage(w http.ResponseWriter, r *http.Request) {
	messageUpdate := new(MessageUpdate)
	c := appengine.NewContext(r)
	json.NewDecoder(r.Body).Decode(messageUpdate)
	if err := UpdateMessage(c, messageUpdate); err != nil {
		res := utils.JSONResult{Success: false, Error: err.Error()}
		res.JSONf(w)
	}
}

func ClanConnect(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	player := r.FormValue("pkey")
	subject := r.FormValue("d_clan")
	if err := Connect(c, player, subject); err != nil {
		res := utils.JSONResult{Success: false, Error: err.Error()}
		res.JSONf(w)
	}
}

func ClanDisConnect(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	player := r.FormValue("pkey")
	subject := r.FormValue("conn")
	if err := DisConnect(c, player, subject); err != nil {
		res := utils.JSONResult{Success: false, Error: err.Error()}
		res.JSONf(w)
	}
}

func LeaveClan(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	player := r.FormValue("pkey")
	if err := Leave(c, player); err != nil {
		res := utils.JSONResult{Success: false, Error: err.Error()}
		res.JSONf(w)
	}
}

func JoinClan(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	invite := r.FormValue("invite")
	player := r.FormValue("pkey")
	if err := Join(c, player, invite); err != nil {
		res := utils.JSONResult{Success: false, Error: err.Error()}
		res.JSONf(w)
	}
}

func ClanInvite(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	player := r.FormValue("pkey")
	invitee := r.FormValue("invitee")
	if err := InvitePlayer(c, player, invitee); err != nil {
		res := utils.JSONResult{Success: false, Error: err.Error()}
		res.JSONf(w)
	}
}

func ClanCreate(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	clanName := r.FormValue("name")
	player := r.FormValue("pkey")
	tag := r.FormValue("tag")
	_, errmap, err := Create(c, player, clanName, tag)
	if err != nil {
		res := utils.JSONResult{Success: false, Error: err.Error()}
		res.JSONf(w)
	} else if errmap["clan_name"]+errmap["clan_tag"] > 0 {
		res := utils.JSONResult{Success: false, Result: errmap}
		res.JSONf(w)
	}
}

func UploadAvatar(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	res := utils.JSONResult{}
	blobs, values, err := blobstore.ParseUpload(r)
	if err != nil {
		res = utils.JSONResult{Success: false, Error: err.Error()}
	} else {
		file := blobs["avatar"]
		if len(file) == 0 {
			res = utils.JSONResult{Success: false, Error: "No Image Uploaded"}
		} else {
			img := file[0]
			if utils.IsNotImage(img) {
				res = utils.JSONResult{Success: false, Error: "No Image Uploaded"}
			} else {
				player := values["pkey"]
				if err := UpdateAvatar(c, player[0], img); err != nil {
					res = utils.JSONResult{Success: false, Error: err.Error()}
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
	uploadURL, err := blobstore.UploadURL(c, "/clan_uploadavatar", nil)
	var res utils.JSONResult
	if err != nil {
		res = utils.JSONResult{Success: false, Error: err.Error()}
	} else {
		res = utils.JSONResult{Success: true, Result: uploadURL}
	}
	res.JSONf(w)
}
