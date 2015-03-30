package attack

import (
	"appengine"
	"appengine/aetest"
	"appengine/datastore"
	"errors"
	"mj0lk.be/netwars/clan"
	"mj0lk.be/netwars/player"
	"mj0lk.be/netwars/program"
	"mj0lk.be/netwars/testutils"
	"mj0lk.be/netwars/utils"
	"testing"
	"time"
)

const (
	ANICK         = "attacker"
	BNICK         = "defender"
	AEMAIL        = "testmail@hotmail.com"
	BEMAIL        = "testmail2@hotmail.com"
	SWCONN        = "Swarm Connection"
	HKCONN        = "Hunter/Killer Connection"
	D0SCONN       = "D0S Connection"
	MUTCONN       = "Mutator Connection"
	INTCONN       = "Spy Connection"
	ICECONN       = "Ice connection"
	SWARM         = "Swarm mark IV"
	MUTATOR       = "Mutator IV"
	HUNTERKILLER  = "Hunter/Killer program"
	HUNTERKILLER2 = "Hunter/Killer program2"
	D0S           = "d0s program"
	INTP          = "spy program"
	ICEP          = "ice program"
	INFECTP       = "infect program"
	CLAN1         = "Clan1"
	CLAN2         = "Clan2"
)

func setupPrograms(c appengine.Context) error {
	SWConnect := &program.Program{
		Name:        SWCONN,
		TypeName:    "Connection",
		Attack:      0,
		Life:        80,
		Cycles:      200,
		Memory:      1,
		Bandwidth:   3000,
		Description: "Connector for Swarm type programs",
		Effectors:   []string{"Swarm"},
	}
	HKConnect := &program.Program{
		Name:        HKCONN,
		TypeName:    "Connection",
		Attack:      0,
		Life:        80,
		Cycles:      100,
		Memory:      1,
		Bandwidth:   3000,
		Description: "Connector for Hunter/Killer type programs",
		Effectors:   []string{"Hunter/Killer"},
	}
	D0SConnect := &program.Program{
		Name:        D0SCONN,
		TypeName:    "Connection",
		Attack:      0,
		Life:        80,
		Cycles:      200,
		Memory:      1,
		Bandwidth:   3000,
		Description: "Connector for D0S type programs",
		Effectors:   []string{"d0s"},
	}
	MUTConnect := &program.Program{
		Name:        MUTCONN,
		TypeName:    "Connection",
		Attack:      0,
		Life:        80,
		Cycles:      200,
		Memory:      1,
		Bandwidth:   3000,
		Description: "Connector for Mutator type programs",
		Effectors:   []string{"Mutator"},
	}
	INTConnect := &program.Program{
		Name:        INTCONN,
		TypeName:    "Connection",
		Attack:      0,
		Life:        80,
		Cycles:      200,
		Memory:      1,
		Bandwidth:   3000,
		Description: "Connector for Intelligence type programs",
		Effectors:   []string{"Intelligence"},
	}
	ICEConnect := &program.Program{
		Name:        ICECONN,
		TypeName:    "Connection",
		Attack:      0,
		Life:        80,
		Cycles:      200,
		Memory:      1,
		Bandwidth:   3000,
		Description: "Connector for Intelligence type programs",
		Effectors:   []string{"Ice"},
	}
	swarmProg := &program.Program{
		Name:        SWARM,
		Attack:      40,
		Life:        40,
		TypeName:    "Swarm",
		Cycles:      70,
		Memory:      1,
		Description: "Swarm mark IV lethal bandwidth threat",
		Effectors:   []string{"Hunter/Killer", "Connection"},
	}
	mutProg := &program.Program{
		Name:        MUTATOR,
		Attack:      15,
		Life:        15,
		TypeName:    "Mutator",
		Cycles:      20,
		Memory:      0.2,
		Description: "Mutator program",
		Effectors:   []string{"Mutator", "d0s"},
	}
	hkProg := &program.Program{
		Name:        HUNTERKILLER,
		Attack:      15,
		Life:        15,
		TypeName:    "Hunter/Killer",
		Cycles:      80,
		Memory:      0.2,
		Description: "Hunter/Killer mark II",
		Effectors:   []string{"Mutator", "Swarm"},
	}
	hkProg2 := &program.Program{
		Name:        HUNTERKILLER2,
		Attack:      15,
		Life:        15,
		TypeName:    "Hunter/Killer",
		Cycles:      20,
		Memory:      0.2,
		Description: "Hunter/Killer mark VI",
		Effectors:   []string{"Swarm", "d0s"},
	}
	d0sProg := &program.Program{
		Name:        D0S,
		Attack:      5,
		Life:        5,
		TypeName:    "d0s",
		Cycles:      20,
		Memory:      0.1,
		Description: "d0s prog",
		Effectors:   []string{"Hunter/Killer", "Swarm"},
	}
	intProg := &program.Program{
		Name:        INTP,
		Attack:      0,
		Life:        5,
		TypeName:    "Intelligence",
		Cycles:      200,
		Memory:      1,
		Description: "intelligence prog",
		Effectors:   []string{"Hunter/Killer"}, //just assign one program it will pick the right group in this case it will send a report for offensive programs
	}
	iceProg := &program.Program{
		Name:        ICEP,
		Attack:      0,
		Life:        0,
		TypeName:    "Ice",
		Cycles:      200,
		Memory:      1,
		Description: "ice prog",
		InfectName:  INFECTP,
	}
	dur := time.Duration(3) * time.Hour
	infProg := &program.Program{
		Name:        INFECTP,
		Attack:      0,
		Life:        0,
		TypeName:    "Infect",
		Cycles:      200,
		Memory:      2,
		Description: "infect prog",
		Ettl:        int64(dur.Seconds()),
	}
	programs := []*program.Program{SWConnect, HKConnect, D0SConnect, MUTConnect, INTConnect, ICEConnect, swarmProg, mutProg, hkProg, hkProg2, d0sProg, intProg, iceProg, infProg}
	for _, prog := range programs {
		if err := program.CreateOrUpdate(c, prog); err != nil {
			return err
		}
	}
	return nil
}

func setupPlayer(c appengine.Context, nick string, email string) (string, error) {
	cr := player.Creation{email, nick, "testpassword"}
	tokenStr, usererr, err := player.Create(c, cr)
	if err != nil {
		return "", err

	}
	if usererr != nil {
		return "", errors.New("unexpected user error")
	}
	playerKeyStr, _ := utils.ValidateToken(tokenStr)
	return playerKeyStr, nil
}

//create attacklist with all available offensive programs
func getAttackPrograms(c appengine.Context, playerKey *datastore.Key) ([]ActiveProgram, error) {
	state := new(player.Player)
	if err := player.Status(c, playerKey.Encode(), state); err != nil {
		return nil, err
	}
	var attackerPrograms []ActiveProgram
	for _, atpe := range OffensiveTypes {
		if aGroupForType, ok := state.Programs[atpe]; ok {
			for _, prog := range aGroupForType.Programs {
				attackerPrograms = append(attackerPrograms, ActiveProgram{prog.ProgramKey.Encode(), prog.Amount})
			}
		}
	}
	return attackerPrograms, nil
}

func getActiveProgram(c appengine.Context, playerKey *datastore.Key, tpe int64) ([]ActiveProgram, error) {
	state := new(player.Player)
	if err := player.Status(c, playerKey.Encode(), state); err != nil {
		return nil, err
	}
	activeProgs := make([]ActiveProgram, 1)
	if group, ok := state.Programs[tpe]; ok {
		// can only select one program
		activeProgs[0] = ActiveProgram{group.Programs[0].ProgramKey.Encode(), 1}
	}
	return activeProgs, nil
}

func TestSpy(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	attackerStr, err := setupPlayer(c, ANICK, AEMAIL)
	if err != nil {
		t.Fatalf("setup players error %s \n", err)
	}
	defenderStr, err := setupPlayer(c, BNICK, BEMAIL)
	if err != nil {
		t.Fatalf("setup players error %s \n", err)
	}
	if err := setupPrograms(c); err != nil {
		t.Fatalf("error setup programs: %s \n", err)
	}
	attackerKey, err := datastore.DecodeKey(attackerStr)
	defenderKey, err := datastore.DecodeKey(defenderStr)
	if err != nil {
		t.Fatalf("error decoding key %s \n", err)
	}
	//	swarmConnKey := datastore.NewKey(c, "Program", SWCONN, 0, nil)
	//	swarmKey := datastore.NewKey(c, "Program", SWARM, 0, nil)
	hkConnKey := datastore.NewKey(c, "Program", HKCONN, 0, nil)
	hkKey := datastore.NewKey(c, "Program", HUNTERKILLER, 0, nil)
	hkKey2 := datastore.NewKey(c, "Program", HUNTERKILLER2, 0, nil)
	intConnKey := datastore.NewKey(c, "Program", INTCONN, 0, nil)
	intKey := datastore.NewKey(c, "Program", INTP, 0, nil)
	//mutConnKey := datastore.NewKey(c, "Program", MUTCONN, 0, nil)
	//mutKey := datastore.NewKey(c, "Program", MUTATOR, 0, nil)
	all := player.Allocation{intConnKey.Encode(), 1}
	if err := player.Allocate(c, attackerStr, all); err != nil {
		t.Fatalf("allocate attacker connection error %s \n", err)
	}
	all.PrgKey = intKey.Encode()
	all.Amount = 1
	if err := player.Allocate(c, attackerStr, all); err != nil {
		t.Fatalf("allocate attacker spy program error %s \n", err)
	}
	all.PrgKey = hkConnKey.Encode()
	all.Amount = 1
	if err := player.Allocate(c, defenderStr, all); err != nil {
		t.Fatalf("allocate defending program connection error %s \n", err)
	}
	all.PrgKey = hkKey.Encode()
	all.Amount = 5
	if err := player.Allocate(c, defenderStr, all); err != nil {
		t.Fatalf("allocate defending program error %s \n", err)
	}
	all.PrgKey = hkKey2.Encode()
	all.Amount = 5
	if err := player.Allocate(c, defenderStr, all); err != nil {
		t.Fatalf("allocate defending program error %s \n", err)
	}
	/*	if err := player.Allocate(c, defenderStr, swarmConnKey.Encode(), "1"); err != nil {
			t.Fatalf("allocate defending program connection error %s \n", err)
		}
		if err := player.Allocate(c, defenderStr, swarmKey.Encode(), "1"); err != nil {
			t.Fatalf("allocate defending program error %s \n", err)
		}*/
	attackPrograms, err := getActiveProgram(c, attackerKey, program.INT)
	if err != nil {
		t.Fatalf("error loading spyprogram %s\n", err)
	}
	defender := new(player.Player)
	if err := datastore.Get(c, defenderKey, defender); err != nil {
		t.Fatalf("errror loading defender \n", err)
	}
	attackCfg := AttackCfg{
		AttackType:     INT,
		Pkey:           attackerKey.Encode(),
		Target:         defender.PlayerID,
		ActivePrograms: attackPrograms,
	}
	testutils.PurgeQueue(c, t)
	spyEvent, err := Spy(c, attackCfg)
	if err != nil {
		t.Fatalf("spy error %s \n", err)
	}
	t.Logf("\n <<< SPYEVENT >>> \n%+v\n", spyEvent)
	testutils.CheckQueue(c, t, 1)
}

func TestIce(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	attackerStr, err := setupPlayer(c, ANICK, AEMAIL)
	if err != nil {
		t.Fatalf("setup players error %s \n", err)
	}
	defenderStr, err := setupPlayer(c, BNICK, BEMAIL)
	if err != nil {
		t.Fatalf("setup players error %s \n", err)
	}
	if err := setupPrograms(c); err != nil {
		t.Fatalf("error setup programs: %s \n", err)
	}
	attackerKey, err := datastore.DecodeKey(attackerStr)
	defenderKey, err := datastore.DecodeKey(defenderStr)
	if err != nil {
		t.Fatalf("error decoding key %s \n", err)
	}
	iceConnKey := datastore.NewKey(c, "Program", ICECONN, 0, nil)
	iceKey := datastore.NewKey(c, "Program", ICEP, 0, nil)
	all := player.Allocation{iceConnKey.Encode(), 1}
	if err := player.Allocate(c, attackerStr, all); err != nil {
		t.Fatalf("allocate ice connection error %s \n", err)
	}
	all.PrgKey = iceKey.Encode()
	all.Amount = 1
	if err := player.Allocate(c, attackerStr, all); err != nil {
		t.Fatalf("allocate attacker spy program error %s \n", err)
	}
	attackPrograms, err := getActiveProgram(c, attackerKey, program.ICE)
	if err != nil {
		t.Fatalf("error loading spyprogram %s\n", err)
	}
	defender := new(player.Player)
	if err := datastore.Get(c, defenderKey, defender); err != nil {
		t.Fatalf("errror loading defender \n", err)
	}
	attackCfg := AttackCfg{
		AttackType:     ICE,
		Pkey:           attackerKey.Encode(),
		Target:         defender.PlayerID,
		ActivePrograms: attackPrograms,
	}
	if err := player.Status(c, defenderStr, defender); err != nil {
		t.Fatalf("error fetching state defender")
	}
	t.Logf("defender bandwidthusage: %f \n", defender.BandwidthUsage)
	testutils.PurgeQueue(c, t)
	spyEvent, err := Ice(c, attackCfg)
	if err != nil {
		t.Fatalf("spy error %s \n", err)
	}
	t.Logf("\n <<< ICEEVENT >>> \n%+v\n", spyEvent.Event)
	testutils.CheckQueue(c, t, 1)
	if err := player.Status(c, defenderStr, defender); err != nil {
		t.Fatalf("error fetching state defender")
	}
	t.Logf("defender bandwidthusage: %f \n", defender.BandwidthUsage)
}

func TestAttack(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	attackerStr, err := setupPlayer(c, ANICK, AEMAIL)
	if err != nil {
		t.Fatalf("setup players error %s \n", err)
	}
	defenderStr, err := setupPlayer(c, BNICK, BEMAIL)
	if err != nil {
		t.Fatalf("setup players error %s \n", err)
	}
	if err := setupPrograms(c); err != nil {
		t.Fatalf("error setup programs: %s \n", err)
	}
	attackerKey, err := datastore.DecodeKey(attackerStr)
	defenderKey, err := datastore.DecodeKey(defenderStr)
	if err != nil {
		t.Fatalf("error decoding key %s \n", err)
	}
	swarmConnKey := datastore.NewKey(c, "Program", SWCONN, 0, nil)
	swarmKey := datastore.NewKey(c, "Program", SWARM, 0, nil)
	hkConnKey := datastore.NewKey(c, "Program", HKCONN, 0, nil)
	hkKey := datastore.NewKey(c, "Program", HUNTERKILLER, 0, nil)
	d0sConnKey := datastore.NewKey(c, "Program", D0SCONN, 0, nil)
	d0sKey := datastore.NewKey(c, "Program", D0S, 0, nil)
	//d0sConnKey := datastore.NewKey(c, "Program", D0SCONN, 0, nil)
	//d0sKey := datastore.NewKey(c, "Program", D0S, 0, nil)
	//mutConnKey := datastore.NewKey(c, "Program", MUTCONN, 0, nil)
	//mutKey := datastore.NewKey(c, "Program", MUTATOR, 0, nil)
	all := player.Allocation{swarmConnKey.Encode(), 1}
	if err := player.Allocate(c, attackerStr, all); err != nil {
		t.Fatalf("allocate attacker connection error %s \n", err)
	}
	all.PrgKey = swarmKey.Encode()
	all.Amount = 2
	if err := player.Allocate(c, attackerStr, all); err != nil {
		t.Fatalf("allocate attacker offensive program error %s \n", err)
	}
	all.PrgKey = hkConnKey.Encode()
	all.Amount = 1
	if err := player.Allocate(c, defenderStr, all); err != nil {
		t.Fatalf("allocate defending program connection error %s \n", err)
	}
	all.PrgKey = hkKey.Encode()
	all.Amount = 5
	if err := player.Allocate(c, defenderStr, all); err != nil {
		t.Fatalf("allocate defending program error %s \n", err)
	}
	all.PrgKey = d0sConnKey.Encode()
	all.Amount = 1
	if err := player.Allocate(c, defenderStr, all); err != nil {
		t.Fatalf("allocate defending program error %s \n", err)
	}
	all.PrgKey = d0sKey.Encode()
	all.Amount = 10
	if err := player.Allocate(c, defenderStr, all); err != nil {
		t.Fatalf("allocate defending program error %s \n", err)
	}
	attackPrograms, err := getAttackPrograms(c, attackerKey)
	if err != nil {
		t.Fatalf("error loading attackprograms %s\n", err)
	}
	defender := new(player.Player)
	if err := datastore.Get(c, defenderKey, defender); err != nil {
		t.Fatalf("errror loading defender \n", err)
	}
	attackCfg := AttackCfg{
		AttackType:     BW,
		Pkey:           attackerKey.Encode(),
		Target:         defender.PlayerID,
		ActivePrograms: attackPrograms,
	}
	testutils.PurgeQueue(c, t)
	attackEvent, err := Attack(c, attackCfg)
	if err != nil {
		t.Fatalf("attack error %s \n", err)
	}
	t.Logf("\n <<< ATTACKEVENT >>> \n%+v\n", attackEvent)
	testutils.CheckQueue(c, t, 1)
}

func TestAttackWithClan(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	attackerStr, err := setupPlayer(c, ANICK, AEMAIL)
	if err != nil {
		t.Fatalf("setup players error %s \n", err)
	}
	aClanStr, errmap, err := clan.Create(c, attackerStr, CLAN1, "lol")
	if err != nil {
		t.Fatalf("\n create clan error %s", err)
	}
	if errmap["clan_name"]+errmap["clan_tag"] > 0 {
		t.Fatalf(" bad clan name or clan tag \n")
	}
	defenderStr, err := setupPlayer(c, BNICK, BEMAIL)
	if err != nil {
		t.Fatalf("setup players error %s \n", err)
	}
	clanStr, errmap, err := clan.Create(c, defenderStr, CLAN2, "lol1")
	if err != nil {
		t.Fatalf("\n create clan error %s", err)
	}
	if errmap["clan_name"]+errmap["clan_tag"] > 0 {
		t.Fatalf(" bad clan name or clan tag \n")
	}
	targetClanKey := datastore.NewKey(c, "Clan", clanStr, 0, nil)
	if err := clan.Connect(c, attackerStr, targetClanKey.Encode()); err != nil {
		t.Fatalf("error connecting to clan: %s", err)
	}
	targetClanKey0 := datastore.NewKey(c, "Clan", aClanStr, 0, nil)
	if err := clan.Connect(c, defenderStr, targetClanKey0.Encode()); err != nil {
		t.Fatalf("error connecting to clan: %s", err)
	}
	if err := setupPrograms(c); err != nil {
		t.Fatalf("error setup programs: %s \n", err)
	}
	attackerKey, err := datastore.DecodeKey(attackerStr)
	defenderKey, err := datastore.DecodeKey(defenderStr)
	if err != nil {
		t.Fatalf("error decoding key %s \n", err)
	}
	swarmConnKey := datastore.NewKey(c, "Program", SWCONN, 0, nil)
	swarmKey := datastore.NewKey(c, "Program", SWARM, 0, nil)
	hkConnKey := datastore.NewKey(c, "Program", HKCONN, 0, nil)
	hkKey := datastore.NewKey(c, "Program", HUNTERKILLER, 0, nil)
	d0sConnKey := datastore.NewKey(c, "Program", D0SCONN, 0, nil)
	d0sKey := datastore.NewKey(c, "Program", D0S, 0, nil)
	//d0sConnKey := datastore.NewKey(c, "Program", D0SCONN, 0, nil)
	//d0sKey := datastore.NewKey(c, "Program", D0S, 0, nil)
	//mutConnKey := datastore.NewKey(c, "Program", MUTCONN, 0, nil)
	//mutKey := datastore.NewKey(c, "Program", MUTATOR, 0, nil)
	all := player.Allocation{swarmConnKey.Encode(), 1}
	if err := player.Allocate(c, attackerStr, all); err != nil {
		t.Fatalf("allocate attacker connection error %s \n", err)
	}
	all.PrgKey = swarmKey.Encode()
	all.Amount = 4
	if err := player.Allocate(c, attackerStr, all); err != nil {
		t.Fatalf("allocate attacker offensive program error %s \n", err)
	}
	all.PrgKey = hkConnKey.Encode()
	all.Amount = 1
	if err := player.Allocate(c, defenderStr, all); err != nil {
		t.Fatalf("allocate defending program connection error %s \n", err)
	}
	all.PrgKey = hkKey.Encode()
	all.Amount = 5
	if err := player.Allocate(c, defenderStr, all); err != nil {
		t.Fatalf("allocate defending program error %s \n", err)
	}
	all.PrgKey = d0sConnKey.Encode()
	all.Amount = 1
	if err := player.Allocate(c, defenderStr, all); err != nil {
		t.Fatalf("allocate defending program error %s \n", err)
	}
	all.PrgKey = d0sKey.Encode()
	all.Amount = 10
	if err := player.Allocate(c, defenderStr, all); err != nil {
		t.Fatalf("allocate defending program error %s \n", err)
	}
	attackPrograms, err := getAttackPrograms(c, attackerKey)
	if err != nil {
		t.Fatalf("error loading attackprograms %s\n", err)
	}
	defender := new(player.Player)
	if err := datastore.Get(c, defenderKey, defender); err != nil {
		t.Fatalf("errror loading defender \n", err)
	}
	attackCfg := AttackCfg{
		AttackType:     BW,
		Pkey:           attackerKey.Encode(),
		Target:         defender.PlayerID,
		ActivePrograms: attackPrograms,
	}
	testutils.PurgeQueue(c, t)
	attackEvent, err := Attack(c, attackCfg)
	if err != nil {
		t.Fatalf("attack error %s \n", err)
	}
	t.Logf("\n <<< ATTACKEVENT >>> \n%+v\n", attackEvent)
	testutils.CheckQueue(c, t, 1)
}
