package router

import (
	"github.com/gorilla/mux"
	"mj0lk.be/netwars/attack"
	"mj0lk.be/netwars/clan"
	"mj0lk.be/netwars/message"
	"mj0lk.be/netwars/player"
	"mj0lk.be/netwars/program"
	"net/http"
)

func Setup() {
	r := mux.NewRouter().StrictSlash(true)
	ps := r.PathPrefix("/players").Subrouter()
	ps.HandleFunc("/", player.CreatePlayer).Methods("POST")
	ps.HandleFunc("/status", player.StatusPlayer).Methods("GET").Headers("Accept", "application/json").Queries("pkey", "")
	ps.HandleFunc("/allocation", player.AllocatePrograms).Methods("POST")
	ps.HandleFunc("/deallocation", player.DeallocatePrograms).Methods("POST")
	ps.HandleFunc("/", player.GetPlayerList).Methods("GET").Queries("pkey", "")
	ps.HandleFunc("/avatar", player.UploadAvatar).Methods("GET")
	ps.HandleFunc("/avatar", player.EditAvatar).Methods("POST")
	ps.HandleFunc("/profile", player.EditProfile).Methods("POST")
	ps.HandleFunc("/trackers", player.PlayerTracker).Methods("GET")
	ps.HandleFunc("/events", player.EventList).Methods("GET")

	cs := r.PathPrefix("/clans").Subrouter()
	cs.HandleFunc("/", clan.CreateClan).Methods("POST")
	cs.HandleFunc("/", clan.ClanStatus).Methods("GET")
	cs.HandleFunc("/invitations", clan.ClanInvite).Methods("POST")
	cs.HandleFunc("/invitations", clan.Invites).Methods("GET")
	cs.HandleFunc("/links", clan.JoinClan).Methods("POST")
	cs.HandleFunc("/unlinks", clan.LeaveClan).Methods("POST")
	cs.HandleFunc("/connections", clan.ClanConnect).Methods("POST")
	cs.HandleFunc("/disconnections", clan.ClanDisConnect).Methods("POST")
	cs.HandleFunc("/avatar", clan.EditAvatar).Methods("GET")
	cs.HandleFunc("/avatar", clan.UploadAvatar).Methods("POST")
	cs.HandleFunc("/messages", clan.EditMessage).Methods("POST")
	cs.HandleFunc("/promotions", clan.EditLeaderShip).Methods("POST")
	cs.HandleFunc("/demotions", clan.EditLeaderShip).Methods("POST")
	cs.HandleFunc("/removals", clan.KickPlayer).Methods("POST")

	r.HandleFunc("/attack", attack.AttackPlayer).Methods("POST")

	ms := r.PathPrefix("/messages").Subrouter()
	ms.HandleFunc("/", message.CreateOrUpdateMessage).Methods("POST")
	//s.HandleFunc("/message_delete", DeleteMessage)
	ms.HandleFunc("/boards/clan", message.ListClanBoards).Methods("GET").Queries("pkey", "")
	ms.HandleFunc("/boards/public", message.ListPublicBoards).Methods("GET").Queries("pkey", "")
	ms.HandleFunc("/threads", message.ListThreads).Methods("GET").Queries("pkey", "", "tkey", "")
	ms.HandleFunc("/messages", message.ListMessages).Methods("GET").Queries("pkey", "", "tkey", "")

	prs := r.PathPrefix("/programs").Subrouter()
	prs.HandleFunc("/", program.GetAllPrograms).Methods("GET")
	prs.HandleFunc("/", program.CreateOrUpdateProgram).Methods("POST")
	prs.HandleFunc("/", program.GetProgram).Methods("GET").Queries("pkey", "")
	r.HandleFunc("/load/programs", program.LoadPrograms)

	http.Handle("/", r)
}
