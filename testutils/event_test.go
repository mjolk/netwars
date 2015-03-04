package testutils

import (
	"appengine"
	"appengine/aetest"
	"appengine/datastore"
	"mj0lk.be/netwars/clan"
	"mj0lk.be/netwars/event"
	"mj0lk.be/netwars/player"
	"testing"
)

const (
	CLAN1      = "CLANA"
	CLAN2      = "CLANB"
	TESTNICK1  = "testnick"
	TESTEMAIL1 = "testemail@mail.com"
	TESTNICK2  = "testnickb"
	TESTEMAIL2 = "testemail_1@mail.com"
)

func setupPlayer(c appengine.Context, nick, email string) (string, error) {
	playerStr, _, err := player.Create(c, nick, email)
	if err != nil {
		return "", err
	}
	playerKey, _ := datastore.DecodeKey(playerStr)
	notif := event.PlayerNotification{
		EventType:        "Invite",
		NotificationType: "Email",
		Player:           playerKey,
	}
	notifKey := datastore.NewKey(c, "PlayerNotification", "", 1, nil)
	if _, err := datastore.Put(c, notifKey, &notif); err != nil {
		return "", err
	}
	return playerStr, nil
}

func TestNotify(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	playerStr, err := setupPlayer(c, TESTNICK1, TESTEMAIL1)
	if err != nil {
		t.Fatalf("\n error setup owner %s", err)
	}
	clanGuid, _, err := clan.Create(c, playerStr, CLAN1, "lol")
	if err != nil {
		t.Fatalf("\n create clan error %s", err)
	}
	playerKey, _ := datastore.DecodeKey(playerStr)
	clanKey := datastore.NewKey(c, "Clan", clanGuid, 0, nil)
	ev := event.Event{
		Player:    playerKey,
		Clan:      clanKey,
		EventType: "Clan",
		Action:    "Invite",
	}
	ch := make(chan int, 1)
	ev.Notify(c, ch)
}

func TestNotify2(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	playerStr, err := setupPlayer(c, TESTNICK1, TESTEMAIL1)
	if err != nil {
		t.Fatalf("\n error setup owner %s", err)
	}
	clanGuid, _, err := clan.Create(c, playerStr, CLAN1, "lol")
	if err != nil {
		t.Fatalf("\n create clan error %s", err)
	}

	playerStr2, err := setupPlayer(c, TESTNICK2, TESTEMAIL2)
	if err != nil {
		t.Fatalf("\n error setup owner %s", err)
	}
	clanKey := datastore.NewKey(c, "Clan", clanGuid, 0, nil)
	pl := new(player.Player)
	playerKey2, _ := datastore.DecodeKey(playerStr2)
	if err := datastore.Get(c, playerKey2, pl); err != nil {
		t.Fatalf("\n error getting player %s", err)
	}
	pl.ClanKey = clanKey
	pl.Clan = CLAN1
	pl.ClanTag = "lol"
	if _, err := datastore.Put(c, playerKey2, pl); err != nil {
		t.Fatalf("\n error saving player %s", err)
	}
	playerKey, _ := datastore.DecodeKey(playerStr)
	ev := event.Event{
		Player:    playerKey,
		Clan:      clanKey,
		EventType: "Clan",
		Action:    "Invite",
	}
	ch := make(chan int, 1)
	ev.Notify(c, ch)
}
