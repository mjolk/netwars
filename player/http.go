package player

import (
	"appengine"
	"appengine/blobstore"
	"fmt"
	"mj0lk.be/netwars/utils"
	"net/http"
)

func EditProfile(w http.ResponseWriter, r *http.Request) {
	update := ProfileUpdate{}
	c := appengine.NewContext(r)
	var res utils.JSONResult
	if err := utils.DecodeJsonBody(r, &update); err != nil {
		res = utils.JSONResult{Success: false, EntityError: true, Error: err.Error()}
		res.JSONf(w)
		return
	}
	playerStr := utils.Pkey(r)
	if err := UpdateProfile(c, playerStr, update); err != nil {
		res = utils.JSONResult{Success: false, Error: err.Error()}
		res.JSONf(w)
	}
}

func CreatePlayer(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	cr := Creation{}
	var res utils.JSONResult
	if err := utils.DecodeJsonBody(r, &cr); err != nil {
		res = utils.JSONResult{Success: false, EntityError: true, Error: err.Error()}
		res.JSONf(w)
		return
	}
	enckey, errmap, err := Create(c, cr)
	if err != nil {
		res = utils.JSONResult{Success: false, Error: err.Error()}
	} else if len(errmap) > 0 {
		res = utils.JSONResult{Success: false, Error: fmt.Sprint("errmap: %+v", errmap)}
	} else {
		res = utils.JSONResult{Success: true, Result: enckey}

	}
	res.JSONf(w)
}

func GetPlayerList(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	cur := utils.Var(r, "cursor")
	playerKey := utils.Pkey(r)
	attackRange := utils.Var(r, "range")
	list, err := List(c, playerKey, attackRange, cur)
	var res utils.JSONResult
	if err != nil {
		res = utils.JSONResult{Success: false, Error: err.Error()}
	} else {
		res = utils.JSONResult{Success: true, Result: list}
	}
	res.JSONf(w)
}

func StatusPlayer(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	playerStr := utils.Pkey(r)
	iplayer := new(Player)
	var res utils.JSONResult
	if err := Tstatus(c, playerStr, iplayer); err != nil {
		res = utils.JSONResult{Success: false, Error: err.Error()}
	} else {
		res = utils.JSONResult{Success: true, Result: iplayer}
	}
	res.JSONf(w)
}

func AllocatePrograms(w http.ResponseWriter, r *http.Request) {
	al := Allocation{}
	var res utils.JSONResult
	if err := utils.DecodeJsonBody(r, &al); err != nil {
		res = utils.JSONResult{Success: false, EntityError: false, Error: err.Error()}
		res.JSONf(w)
		return
	}
	c := appengine.NewContext(r)
	playerStr := utils.Pkey(r)
	if err := Allocate(c, playerStr, al); err != nil {
		c.Debugf("error allocating %s \n", err)
		res = utils.JSONResult{Success: false, Error: err.Error()}
		res.JSONf(w)
	}
}

func AuthenticatePlayer(w http.ResponseWriter, r *http.Request) {
	al := Authentication{}
	var res utils.JSONResult
	if err := utils.DecodeJsonBody(r, &al); err != nil {
		res = utils.JSONResult{Success: false, EntityError: false, Error: err.Error()}
		res.JSONf(w)
		return
	}
	c := appengine.NewContext(r)
	if token, err := Login(c, al); err != nil {
		c.Debugf("error login %s \n", err)
		res = utils.JSONResult{Success: false, Error: err.Error()}
	} else {
		res = utils.JSONResult{Success: true, Result: token, Error: err.Error()}
	}
	res.JSONf(w)
}

func DeallocatePrograms(w http.ResponseWriter, r *http.Request) {
	var res utils.JSONResult
	al := Allocation{}
	if err := utils.DecodeJsonBody(r, &al); err != nil {
		res = utils.JSONResult{Success: false, EntityError: true, Error: err.Error()}
		res.JSONf(w)
		return
	}
	playerStr := utils.Pkey(r)
	c := appengine.NewContext(r)
	if err := Deallocate(c, playerStr, al); err != nil {
		res = utils.JSONResult{Success: false, Error: err.Error()}
		res.JSONf(w)
	}
}

func UploadAvatar(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	res := utils.JSONResult{}
	blobs, _, err := blobstore.ParseUpload(r)
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
				if err := UpdateAvatar(c, utils.Pkey(r), img); err != nil {
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
	uploadURL, err := blobstore.UploadURL(c, "/players/avatar", nil)
	var res utils.JSONResult
	if err != nil {
		res = utils.JSONResult{Success: false, Error: err.Error()}
	} else {
		res = utils.JSONResult{Success: true, Result: uploadURL.String()}
	}
	res.JSONf(w)
}

func PlayerTracker(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	playerKey := utils.Pkey(r)
	clanKey := utils.Var(r, "clankey")
	var res utils.JSONResult
	tracker, err := Tracker(c, playerKey, clanKey)
	if err != nil {
		res = utils.JSONResult{Success: false, Error: err.Error()}
	}
	res = utils.JSONResult{Success: true, Result: tracker}
	res.JSONf(w)
}

func LocalEvents(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	playerKey := utils.Pkey(r)
	cursor := utils.Var(r, "cursor")
	var res utils.JSONResult
	events, err := Events(c, playerKey, "Player", cursor)
	if err != nil {
		res = utils.JSONResult{Success: false, Error: err.Error()}
	}
	res = utils.JSONResult{Success: true, Result: events}
	res.JSONf(w)
}

func GlobalEvents(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	playerKey := utils.Pkey(r)
	cursor := utils.Var(r, "cursor")
	var res utils.JSONResult
	events, err := Events(c, playerKey, "Clan", cursor)
	if err != nil {
		res = utils.JSONResult{Success: false, Error: err.Error()}
	}
	res = utils.JSONResult{Success: true, Result: events}
	res.JSONf(w)
}
