package clan

import (
	"appengine"
	"appengine/aetest"
	"appengine/datastore"
	"errors"
	"fmt"
	"mj0lk.be/netwars/player"
	"mj0lk.be/netwars/secure"
	"mj0lk.be/netwars/testutils"
	"testing"
	"time"
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
	return playerKeyStr, nil
}

func TestValidClan(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	playerStr, err := setupPlayer(c, TESTNICK1, TESTEMAIL1)
	if err != nil {
		t.Fatalf("\n error setup owner %s", err)
	}
	_, errmap, err := Create(c, playerStr, CLAN1, "lol")
	if err != nil {
		t.Fatalf("\n create clan error %s", err)
	}
	if errmap["clan_name"]+errmap["clan_tag"] > 0 {
		t.Fatalf(" bad clan name or clan tag \n")
	}

	_, errmap2, err := Create(c, playerStr, CLAN1, "lol")
	if errmap2["clan_name"]+errmap2["clan_tag"] == 0 {
		t.Fatalf(" error bad clan name or clan tag \n")
	}
	t.Logf("errmap %+v \n", errmap2)
}

func TestCreate(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	playerStr, err := setupPlayer(c, TESTNICK1, TESTEMAIL1)
	if err != nil {
		t.Fatalf("\n error setup owner %s", err)
	}
	_, errmap, err := Create(c, playerStr, CLAN1, "lol")
	if err != nil {
		t.Fatalf("\n create clan error %s", err)
	}
	if errmap["clan_name"]+errmap["clan_tag"] > 0 {
		t.Fatalf(" bad clan name or clan tag \n")
	}
	playerKey, _ := datastore.DecodeKey(playerStr)
	player := new(player.Player)
	if err := datastore.Get(c, playerKey, player); err != nil {
		t.Fatalf("\n error loading player %s", err)
	}
	if player.ClanKey == nil {
		t.Fatalf("\n clankey is nil, something went wrong")
	}
	t.Logf("player : %+v", player)
	testutils.CheckQueue(c, t, 1)
}

func TestInvite(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	inviterStr, err := setupPlayer(c, TESTNICK1, TESTEMAIL1)
	if err != nil {
		t.Fatalf("Error setting up inviter")
	}
	_, _, err = Create(c, inviterStr, CLAN1, "lol")
	if err != nil {
		t.Fatalf("\nError creating clan %s", err)
	}
	inviteeStr, err := setupPlayer(c, TESTNICK2, TESTEMAIL2)
	if err != nil {
		t.Fatalf("\nError setting up invitee", err)
	}
	inviteePlayerKey, err := datastore.DecodeKey(inviteeStr)
	if err != nil {
		t.Fatalf("\nError decoding key", err)
	}
	invitedPlayer := new(player.Player)
	if err := datastore.Get(c, inviteePlayerKey, invitedPlayer); err != nil {
		t.Fatalf("\nError getting invited player", err)
	}
	testutils.PurgeQueue(c, t)
	if err := InvitePlayer(c, inviterStr, invitedPlayer.ID); err != nil {
		t.Fatalf("\nError sending invite %s", err)
	}
	testutils.CheckQueue(c, t, 1)
	testutils.PurgeQueue(c, t)
	if err := InvitePlayer(c, inviterStr, invitedPlayer.ID); err != nil {
		if err != PlayerAlreadyInvitedError {
			t.Fatalf("Error checking player exists")
		}
	}
	testutils.CheckQueue(c, t, 0)
}

func TestCancelInvite(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	inviterStr, err := setupPlayer(c, TESTNICK1, TESTEMAIL1)
	if err != nil {
		t.Fatalf("Error setting up inviter")
	}
	_, _, err = Create(c, inviterStr, CLAN1, "lol")
	if err != nil {
		t.Fatalf("\nError creating clan %s", err)
	}
	inviteeStr, err := setupPlayer(c, TESTNICK2, TESTEMAIL2)
	if err != nil {
		t.Fatalf("\nError setting up invitee", err)
	}
	inviteePlayerKey, err := datastore.DecodeKey(inviteeStr)
	if err != nil {
		t.Fatalf("\nError decoding key", err)
	}
	invitedPlayer := new(player.Player)
	if err := datastore.Get(c, inviteePlayerKey, invitedPlayer); err != nil {
		t.Fatalf("\nError getting invited player", err)
	}
	testutils.PurgeQueue(c, t)
	if err := InvitePlayer(c, inviterStr, invitedPlayer.ID); err != nil {
		t.Fatalf("\nError sending invite %s", err)
	}
	testutils.CheckQueue(c, t, 1)
	testutils.PurgeQueue(c, t)
	time.Sleep(1 * time.Second)
	invites := make([]Invite, 0)
	q := datastore.NewQuery("Invite").Filter("Player =", inviteePlayerKey).Limit(1)
	for it := q.Run(c); ; {
		var invite Invite
		key, err := it.Next(&invite)
		if err == datastore.Done {
			break
		}
		if err != nil {
			t.Fatalf("error loading invite")
		}
		invite.DbKey = key
		invite.EncodedKey = key.Encode()
		invites = append(invites, invite)
	}
	invite := invites[0]
	if err := CancelInvite(c, inviterStr, invite.EncodedKey); err != nil {
		t.Fatalf("Error cancelling invite")
	}
	testutils.CheckQueue(c, t, 1)
}

func TestInviteGet(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	inviterStr, err := setupPlayer(c, TESTNICK1, TESTEMAIL1)
	if err != nil {
		t.Fatalf("Error setting up inviter")
	}
	_, _, err = Create(c, inviterStr, CLAN1, "lol")
	if err != nil {
		t.Fatalf("\nError creating clan %s", err)
	}
	inviteeStr, err := setupPlayer(c, TESTNICK2, TESTEMAIL2)
	if err != nil {
		t.Fatalf("\nError setting up invitee", err)
	}
	inviteePlayerKey, err := datastore.DecodeKey(inviteeStr)
	if err != nil {
		t.Fatalf("\nError decoding key", err)
	}
	invitedPlayer := new(player.Player)
	if err := datastore.Get(c, inviteePlayerKey, invitedPlayer); err != nil {
		t.Fatalf("\nError getting player", err)
	}
	testutils.PurgeQueue(c, t)
	if err := InvitePlayer(c, inviterStr, invitedPlayer.ID); err != nil {
		t.Fatalf("\nError sending invite %s", err)
	}
	testutils.CheckQueue(c, t, 1)
	testutils.PurgeQueue(c, t)
	time.Sleep(1 * time.Second)
	invites, err := InvitesForPlayer(c, inviteeStr)
	if err != nil {
		t.Fatalf("\nError getting invites %s", err)
	}
	t.Logf("\n invites %+v", invites[0])
}

func TestJoin(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	inviterStr, err := setupPlayer(c, TESTNICK1, TESTEMAIL1)
	if err != nil {
		t.Errorf("Error setting up inviter")
	}
	_, _, err = Create(c, inviterStr, CLAN1, "lol")
	if err != nil {
		t.Errorf("\nError creating clan %s", err)
	}
	inviteeStr, err := setupPlayer(c, TESTNICK2, TESTEMAIL2)
	if err != nil {
		t.Errorf("\nError setting up invitee", err)
	}
	inviteePlayerKey, err := datastore.DecodeKey(inviteeStr)
	if err != nil {
		t.Errorf("\ninvitee decode key error", err)
	}
	invitedPlayer := new(player.Player)
	if err := datastore.Get(c, inviteePlayerKey, invitedPlayer); err != nil {
		t.Fatalf("\nError getting player", err)
	}
	if err := InvitePlayer(c, inviterStr, invitedPlayer.ID); err != nil {
		t.Fatalf("\nError sending invite %s", err)
	}
	inviteStr := fmt.Sprintf("%d%d", 1, 2)
	inviteKey := datastore.NewKey(c, "Invite", inviteStr, 0, nil)
	if err := Join(c, inviteeStr, inviteKey.Encode()); err != nil {
		t.Fatalf("\nerror joining clan %s", err)

	}
	joinedPlayer := new(player.Player)
	if err := datastore.Get(c, inviteePlayerKey, joinedPlayer); err != nil {
		t.Fatalf("\nerror getting player joined %s", err)
	}
	t.Logf("\n joined player clan %+v", joinedPlayer)
}

func TestStatus(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	inviterStr, err := setupPlayer(c, TESTNICK1, TESTEMAIL1)
	if err != nil {
		t.Fatalf("Error setting up inviter")
	}
	clanGuid, _, err := Create(c, inviterStr, CLAN1, "lol")
	if err != nil {
		t.Fatalf("\nError creating clan %s, %s", err, clanGuid)
	}
	inviteeStr, err := setupPlayer(c, TESTNICK2, TESTEMAIL2)
	if err != nil {
		t.Fatalf("\nError setting up invitee", err)
	}
	inviteePlayerKey, err := datastore.DecodeKey(inviteeStr)
	if err != nil {
		t.Errorf("error loading profile %s", err)
	}
	invitedPlayer := new(player.Player)
	if err := datastore.Get(c, inviteePlayerKey, invitedPlayer); err != nil {
		t.Fatalf("\nError getting player", err)
	}
	//clanKey := datastore.NewKey(c, "Clan", clanGuid, 0, nil)
	//clanStr := clanKey.Encode()
	if err := InvitePlayer(c, inviterStr, invitedPlayer.ID); err != nil {
		t.Fatalf("\nError sending invite %s", err)
	}
	inviteStr := fmt.Sprintf("%d%d", 1, 2)
	inviteKey := datastore.NewKey(c, "Invite", inviteStr, 0, nil)
	if err := Join(c, inviteeStr, inviteKey.Encode()); err != nil {
		t.Fatalf("\nerror joining clan %s", err)

	}
	time.Sleep(1 * time.Second)
	clan := new(Clan)
	if err := Status(c, inviterStr, clan); err != nil {
		t.Fatalf("\n error status clan %s", err)
	}

	t.Logf("status clan : %+v \n\n", clan)
	if err := Leave(c, inviteeStr); err != nil {
		t.Fatalf("\n error leaving clan %s", err)
	}
	time.Sleep(1 * time.Second)
	if err := Status(c, inviterStr, clan); err != nil {
		t.Fatalf("\n error status clan %s", err)
	}
	t.Logf("status clan : %+v", clan)
}

func TestLeave(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	inviterStr, err := setupPlayer(c, TESTNICK1, TESTEMAIL1)
	if err != nil {
		t.Fatalf("Error setting up inviter")
	}
	_, _, err = Create(c, inviterStr, CLAN1, "lol")
	if err != nil {
		t.Fatalf("\nError creating clan %s", err)
	}
	inviteeStr, err := setupPlayer(c, TESTNICK2, TESTEMAIL2)
	if err != nil {
		t.Fatalf("\nError setting up invitee", err)
	}
	inviteePlayerKey, err := datastore.DecodeKey(inviteeStr)
	if err != nil {
		t.Errorf("error loading profile %s", err)
	}
	invitedPlayer := new(player.Player)
	if err := datastore.Get(c, inviteePlayerKey, invitedPlayer); err != nil {
		t.Fatalf("\nError getting player", err)
	}
	if err := InvitePlayer(c, inviterStr, invitedPlayer.ID); err != nil {
		t.Fatalf("\nError sending invite %s", err)
	}
	inviteStr := fmt.Sprintf("%d%d", 1, 2)
	inviteKey := datastore.NewKey(c, "Invite", inviteStr, 0, nil)
	if err := Join(c, inviteeStr, inviteKey.Encode()); err != nil {
		t.Fatalf("\nerror joining clan %s", err)

	}

	time.Sleep(1 * time.Second)
	if err := Leave(c, inviteeStr); err != nil {
		t.Fatalf("\n error leaving clan %s", err)
	}

	leftPlayer := new(player.Player)
	if err := datastore.Get(c, inviteePlayerKey, leftPlayer); err != nil {
		t.Fatalf("\nerror getting player left %s", err)
	}
	t.Logf("\n left player clan member %+v", leftPlayer)
	time.Sleep(1 * time.Second)
	if err := Join(c, inviteeStr, inviteKey.Encode()); err != nil {
		t.Logf("error %s", err)

	} else {
		//t.Fatalf("expected join error") can't test with aetest: no events recorded because of no taskqueue...
	}

}

func TestConnection(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	declareStr, err := setupPlayer(c, TESTNICK1, TESTEMAIL1)
	if err != nil {
		t.Fatalf("Error setting up declare player")
	}
	victimStr, err := setupPlayer(c, TESTNICK2, TESTEMAIL2)
	if err != nil {
		t.Fatalf("Error setting up victim player")
	}
	clanGuid1, _, err := Create(c, declareStr, CLAN1, "lol")
	if err != nil {
		t.Fatalf("\nError creating clan %s %s", err, clanGuid1)
	}
	clanGuid2, _, err := Create(c, victimStr, CLAN2, "lol")
	if err != nil {
		t.Fatalf("\nError creating clan %s %s", err, clanGuid2)
	}
	clan2Key := datastore.NewKey(c, "Clan", clanGuid2, 0, nil)
	time.Sleep(1 * time.Second)
	team := new(Clan)
	if err := datastore.Get(c, clan2Key, team); err != nil {
		t.Fatalf("\n error getting defending clan %s", err)
	}
	if err := Connect(c, declareStr, team.ID); err != nil {
		t.Fatalf("\n error connecting to clan %s", err)
	}

}

func TestCloseConnection(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	declareStr, err := setupPlayer(c, TESTNICK1, TESTEMAIL1)
	if err != nil {
		t.Fatalf("Error setting up declare player")
	}
	victimStr, err := setupPlayer(c, TESTNICK2, TESTEMAIL2)
	if err != nil {
		t.Fatalf("Error setting up victim player")
	}
	clanGuid1, _, err := Create(c, declareStr, CLAN1, "lol")
	if err != nil {
		t.Fatalf("\nError creating clan %s, %s", err, clanGuid1)
	}
	clanGuid2, _, err := Create(c, victimStr, CLAN2, "lol")
	if err != nil {
		t.Fatalf("\nError creating clan %s", err)
	}
	clan2Key := datastore.NewKey(c, "Clan", clanGuid2, 0, nil)
	time.Sleep(1 * time.Second)
	team := new(Clan)
	if err := datastore.Get(c, clan2Key, team); err != nil {
		t.Fatalf("\n error getting defending clan %s", err)
	}
	if err := Connect(c, declareStr, team.ID); err != nil {
		t.Fatalf("\n error connecting to clan %s", err)
	}

	if err := DisConnect(c, declareStr, team.ID); err != nil {
		t.Logf("\n error closing connection %s", err)
	}

}

func TestPromoteOrDemote(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	initStr, err := setupPlayer(c, TESTNICK1, TESTEMAIL1)
	if err != nil {
		t.Fatalf("Error setting up init player")
	}
	promoteStr, err := setupPlayer(c, TESTNICK2, TESTEMAIL2)
	if err != nil {
		t.Fatalf("Error setting up promote player")
	}
	clanGuid1, _, err := Create(c, initStr, CLAN1, "lol")
	if err != nil {
		t.Fatalf("\nError creating clan %s", err)
	}

	promoteKey, err := datastore.DecodeKey(promoteStr)
	if err != nil {
		t.Fatalf("Error decoding key")
	}

	promotePlayer := new(player.Player)
	if err := datastore.Get(c, promoteKey, promotePlayer); err != nil {
		t.Fatalf("Error fetching promote player")
	}
	promotePlayer.ClanKey = datastore.NewKey(c, "Clan", clanGuid1, 0, nil)
	promotePlayer.Clan = CLAN1
	promotePlayer.ClanTag = "lol"

	if _, err := datastore.Put(c, promoteKey, promotePlayer); err != nil {
		t.Fatalf("\n error saving promotePlayer %s", err)
	}
	if err := PromoteOrDemote(c, initStr, promotePlayer.ID, player.LIEUTENANT); err != nil {
		t.Fatalf("\n error promoting player %s", err)
	}
}

func TestKick(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	initStr, err := setupPlayer(c, TESTNICK1, TESTEMAIL1)
	if err != nil {
		t.Fatalf("Error setting up init player")
	}
	kickStr, err := setupPlayer(c, TESTNICK2, TESTEMAIL2)
	if err != nil {
		t.Fatalf("Error setting up promote player")
	}
	clanGuid1, _, err := Create(c, initStr, CLAN1, "lol")
	if err != nil {
		t.Fatalf("\nError creating clan %s", err)
	}

	kickKey, err := datastore.DecodeKey(kickStr)
	if err != nil {
		t.Fatalf("Error decoding key")
	}

	kickPlayer := new(player.Player)
	if err := datastore.Get(c, kickKey, kickPlayer); err != nil {
		t.Fatalf("Error fetching kick player")
	}

	kickPlayer.ClanKey = datastore.NewKey(c, "Clan", clanGuid1, 0, nil)
	kickPlayer.Clan = CLAN1
	kickPlayer.ClanTag = "lol"

	if _, err := datastore.Put(c, kickKey, kickPlayer); err != nil {
		t.Fatalf("\n error saving player %s", err)
	}
	if err := Kick(c, initStr, kickPlayer.ID); err != nil {
		t.Fatalf("\n error kicking player %s", err)
	}
}

func TestUpdateMessage(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	initStr, err := setupPlayer(c, TESTNICK1, TESTEMAIL1)
	if err != nil {
		t.Fatalf("Error setting up init player")
	}

	clanGuid1, _, err := Create(c, initStr, CLAN1, "lol")
	if err != nil {
		t.Fatalf("\nError creating clan %s", err)
	}

	clan1Key := datastore.NewKey(c, "Clan", clanGuid1, 0, nil)

	update := &MessageUpdate{
		Content: "blablabalbal",
	}

	if err := UpdateMessage(c, initStr, update); err != nil {
		t.Fatalf("\n error updating clanmessage %s", err)
	}

	team := new(Clan)
	if err := datastore.Get(c, clan1Key, team); err != nil {
		t.Fatalf("Error fetching clan")
	}

	t.Logf("updated clanmessage : %s", team.Message)

}

func TestPublicStatus(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	playerStr, err := setupPlayer(c, TESTNICK1, TESTEMAIL1)
	if err != nil {
		t.Fatalf("Error setting up player")
	}
	clanGuid, _, err := Create(c, playerStr, CLAN1, "lol")
	if err != nil {
		t.Fatalf("\nError creating clan %s", err)
	}
	time.Sleep(1 * time.Second)
	clanKey := datastore.NewKey(c, "Clan", clanGuid, 0, nil)
	clanStr := clanKey.Encode()
	clan := new(Clan)
	if err := PublicStatus(c, clanStr, "1", clan); err != nil {
		t.Fatalf("\n error status clan %s", err)
	}
	t.Logf("status clan : %+v \n\n", clan)
}

func TestClanList(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	playerStr, err := setupPlayer(c, TESTNICK1, TESTEMAIL1)
	if err != nil {
		t.Fatalf("Error setting up player")
	}
	clanGuid, _, err := Create(c, playerStr, CLAN1, "lol")
	if err != nil {
		t.Fatalf("\nError creating clan %s, guid %s", err, clanGuid)
	}
	time.Sleep(1 * time.Second)
	list, err := List(c, playerStr, "0", "")
	if err != nil {
		t.Fatalf("error getting public player list: %s", err)
	}
	t.Logf("list retrieved : %+v \n", list)

}
