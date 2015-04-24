package testutils

import (
	"appengine"
	"appengine/aetest"
	"appengine/datastore"
	"errors"
	"mj0lk.be/netwars/clan"
	"mj0lk.be/netwars/event"
	"mj0lk.be/netwars/player"
	"mj0lk.be/netwars/secure"
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

func setupPlayer(c appengine.Context, nick string, email string) (string, error) {
	cr := player.Creation{email, nick, "testpassword"}
	tokenStr, usererr, err := player.Create(c, cr)
	if err != nil {
		return "", err

	}
	if usererr != nil {
		return "", errors.New("unexpected user error")
	}
	playerKeyStr, _ := secure.ValidateToken(tokenStr)
	playerKey, _ := datastore.DecodeKey(playerKeyStr)
	notif := event.PlayerNotification{
		EventType:        "Invite",
		NotificationType: "Email",
		Player:           playerKey,
	}
	notifKey := datastore.NewKey(c, "PlayerNotification", "", 1, nil)
	if _, err := datastore.Put(c, notifKey, &notif); err != nil {
		return "", err
	}
	return playerKeyStr, nil
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
	tracker := new(event.Tracker)
	trackerKey := datastore.NewKey(c, "Tracker", playerKey2.StringID(), 0, clanKey)
	if _, err := datastore.PutMulti(c, []*datastore.Key{playerKey2, trackerKey},
		[]interface{}{pl, tracker}); err != nil {
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
