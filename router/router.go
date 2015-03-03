package router

import (
	"github.com/gorilla/mux"
	"mj0lk.be/netwars/clan"
	"mj0lk.be/netwars/player"
)

func init() {
	router := mux.NewRouter()
	s := r.PathPrefix("/players").Subrouter()
	s.HandleFunc("/", player.CreatePlayer).Methods("POST")
	s.HandleFunc("/status", player.StatusPlayer).Methods("GET").Headers("Accept", "application/json").Queries("pkey", "")
	s.HandleFunc("/allocation", player.AllocatePrograms).Methods("POST")
	s.HandleFunc("/deallocation", player.DeallocatePrograms).Methods("POST")
	s.HandleFunc("/", player.GetPlayerList).Methods("GET").Queries("pkey", "")
	s.HandleFunc("/avatar", player.UploadAvatar).Methods("GET")
	s.HandleFunc("/avatar", player.EditAvatar).Methods("POST")
	s.HandleFunc("/profile", player.EditProfile).Methods("POST")

	s := r.PathPrefix("/clans").Subrouter()
	s.HandleFunc("/", clan.CreateClan).Methods("POST")
	s.HandleFunc("/", clan.ClanStatus).Methods("GET")
	s.HandleFunc("/invitations", clan.ClanInvite).Methods("POST")
	s.HandleFunc("/invitations", clan.Invites).Methods("GET")
	s.HandleFunc("/links", clan.JoinClan).Methods("POST")
	s.HandleFunc("/unlinks", clan.LeaveClan).Methods("POST")
	s.HandleFunc("/connections", clan.ClanConnect).Methods("POST")
	s.HandleFunc("/disconnections", clan.ClanDisConnect).Methods("POST")
	s.HandleFunc("/avatar", clan.EditAvatar).Methods("GET")
	s.HandleFunc("/avatar", clan.UploadAvatar).Methods("POST")
	s.HandleFunc("/messages", clan.EditMessage).Methods("POST")
	s.HandleFunc("/promotions", clan.EditLeaderShip).Methods("POST")
	s.HandleFunc("/demotions", clan.EditLeaderShip).Methods("POST")
	s.HandleFunc("/removals", clan.KickPlayer).Methods("POST")
}
