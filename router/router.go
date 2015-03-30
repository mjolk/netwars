package router

import (
	"github.com/gorilla/mux"
	"mj0lk.be/netwars/attack"
	"mj0lk.be/netwars/clan"
	"mj0lk.be/netwars/event"
	"mj0lk.be/netwars/message"
	"mj0lk.be/netwars/player"
	"mj0lk.be/netwars/program"
	"mj0lk.be/netwars/utils"
	"net/http"
)

type Route struct {
	Name     string
	Path     []string
	Method   string
	Headers  []string
	Handler  http.HandlerFunc
	Request  interface{}
	Response interface{}
	Auth     bool
}

type Routes []Route

var jsonheader = []string{"Accept", "application/json; charset=UTF-8"}

var routes = map[string]Routes{
	"/players": Routes{
		Route{
			"login",
			[]string{"/login"},
			"POST",
			jsonheader,
			player.AuthenticatePlayer,
			player.Authentication{"email", "password"},
			utils.JSONResult{Result: "token"},
			false,
		},
		Route{
			"createplayer",
			[]string{"/"},
			"POST",
			jsonheader,
			player.CreatePlayer,
			player.Creation{"blabla@mail.com", "nickname", "password"},
			utils.JSONResult{Result: "token"},
			false,
		},
		Route{
			"playerstatus",
			[]string{"/status"},
			"GET",
			jsonheader,
			player.StatusPlayer,
			nil,
			utils.JSONResult{Result: player.Player{}},
			true,
		},
		Route{
			"allocate",
			[]string{"/allocation"},
			"POST",
			jsonheader,
			player.AllocatePrograms,
			player.Allocation{"program key", 10},
			http.StatusOK,
			true,
		},
		Route{
			"deallocate",
			[]string{"/deallocation"},
			"POST",
			jsonheader,
			player.DeallocatePrograms,
			player.Allocation{"program key", 10},
			http.StatusOK,
			true,
		},
		Route{
			"playerlist",
			[]string{"/", "/{range}", "/{cursor}", "/{range}/{cursor}"},
			"GET",
			jsonheader,
			player.GetPlayerList,
			nil,
			utils.JSONResult{Result: player.PlayerList{Cursor: "send to get next page",
				Players: []player.Profile{player.Profile{}}}},
			true,
		},
		Route{
			"playeruploadurl",
			[]string{"/avatar"},
			"GET",
			jsonheader,
			player.UploadAvatar,
			nil,
			utils.JSONResult{Result: "upload url"},
			true,
		},
		Route{
			"playerupload",
			[]string{"/avatar"},
			"POST",
			jsonheader,
			player.EditAvatar,
			"file with name: avatar",
			http.StatusOK,
			true,
		},
		Route{
			"updateprofile",
			[]string{"/profile"},
			"POST",
			jsonheader,
			player.EditProfile,
			player.ProfileUpdate{
				Name:      "name",
				Birthday:  player.TIMELAYOUT,
				Country:   "country",
				Language:  "English",
				Address:   "address",
				Signature: "signature",
			},
			http.StatusOK,
			true,
		},
		Route{
			"playertrackers",
			[]string{"/trackers", "/trackers/{clankey}"},
			"GET",
			jsonheader,
			player.PlayerTracker,
			nil,
			utils.JSONResult{Result: event.Tracker{0, 0}},
			true,
		},
		Route{
			"playerevents",
			[]string{"/events/{loc:clan|player}", "/events/{loc:clan|player}/{cursor}"},
			"GET",
			jsonheader,
			player.EventList,
			nil,
			utils.JSONResult{Result: event.EventList{Cursor: "send to get next page",
				Events: []event.Event{event.Event{}}}},
			true,
		},
	},
	"/clans": Routes{
		Route{
			"createclan",
			[]string{"/"},
			"POST",
			jsonheader,
			clan.CreateClan,
			clan.Creation{"tag", "name"},
			http.StatusOK,
			true,
		},
		Route{
			"clanstatus",
			[]string{"/status"},
			"GET",
			jsonheader,
			clan.ClanStatus,
			nil,
			utils.JSONResult{Result: clan.Clan{}},
			true,
		},
		Route{ //TODO add public clan lookup, route does nothing for now
			"clanstatuspublic",
			[]string{"/status/{clankey}"},
			"GET",
			jsonheader,
			clan.ClanStatus,
			nil,
			utils.JSONResult{Result: clan.Clan{}},
			true,
		},
		Route{
			"invite",
			[]string{"/invitations"},
			"POST",
			jsonheader,
			clan.ClanInvite,
			clan.Pmanipulation{},
			http.StatusOK,
			true,
		},
		Route{
			"invites",
			[]string{"/invitations"},
			"GET",
			jsonheader,
			clan.Invites,
			nil,
			utils.JSONResult{Result: []clan.Invite{clan.Invite{}}},
			true,
		},
		Route{
			"joinclan",
			[]string{"/links"},
			"POST",
			jsonheader,
			clan.JoinClan,
			clan.SendKey{Key: "invite key"},
			http.StatusOK,
			true,
		},
		Route{
			"leaveclan",
			[]string{"/unlinks"},
			"POST",
			jsonheader,
			clan.LeaveClan,
			nil,
			http.StatusOK,
			true,
		},
		Route{
			"war",
			[]string{"/connections"},
			"POST",
			jsonheader,
			clan.ClanConnect,
			clan.SendKey{Key: "clan key"},
			http.StatusOK,
			true,
		},
		Route{
			"peace",
			[]string{"/disconnections"},
			"POST",
			jsonheader,
			clan.ClanDisConnect,
			clan.SendKey{Key: "connection key"},
			http.StatusOK,
			true,
		},
		Route{
			"clanuploadurl",
			[]string{"/avatar"},
			"GET",
			jsonheader,
			clan.UploadAvatar,
			nil,
			utils.JSONResult{Result: "upload url"},
			true,
		},
		Route{
			"clanupload",
			[]string{"/avatar"},
			"POST",
			jsonheader,
			clan.EditAvatar,
			"file with name: avatar",
			http.StatusOK,
			true,
		},
		Route{
			"clanmessage",
			[]string{"/messages"},
			"POST",
			jsonheader,
			clan.EditMessage,
			clan.MessageUpdate{},
			http.StatusOK,
			true,
		},
		Route{
			"promote",
			[]string{"/promotions"},
			"POST",
			jsonheader,
			clan.EditLeaderShip,
			clan.Promotion{},
			http.StatusOK,
			true,
		},
		Route{
			"demote",
			[]string{"/demotions"},
			"POST",
			jsonheader,
			clan.EditLeaderShip,
			clan.Promotion{},
			http.StatusOK,
			true,
		},
		Route{
			"kick",
			[]string{"/removals"},
			"POST",
			jsonheader,
			clan.KickPlayer,
			clan.Pmanipulation{},
			http.StatusOK,
			true,
		},
	},
	"/attacks": Routes{
		Route{
			"attack",
			[]string{"/"},
			"POST",
			jsonheader,
			attack.AttackPlayer,
			attack.AttackCfg{},
			utils.JSONResult{Result: attack.AttackEvent{}},
			true,
		},
	},
	"/messages": Routes{
		Route{
			"createorupdate",
			[]string{"/"},
			"POST",
			jsonheader,
			message.CreateOrUpdateMessage,
			message.Message{},
			http.StatusOK,
			true,
		},
		Route{
			"clanforum",
			[]string{"/boards/clan", "/boards/clan/{cursor}"},
			"GET",
			jsonheader,
			message.ListClanBoards,
			nil,
			utils.JSONResult{Result: message.MessageList{Cursor: "paging",
				Messages: []message.Message{message.Message{}}, BoardKey: "board key"}},
			true,
		},
		Route{
			"publicboards",
			[]string{"/boards/public", "/boards/public/{cursor}"},
			"GET",
			jsonheader,
			message.ListPublicBoards,
			nil,
			utils.JSONResult{Result: message.MessageList{Cursor: "paging",
				Messages: []message.Message{message.Message{}}, BoardKey: "board key"}},
			true,
		},
		Route{
			"threads",
			[]string{"/threads/{bkey}", "/threads/{bkey}/{cursor}"},
			"GET",
			jsonheader,
			message.ListThreads,
			nil,
			utils.JSONResult{Result: message.MessageList{Cursor: "paging",
				Messages: []message.Message{message.Message{}}, BoardKey: "board key",
				ThreadKey: "thread key"}},
			true,
		},
		Route{
			"messages",
			[]string{"/messages/{tkey}", "/messages/{tkey}/{cursor}"},
			"GET",
			jsonheader,
			message.ListMessages,
			nil,
			utils.JSONResult{Result: message.MessageList{Cursor: "paging",
				Messages: []message.Message{message.Message{}}, BoardKey: "board key",
				ThreadKey: "thread key"}},
			true,
		},
	},
	"/programs": Routes{
		Route{
			"getallprograms",
			[]string{"/"},
			"GET",
			jsonheader,
			program.GetAllPrograms,
			nil,
			utils.JSONResult{Result: map[string][]program.Program{"type": []program.Program{program.Program{}}}},
			true,
		},
		Route{
			"createorupdate",
			[]string{"/"},
			"POST",
			jsonheader,
			program.CreateOrUpdateProgram,
			program.Program{},
			http.StatusOK,
			true,
		},
		Route{
			"getprogram",
			[]string{"/{key}"},
			"GET",
			jsonheader,
			program.GetProgram,
			nil,
			program.Program{},
			true,
		},
		Route{
			"loadjsonprograms",
			[]string{"/load"},
			"POST",
			jsonheader,
			program.LoadPrograms,
			[]program.Program{program.Program{}},
			http.StatusOK,
			false,
		},
	},
}

func Setup() {
	r := mux.NewRouter().StrictSlash(true)

	for prefix, routes := range routes {
		//check multiple paths (optional variables)
		subRouter := r.PathPrefix(prefix).Subrouter()
		for _, route := range routes {
			for _, path := range route.Path {
				handler := route.Handler
				if route.Auth {
					handler = utils.Validator(route.Handler)
				}
				subRouter.HandleFunc(path, handler).
					Methods(route.Method).
					Headers(route.Headers...).
					Name(route.Name)
			}
		}
	}

	http.Handle("/", r)
}
