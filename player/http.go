package player

import (
	"appengine"
	"appengine/blobstore"
	"encoding/json"
	"fmt"
	"net/http"
	"netwars/utils"
)

func init() {
	r := utils.Router()
	s := r.PathPrefix("/players").Subrouter()
	s.HandleFunc("/", CreatePlayer).Methods("POST")
	s.HandleFunc("/status", StatusPlayer).Methods("GET").Headers("Accept", "application/json").Queries("pkey", "")
	s.HandleFunc("/allocation", AllocatePrograms).Methods("POST")
	s.HandleFunc("/deallocation", DeallocatePrograms).Methods("POST")
	s.HandleFunc("/", GetPlayerList).Methods("GET").Queries("pkey", "")
	s.HandleFunc("/avatar", UploadAvatar).Methods("GET")
	s.HandleFunc("/avatar", EditAvatar).Methods("POST")
	s.HandleFunc("/profile", EditProfile).Methods("POST")
	//dev to be moved to admin section
}

// EditProfile edits a player's profile
// POST http(s)://{netwars host}/player_editprofile
//
// parameters:
// ProfileUpdate
// json:
// {
//   pkey: player key (string)
//   name: player name (string)
//   birthday:"2006-Jan-02" (string)
//   country: country (string)
//   language: "nl/eng/..." (string)
//   address: address   (string)
//   signature: signature (string)
// }
//
// result:
// http 200 ok OR
// json:
// {
//   success: false
//   error: error (string)
// }
func EditProfile(w http.ResponseWriter, r *http.Request) {
	update := ProfileUpdate{}
	c := appengine.NewContext(r)
	json.NewDecoder(r.Body).Decode(update)
	if err := UpdateProfile(c, update); err != nil {
		res := utils.JSONResult{Success: false, Error: err.Error()}
		res.JSONf(w)
	}
}

// CreatePlayer creates new player
// POST http://{netwars host}/player_create
//
// parameters:
// nick="nick"
// email="user@email.com"
//
// result json:
// {
//  success: true/false (boolean),
//  error: error (string),
//  result: new player key (string),
// }
//
// if nick or email already exist :
// Result: [{"email": 1/0},{"nick": 1/0}]
// in case the value is 1 it means the email and/or nick already exist in the system.
func CreatePlayer(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	nick := r.FormValue("nick")
	email := r.FormValue("email")
	enckey, errmap, err := Create(c, nick, email)
	var res utils.JSONResult
	c.Debugf("%s", errmap)
	if err != nil {
		res = utils.JSONResult{Success: false, Error: err.Error()}
	} else if len(errmap) > 0 {
		res = utils.JSONResult{Success: false, Error: fmt.Sprint("errmap: %+v", errmap)}
	} else {
		res = utils.JSONResult{Success: true, Result: enckey}

	}
	res.JSONf(w)
}

// GetPlayerList returns a list of players
// GET http://{netwars host}/player_list?pkey={player key}&c={next result set key}
//
// parameters:
// pkey="unique player key" player key of the requestor
// c="optional cursor string acting as a pointer in the list"
// a cursor key is returned on the first request, sending it on the next request will request the next page of results
//
// result json:
// {
//   success: true,
//   error: error (string),
//   result: {
//                c: "cursor",
//                players: [
//                  {
//                    created: "timestamp" created,
//                    nick: "nick name",
//                    avatar_thumb: "url",
//                    player_id: player id,
//                    bandwidth_usage: bandwidth usage,
//                    status: "status",
//                    type: "player type",
//                    clan: "clan name",
//                    clan_tag: "clan tag",
//                    key: "player (profile) key",
//                   },
//                  ...
//                ]
//             }
// }
func GetPlayerList(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	cur := r.FormValue("c")
	playerKey := r.FormValue("pkey")
	attackRange := r.FormValue("range")
	list, err := List(c, playerKey, attackRange, cur)
	var res utils.JSONResult
	if err != nil {
		res = utils.JSONResult{Success: false, Error: err.Error()}
	} else {
		res = utils.JSONResult{Success: true, Result: list}
	}
	res.JSONf(w)
}

// StatusPlayer return the status of a player
// GET http://{netwars host}/player_status?pkey={player key string}
//
// parameters :
// pkey="unique player key"
//
// result json:
// {
//   error: "error",
//   success: true/false,
//   result: {
//                success: true/false,
//                error: "error string",
//                result: {
//                          player: {}
//                          programs: [
//                                      {
//                                          "programtype":
//                                                          {
//                                                              available_bw: available bandwidth for type,
//                                                              used_bw: bandwidth used for type,
//                                                              power: true/false power status for type,
//                                                              programs: [
//                                                                          {
//                                                                              amount: amount programs,
//                                                                              key: player program key,
//                                                                              expires: expiration (infect),
//                                                                              active: true/false,
//                                                                              program_key: program key,
//                                                                              name: program name,
//                                                                          },
//                                                                          ...
//                                                                           ]
//                                                          }
//                                      },
//                                      ...
//                                  ]
//                      }
//              }
func StatusPlayer(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	playerstr := r.FormValue("pkey")
	state := &PlayerState{}
	var res utils.JSONResult
	if _, err := Status(c, playerstr, state); err != nil {
		res = utils.JSONResult{Success: false, Error: err.Error()}
	} else {
		res = utils.JSONResult{Success: true, Result: state}
	}
	res.JSONf(w)
}

// AllocatePrograms allocates a program to a player
// POST http://{netwars host}/player_allocate
//
// parameters :
// pkey="unique player key"
// prgkey="key of the program to add"
// amount="amount of programs of type prgkey to add" (string)
//
// result: http 200 OK or
// json: {
//          error: "error string",
//          success: false
//      }
func AllocatePrograms(w http.ResponseWriter, r *http.Request) {
	player := r.FormValue("pkey")
	program := r.FormValue("prgkey")
	amount := r.FormValue("amount")
	c := appengine.NewContext(r)
	c.Debugf("input variables PLAYER: %s, PROGRAM: %s, AMOUNT: %s", player, program, amount)
	if err := Allocate(c, player, program, amount); err != nil {
		c.Debugf("error allocating %s \n", err)
		res := utils.JSONResult{Success: false, Error: err.Error()}
		res.JSONf(w)
	}
}

// DeallocatePrograms deallocates programs
// POST http://{netwars host}/player_deallocate
//
// parameters:
// pkey="unique player key"
// prgkey="program to remove"
// amount="amount of type prgkey to remove" (string)
//
// result: http 200 OK or
// json: {
//          error: "error string",
//          success: false
//      }
func DeallocatePrograms(w http.ResponseWriter, r *http.Request) {
	player := r.FormValue("pkey")
	program := r.FormValue("prgkey")
	amount := r.FormValue("amount")
	c := appengine.NewContext(r)
	if err := Deallocate(c, player, program, amount); err != nil {
		res := utils.JSONResult{Success: false, Error: err.Error()}
		res.JSONf(w)
	}
}

// UploadAvatar upload image as profile picture
// POST multipart form url generated by EditAvatar : /player_uploadavatar
//
// parameters:
// multipart form where =>
// input field with id pkey = unique player key
// input file with id avatar = image file
//
// result : http 200 OK or
//
// json: {
//          error: "error string",
//          success: false
//      }
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

// EditAvatar requests an upload url for a new profile image
// GET http://{netwars host}/player_editavatar
//
// parameters: none
//
// result:
// json: {
//          error: "error string",
//          success: false/true,
//          result: upload url,
//      }
func EditAvatar(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	uploadURL, err := blobstore.UploadURL(c, "/player_uploadavatar", nil)
	var res utils.JSONResult
	if err != nil {
		res = utils.JSONResult{Success: false, Error: err.Error()}
	} else {
		res = utils.JSONResult{Success: true, Result: uploadURL.String()}
	}
	res.JSONf(w)
}
