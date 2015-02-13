package user

import (
	"appengine"
	"appengine/aetest"
	"appengine/datastore"
	"errors"
	"netwars/program"
	"netwars/testutils"
	"testing"
	"time"
)

const (
	TESTNICK  = "testnick"
	TESTEMAIL = "2mjolk@gmail.com"
	PROGRAM1  = "Swarm connector"
	PROGRAM2  = "Swarm mark IV"
)

func TestValidPlayer(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	uErr, err := ValidPlayer(c, "2mjolk@gmail.com", "mjolk")
	if err != nil {
		t.Fatalf("error validplayer %s", err)
	}
	if uErr != nil {
		t.Logf("usererr %v", uErr)
	}
	cuErr, cerr := ValidPlayer(c, "2mjolk@gmail.com", "mjolk")
	if cerr != nil {
		t.Fatalf("error validplayer %s", err)
	}
	if cuErr != nil {
		t.Logf("usererr %v", cuErr)
	}
	if cuErr["nick"] < 1 || cuErr["email"] < 1 {
		t.Fatalf("failed, shoould detect existing email and nick")
	}
}

func TestCreate(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	playerKeyStr, usererr, err := Create(c, TESTNICK, TESTEMAIL)
	if err != nil {
		t.Fatalf("error creating player : %s", err)
	}
	if len(playerKeyStr) == 0 {
		t.Fatalf("no player key generated \n")
	}
	if usererr["nick"]+usererr["email"] > 0 {
		t.Fatalf("Unexpected user error creating player: %v \n", usererr)
	}
}

func TestUpdateProfile(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	playerKeyStr, usererr, err := Create(c, TESTNICK, TESTEMAIL)
	if err != nil {
		t.Fatalf("error creating player : %s", err)
	}
	if len(playerKeyStr) == 0 {
		t.Fatalf("no player key generated \n")
	}
	if len(usererr) > 0 {
		t.Fatalf("Unexpected user error creating player: %v \n", usererr)
	}
	profileUpdate := ProfileUpdate{
		Pkey:      playerKeyStr,
		Name:      "Dries",
		Birthday:  "1979-Apr-13",
		Country:   "Belgium",
		Language:  "Nl",
		Address:   "PLantin & Moretuslei 2018 Antwerpen",
		Signature: "Carpe Diem",
	}

	if err := UpdateProfile(c, profileUpdate); err != nil {
		t.Fatalf("Error updating profile")
	}

	player := new(Player)

	playerKey, _ := datastore.DecodeKey(playerKeyStr)

	if err := datastore.Get(c, playerKey, player); err != nil {
		t.Fatalf("Error loading player")
	}

	t.Logf("updated profile %+v \n", player)
}

func setupPlayer(c appengine.Context) (string, error) {
	playerKeyStr, usererr, err := Create(c, TESTNICK, TESTEMAIL)
	if err != nil {
		return "", err

	}
	if usererr != nil {
		return "", errors.New("unexpected user error")
	}
	return playerKeyStr, nil
}

func setupProgram(c appengine.Context) error {
	connProgram := &program.Program{
		Name:        PROGRAM1,
		TypeName:    "Connection",
		Attack:      0,
		Life:        80,
		Cycles:      200,
		Memory:      1,
		Bandwidth:   3000,
		Description: "Connector for Swarm type programs",
		Effectors:   []string{"Swarm"},
	}
	jprogram := &program.Program{
		Name:        PROGRAM2,
		Attack:      120,
		Life:        180,
		TypeName:    "Swarm",
		Cycles:      80,
		Memory:      0.50,
		Description: "Swarm mark IV lethal bandwidth threat",
		Effectors:   []string{"Swarm", "Hunter/Killer", "d0s"},
	}

	if err := program.CreateOrUpdate(c, connProgram); err != nil {
		return err
	}
	if err := program.CreateOrUpdate(c, jprogram); err != nil {
		return err
	}
	return nil
}

func checkProgram(t *testing.T, status *PlayerState, name string, amount int64) {
	for _, group := range status.Programs {
		t.Logf("--------------------------- %+v \n", group)
		t.Logf("| group usage : %d |\n", group.Usage)
		t.Logf("| group Yield : %d |\n", group.Yield)
		t.Logf("| group.Power %s |\n", group.Power)
		for _, program := range group.Programs {
			//t.Logf("\n\n program : %s", program.Name)
			t.Logf("|program name: %s  program amount: %d |\n", program.Name, program.Amount)
			if program.Name == name {
				if program.Amount == amount {
					t.Logf("|program amount ok -- |\n")
				} else {
					t.Fatalf("|prograp amount fail amount: %d|", program.Amount)
				}
			}
		}
	}
}

func TestAllocate(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	connectorKey := datastore.NewKey(c, "Program", PROGRAM1, 0, nil)
	programKey := datastore.NewKey(c, "Program", PROGRAM2, 0, nil)
	playerKeyStr, err := setupPlayer(c)
	if err != nil {
		t.Fatalf("player setup error : %s \n", err)
	}
	if err := setupProgram(c); err != nil {
		t.Fatalf("setup program error %s", err)
	}
	if err := Allocate(c, playerKeyStr, connectorKey.Encode(), "1"); err != nil {
		t.Fatalf("allocate1 error %s \n", err)
	}
	if err := Allocate(c, playerKeyStr, programKey.Encode(), "5"); err != nil {
		t.Fatalf("allocate2 error :%+v \n", err)
	}
	status := new(PlayerState)
	if _, err := Status(c, playerKeyStr, status); err != nil {
		t.Fatalf(" status err : %s", err)
	}
	checkProgram(t, status, PROGRAM2, 5)
	testutils.CheckQueue(c, t, 2)
}

func TestDeallocate(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	playerKeyStr, err := setupPlayer(c)
	if err != nil {
		t.Fatalf("setup player error: %s", err)
	}
	if err := setupProgram(c); err != nil {
		t.Fatalf("error setup program : %s \n", err)
	}
	connectorKey := datastore.NewKey(c, "Program", PROGRAM1, 0, nil)
	programKey := datastore.NewKey(c, "Program", PROGRAM2, 0, nil)
	if err := Allocate(c, playerKeyStr, connectorKey.Encode(), "1"); err != nil {
		t.Fatalf("allocate error %s \n", err)
	}

	if err := Allocate(c, playerKeyStr, programKey.Encode(), "5"); err != nil {
		t.Fatalf("allocate error: %s \n", err)
	}

	status := new(PlayerState)
	if _, err := Status(c, playerKeyStr, status); err != nil {
		t.Fatalf(" status err : %s", err)
	}
	checkProgram(t, status, PROGRAM2, 5)

	if err := Deallocate(c, playerKeyStr, programKey.Encode(), "3"); err != nil {
		t.Fatalf("deallocation error: %s \n", err)
	}

	if _, err := Status(c, playerKeyStr, status); err != nil {
		t.Fatalf(" status err : %s", err)
	}
	checkProgram(t, status, PROGRAM2, 2)
	testutils.CheckQueue(c, t, 3)
}

func TestProfileList(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	playerKeyStr, err := setupPlayer(c)
	if err != nil {
		t.Fatalf("setup player error: %s", err)
	}
	time.Sleep(1 * time.Second)
	list, err := List(c, playerKeyStr, "false", "")
	if err != nil {
		t.Fatalf("error getting public player list: %s", err)
	}
	t.Logf("list retrieved : %+v \n", list)

}
