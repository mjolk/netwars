package player

import (
	"appengine/blobstore"
	"mj0lk.be/netwars/app"
	"net/http"
)

func EditProfile(w http.ResponseWriter, r *http.Request, c app.Context) {
	update := ProfileUpdate{}
	var res app.JSONResult
	if err := app.DecodeJsonBody(r, &update); err != nil {
		res = app.JSONResult{Success: false, StatusCode: 422, Error: err.Error()}
		res.JSONf(w)
		return
	}
	if err := UpdateProfile(c, c.User, update); err != nil {
		res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
		res.JSONf(w)
	}
}

func CreatePlayer(w http.ResponseWriter, r *http.Request, c app.Context) {
	cr := Creation{}
	var res app.JSONResult
	if err := app.DecodeJsonBody(r, &cr); err != nil {
		res = app.JSONResult{Success: false, StatusCode: 422, Error: err.Error()}
		res.JSONf(w)
		return
	}
	enckey, errmap, err := Create(c, cr)
	if err != nil {
		res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
	} else if len(errmap) > 0 {
		if errmap["email"] > 0 {
			res = app.JSONResult{Success: false, StatusCode: http.StatusConflict, Error: "error mail"}
		} else {
			res = app.JSONResult{Success: false, StatusCode: http.StatusConflict, Error: "error nick"}
		}
	} else {
		res = app.JSONResult{Success: true, StatusCode: http.StatusOK, Result: enckey}

	}
	res.JSONf(w)
}

func GetPlayerList(w http.ResponseWriter, r *http.Request, c app.Context) {
	list, err := List(c, c.User, c.Param("range_bool"), c.Param("cursor_key"))
	var res app.JSONResult
	if err != nil {
		res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
	} else {
		res = app.JSONResult{Success: true, StatusCode: http.StatusOK, Result: list}
	}
	res.JSONf(w)
}

func StatusPlayer(w http.ResponseWriter, r *http.Request, c app.Context) {
	iplayer := new(Player)
	var res app.JSONResult
	if err := Tstatus(c, c.User, iplayer); err != nil {
		res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
	} else {
		res = app.JSONResult{Success: true, StatusCode: http.StatusOK, Result: iplayer}
	}
	res.JSONf(w)
}

func PublicStatusPlayer(w http.ResponseWriter, r *http.Request, c app.Context) {
	iplayer := new(PublicPlayer)
	err := Public(c, c.User, c.Param("player_id"), iplayer)
	var res app.JSONResult
	if err != nil {
		res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
	} else {
		res = app.JSONResult{Success: true, StatusCode: http.StatusOK, Result: iplayer}
	}
	res.JSONf(w)
}

func AllocatePrograms(w http.ResponseWriter, r *http.Request, c app.Context) {
	al := Allocation{}
	var res app.JSONResult
	if err := app.DecodeJsonBody(r, &al); err != nil {
		res = app.JSONResult{Success: false, StatusCode: 422, Error: err.Error()}
		res.JSONf(w)
		return
	}
	if err := Allocate(c, c.User, al); err != nil {
		c.Debugf("error allocating %s \n", err)
		res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
		res.JSONf(w)
	}
}

func AuthenticatePlayer(w http.ResponseWriter, r *http.Request, c app.Context) {
	al := Authentication{}
	var res app.JSONResult
	if err := app.DecodeJsonBody(r, &al); err != nil {
		res = app.JSONResult{Success: false, StatusCode: 422, Error: err.Error()}
		res.JSONf(w)
		return
	}
	if token, err := Login(c, al); err != nil {
		c.Debugf("error login %s \n", err)
		res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
	} else {
		res = app.JSONResult{Success: true, Result: token, StatusCode: http.StatusOK}
	}
	res.JSONf(w)
}

func DeallocatePrograms(w http.ResponseWriter, r *http.Request, c app.Context) {
	var res app.JSONResult
	al := Allocation{}
	if err := app.DecodeJsonBody(r, &al); err != nil {
		res = app.JSONResult{Success: false, StatusCode: 422, Error: err.Error()}
		res.JSONf(w)
		return
	}
	if err := Deallocate(c, c.User, al); err != nil {
		res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
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
	uploadURL, err := blobstore.UploadURL(c, "/players/avatar", nil)
	var res app.JSONResult
	if err != nil {
		res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
	} else {
		res = app.JSONResult{Success: true, StatusCode: http.StatusOK, Result: uploadURL.String()}
	}
	res.JSONf(w)
}

func PlayerTracker(w http.ResponseWriter, r *http.Request, c app.Context) {
	var res app.JSONResult
	tracker, err := Tracker(c, c.User, c.Param("clan_key"))
	if err != nil {
		res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
	}
	res = app.JSONResult{Success: true, StatusCode: http.StatusOK, Result: tracker}
	res.JSONf(w)
}

func LocalEvents(w http.ResponseWriter, r *http.Request, c app.Context) {
	var res app.JSONResult
	events, err := Events(c, c.User, "Player", c.Param("cursor_key"))
	if err != nil {
		res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
	}
	res = app.JSONResult{Success: true, StatusCode: http.StatusOK, Result: events}
	res.JSONf(w)
}

func GlobalEvents(w http.ResponseWriter, r *http.Request, c app.Context) {
	var res app.JSONResult
	events, err := Events(c, c.User, "Clan", c.Param("cursor_key"))
	if err != nil {
		res = app.JSONResult{Success: false, StatusCode: http.StatusInternalServerError, Error: err.Error()}
	}
	res = app.JSONResult{Success: true, StatusCode: http.StatusOK, Result: events}
	res.JSONf(w)
}
