package message

import (
	"appengine"
	"appengine/aetest"
	//"appengine/datastore"
	//"errors"
	//"fmt"
	"mj0lk.be/netwars/clan"
	"mj0lk.be/netwars/player"

	"testing"
	"time"
)

const (
	TESTNICK  = "player"
	TESTEMAIL = "player@hotmail.com"
	TESTCLAN  = "clanName"
	TESTTAG   = "tag"
)

func setupPlayer(c appengine.Context, nick, email string) (string, error) {
	playerStr, _, err := player.Create(c, nick, email)
	if err != nil {
		return "", err
	}
	return playerStr, nil
}

func TestCreateBoard(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	playerStr, err := setupPlayer(c, TESTNICK, TESTEMAIL)
	if err != nil {
		t.Fatalf("Error setting up player")
	}

	message := Message{
		Pkey:       playerStr,
		Subject:    "subject",
		Content:    "content",
		Scope:      1,
		AccessName: "Public",
	}

	if err := CreateOrUpdate(c, &message); err != nil {
		t.Fatalf("\nError creating board %s", err)
	}

	time.Sleep(1 * time.Second)

	boards, err := PublicBoards(c, playerStr, "")
	if err != nil {
		t.Fatalf("\nError fetching public boards %s", err)
	}
	t.Logf("fetched public boards: %+v \n", boards)
}

func TestCreateClanBoard(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	playerStr, err := setupPlayer(c, TESTNICK, TESTEMAIL)
	if err != nil {
		t.Fatalf("Error setting up player")
	}
	if _, _, err := clan.Create(c, playerStr, TESTCLAN, TESTTAG); err != nil {
		t.Fatalf("\nError creating clan %s", err)
	}

	message := Message{
		Pkey:       playerStr,
		Subject:    "subject",
		Content:    "content",
		Scope:      2,
		AccessName: "Public",
	}

	if err := CreateOrUpdate(c, &message); err != nil {
		t.Fatalf("\nError creating board %s", err)
	}

	time.Sleep(1 * time.Second)

	boards, err := ClanBoards(c, playerStr, "")
	if err != nil {
		t.Fatalf("\nError fetching clan boards %s", err)
	}
	t.Logf("fetched clan boards: %+v \n", boards)
}

func TestCreateThread(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	playerStr, err := setupPlayer(c, TESTNICK, TESTEMAIL)
	if err != nil {
		t.Fatalf("Error setting up player")
	}
	//create board
	message := &Message{
		Pkey:       playerStr,
		Subject:    "subject",
		Content:    "content",
		Scope:      1,
		AccessName: "Public",
	}

	if err := CreateOrUpdate(c, message); err != nil {
		t.Fatalf("\nError creating board %s", err)
	}

	time.Sleep(1 * time.Second)

	boards, err := PublicBoards(c, playerStr, "")
	if err != nil {
		t.Fatalf("\nError fetching public boards %s", err)
	}
	t.Logf("fetched public boards: %+v \n", boards)
	message.Bkey = boards.Messages[0].EncodedKey
	message.Content = "threadcontent"
	message.Subject = "threadsubject"

	if err := CreateOrUpdate(c, message); err != nil {
		t.Fatalf("\nError creating thread %s", err)
	}

	time.Sleep(1 * time.Second)

	threads, err := Messages(c, "threads", playerStr, message.Bkey, "")
	if err != nil {
		t.Fatalf("\nError fetching threads %s", err)
	}

	t.Logf("fetched threads: %+v \n", threads)
}

func TestCreateMessage(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	playerStr, err := setupPlayer(c, TESTNICK, TESTEMAIL)
	if err != nil {
		t.Fatalf("Error setting up player")
	}

	message := &Message{
		Pkey:       playerStr,
		Subject:    "subject",
		Content:    "content",
		Scope:      1,
		AccessName: "Public",
	}

	if err := CreateOrUpdate(c, message); err != nil {
		t.Fatalf("\nError creating board %s", err)
	}

	time.Sleep(1 * time.Second)

	boards, err := PublicBoards(c, playerStr, "")
	if err != nil {
		t.Fatalf("\nError fetching public boards %s", err)
	}
	t.Logf("fetched public boards: %+v \n", boards)
	message.Bkey = boards.Messages[0].EncodedKey
	message.Content = "threadcontent"
	message.Subject = "threadsubject"

	if err := CreateOrUpdate(c, message); err != nil {
		t.Fatalf("\nError creating thread %s", err)
	}

	time.Sleep(1 * time.Second)

	threads, err := Messages(c, "threads", playerStr, message.Bkey, "")
	if err != nil {
		t.Fatalf("\nError fetching threads %s", err)
	}

	t.Logf("fetched threads: %+v \n", threads)

	message.Bkey = ""
	message.Tkey = threads.Messages[0].EncodedKey
	message.Content = "messagecontent"
	message.Subject = "messagesubject"

	if err := CreateOrUpdate(c, message); err != nil {
		t.Fatalf("\nError creating message %s", err)
	}

	messages, err := Messages(c, "messages", playerStr, message.Tkey, "")
	if err != nil {
		t.Fatalf("\nError fetching messages %s", err)
	}

	t.Logf("fetched messages: %+v \n", messages)
}
