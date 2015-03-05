package attack

import (
	"appengine"
	"appengine/datastore"
	"errors"
	"fmt"
	"math"
	"netwars/clan"
	"netwars/event"
	"netwars/program"
	"netwars/user"
	"netwars/utils"
	"strconv"
	"time"
)

const (
	BAL               = program.MUT | program.HUK | program.D0S | program.SW
	MEM               = program.MUT | program.HUK
	BW                = program.D0S | program.SW
	INT               = program.INT
	ICE               = program.ICE
	SIW               = 2
	MUW               = 1
	ATTACK_EVENT_PATH = "/events/attack"
)

var AttackName = map[int64]string{
	BAL: "Balanced",
	MEM: "Memory",
	BW:  "Bandwidth",
	INT: "Intelligence",
	ICE: "Ice",
}

var AttackType = map[string]int64{
	"Balanced":     BAL,
	"Memory":       MEM,
	"Bandwidth":    BW,
	"Intelligence": INT,
	"Ice":          ICE,
}

var (
	TypeBonus      float64 = 0.2
	OffensiveTypes         = []int64{program.MUT, program.HUK, program.D0S, program.SW}
)

type ActiveProgram struct {
	Key    string `json:"key"`
	Amount int64  `json:"amount"`
}

type AttackCfg struct {
	AttackType     string          `json:"attack_type"`
	Pkey           string          `json:"pkey"`
	Target         string          `json:"target"`
	ActivePrograms []ActiveProgram `json:"attack_programs"`
}

type AttackFrame struct {
	Window    *AttackWindow
	Receiving []*AttackEventProgram
	Dealing   []*AttackEventProgram
}

type AttackWindow struct {
	AttackEvent  *AttackEvent
	DefenseEvent *AttackEvent
	BattleMap    map[int64]*AttackFrame
}

type AttackEvent struct {
	Result          bool    `json:"result"`
	Memory          int64   `json:"mem_cost" datastore:",noindex"`
	BwLost          float64 `json:"bw_lost"`
	ProgramsLost    int64   `json:"programs_lost"`
	BwKilled        float64 `json:"bw_killed"`
	YieldLost       int64   `json:"yield_lost"`
	ProgramsKilled  int64   `json:"programs_killed"`
	ApsGained       int64   `json:"aps_gained" datastore:",noindex`
	CpsGained       int64   `json:"cps_gained" datastore:",noindex`
	CyclesGained    int64   `json:"cycles_gained"`
	Cycles          int64   `json:"cycles_lost"`
	VDamageReceived int64   `datastore:",noindex" json:"-"`
}

type AttackEventProgram struct {
	Amount           float64             `json: "amount"`
	event            *AttackEvent        `json:"-" datastore:"-" gob:"-"`
	AmountUsed       int64               `json:"amount_used" datastore:",noindex`
	AmountBefore     int64               `json:"amount_before" datastore:",noindex`
	AmountLost       []int64             `json:"amount_after" datastore:",noindex`
	Program          *datastore.Key      `json:"program" datastore:",noindex`
	ProgramActive    bool                `json:"program_active" datastore:",noindex`
	BwLost           float64             `json:"bw_lost" datastore:",noindex`
	PlayerProgram    *user.PlayerProgram `datastore:"-" json:"-"`
	ActiveDefender   bool                `json:"-" datastore:",noindex`
	AttackEfficiency float64             `json:"-" datastore:"-" datastore:",noindex`
	YieldLost        int64               `json:"yield_lost" datastore:",noindex`
	Power            bool                `datastore:",noindex" json:"power"`
	VDamageReceived  int64               `datastore:",noindex" json:"-"`
}

func (eprog *AttackEventProgram) AttackDamage() float64 {
	fmt.Printf(" << AttackDamage >>\n")
	//	fmt.Printf(" << Amount used : %d >>\n", eprog.AmountUsed)
	//	fmt.Printf(" << Program's attack: %d >>\n", eprog.PlayerProgram.Attack)
	//	fmt.Printf(" << Efficiency %f >>\n", eprog.AttackEfficiency)
	//	fmt.Printf(" << Damage dealt %f >>\n", float64(eprog.AmountUsed)*float64(eprog.PlayerProgram.Attack)*eprog.AttackEfficiency)
	return float64(eprog.AmountUsed) * float64(eprog.PlayerProgram.Attack) * eprog.AttackEfficiency
}

func (eprog *AttackEventProgram) ReceiveDamage(attackDamage float64, attackEvent *AttackEvent) {
	var attackFactor float64 = 0.8
	if !eprog.ActiveDefender {
		attackFactor = 0.4
	} else if attackEvent.AttackType != BAL {
		attackFactor = 1.0
	}
	attackDamage = attackDamage * attackFactor
	intDamage := int64(attackDamage)
	killedPrograms := (intDamage - (intDamage % eprog.PlayerProgram.Life)) / eprog.PlayerProgram.Life
	programsLeft := eprog.AmountBefore
	for _, amk := range eprog.AmountLost {
		programsLeft -= amk
	}
	if killedPrograms > programsLeft {
		killedPrograms = programsLeft
	}
	eprog.VDamageReceived += intDamage
	eprog.event.VDamageReceived += intDamage
	eprog.PlayerProgram.Amount -= killedPrograms
	eprog.AmountLost = append(eprog.AmountLost, killedPrograms)
	eprog.Amount += float64(killedPrograms)
	eprog.BwLost += float64(killedPrograms) * eprog.PlayerProgram.BandwidthUsage
	fmt.Printf("bwlost: %.2f, killed programs: %d, receiver name: %s, usage: %.2f \n", eprog.BwLost, killedPrograms, eprog.PlayerProgram.Name, eprog.PlayerProgram.BandwidthUsage)
	eprog.PlayerProgram.Usage -= eprog.BwLost
	eprog.event.ProgramsLost += killedPrograms
	eprog.event.BwLost += eprog.BwLost
	attackEvent.BwKilled += eprog.BwLost
	attackEvent.ProgramsKilled += killedPrograms
	if eprog.PlayerProgram.Bandwidth > 0 {
		eprog.YieldLost += killedPrograms * eprog.PlayerProgram.Bandwidth
		eprog.event.YieldLost += eprog.YieldLost
		eprog.PlayerProgram.Yield -= eprog.YieldLost
	}
}

func (window *AttackWindow) AddReceiver(atype int64, a *AttackEventProgram) {
	if _, ok := window.BattleMap[atype]; ok {
		if window.BattleMap[atype].AddReceiver(a) {
			window.DefenseEvent.EventPrograms = append(window.DefenseEvent.EventPrograms, a)
		}
	} else {
		frame := new(AttackFrame)
		if frame.AddReceiver(a) {
			window.DefenseEvent.EventPrograms = append(window.DefenseEvent.EventPrograms, a)
		}
		frame.Window = window
		window.BattleMap[atype] = frame
	}
}

func (window *AttackWindow) AddDealer(atype int64, a *AttackEventProgram) {
	if _, ok := window.BattleMap[atype]; ok {
		window.BattleMap[atype].AddDealer(a)
	} else {
		frame := new(AttackFrame)
		frame.AddDealer(a)
		frame.Window = window
		window.BattleMap[atype] = frame
	}
}

func (window *AttackWindow) Render() {
	for _, frame := range window.BattleMap {
		frame.Render()
	}
}

func (frame *AttackFrame) AddDealer(a *AttackEventProgram) {
	frame.Dealing = append(frame.Dealing, a)
}

func (frame *AttackFrame) AddReceiver(a *AttackEventProgram) bool {
	for _, rec := range frame.Receiving {
		if rec.PlayerProgram.Key.Equal(a.PlayerProgram.Key) {
			return false
		}
	}
	frame.Receiving = append(frame.Receiving, a)
	return true
}

func (frame *AttackFrame) Render() {
	receiverCount := len(frame.Receiving)
	var attackDamage float64
	for _, dealer := range frame.Dealing {
		attackDamage += dealer.AttackDamage()
	}
	attackDamage = attackDamage / float64(receiverCount)
	for _, receiver := range frame.Receiving {
		receiver.ReceiveDamage(attackDamage, frame.Window.AttackEvent)
	}
}

func isValidAttack(attacker *user.Player, defender *user.Player) {
	//	errors := make(map[string]int, 4)

}

func getConnection(c appengine.Context, aMember, bMember *datastore.Key) ([]*datastore.Key, error) {
	conns := make([]clan.ClanConnection, 0, 1)
	q := datastore.NewQuery("ClanConnection").Ancestor(aMember.Parent()).
		Filter("Active =", true).Filter("Target =", bMember.Parent()).KeysOnly().Limit(1)
	connKeys, connErr := q.GetAll(c, &conns)
	if connErr != nil {
		return nil, connErr
	}
	return connKeys, nil
}

func loadWar(c appengine.Context, aMember, dMember *datastore.Key, connKeys []*datastore.Key, warCh chan<- int) {
	connCh := make(chan int, 2)
	var aConnKeys []*datastore.Key
	var dConnKeys []*datastore.Key
	var err error
	go func() {
		aConnKeys, err = getConnection(c, aMember, dMember)
		if err != nil {
			c.Errorf("error getting connection %s \n", err)
		}
		connCh <- 0
	}()
	go func() {
		dConnKeys, err = getConnection(c, dMember, aMember)
		if err != nil {
			c.Errorf("error getting defending clan %s \n", err)
		}
		connCh <- 0
	}()
	for i := 0; i < 2; i++ {
		<-connCh
	}
	var war int
	if len(aConnKeys) > 0 {
		connKeys[0] = aConnKeys[0]
		if len(dConnKeys) > 0 {
			connKeys[1] = dConnKeys[0]
			war = MUW
		} else {
			war = SIW
		}
	}
	warCh <- war
}

func renderAttack(cfg AttackCfg, attackerState, defenderState *user.PlayerState, attack, defense *AttackWindow) error {
	attackType := AttackType[cfg.AttackType]
	for _, attackProgram := range cfg.ActivePrograms {
		attackProgramKey, err := datastore.DecodeKey(attackProgram.Key)
		if err != nil {
			return err
		}
		for _, offensiveType := range OffensiveTypes {
			if aGroupForType, ok := attackerState.Programs[offensiveType]; ok {
				for _, aProg := range aGroupForType.Programs {
					if aProg.ProgramKey.Equal(attackProgramKey) {
						if !aGroupForType.Power {
							return errors.New("Can't use attack program without power")
						}
						aeProgram := &AttackEventProgram{
							event:          attack.AttackEvent,
							Name:           aProg.Name,
							AmountBefore:   aProg.Amount,
							AmountUsed:     attackProgram.Amount,
							ProgramActive:  aProg.Active,
							PlayerProgram:  aProg,
							ActiveDefender: true,
							Power:          aGroupForType.Power,
							Owned:          true,
						}
						for defenseType := range defenderState.Programs {
							if dGroupForType, ok := defenderState.Programs[defenseType]; ok {
								for _, dProg := range dGroupForType.Programs {
									activeDefender := true
									if dProg.EffectorTypes&attackType == 0 {
										activeDefender = false
									}
									daeProgram := &AttackEventProgram{
										event:          defense.AttackEvent,
										Name:           dProg.Name,
										AmountBefore:   dProg.Amount,
										AmountUsed:     dProg.Amount,
										ProgramActive:  dProg.Active,
										PlayerProgram:  dProg,
										ActiveDefender: activeDefender,
										Power:          dGroupForType.Power,
										Owned:          true,
									}
									if defenseType == aProg.EffectorTypes&defenseType {
										aeProgram.AttackEfficiency = 1.0 / float64(len(aProg.Effectors))
										attack.AddDealer(defenseType, aeProgram)
										attack.AddReceiver(defenseType, daeProgram)
									}
									if aProg.Type == dProg.EffectorTypes&aProg.Type && daeProgram.Power && dProg.Active {
										daeProgram.AttackEfficiency = 1.0 / float64(len(dProg.Effectors))
										defense.AddDealer(aProg.Type, daeProgram)
										defense.AddReceiver(aProg.Type, aeProgram)
									}
								}
							} else {
								return errors.New("Fatal error")

							}

						}

					}
				}
			} else {
				continue
			}
		}

	}
	defense.Render()
	attack.Render()
	return nil
}

func Attack(c appengine.Context, cfg AttackCfg) (AttackEvent, error) {
	c.Debugf("running attack  cfg: %+v<<<\n", cfg)
	akey, err := datastore.DecodeKey(cfg.Pkey)
	defenderID, err := strconv.ParseInt(cfg.Target, 10, 64)
	if err != nil {
		return event.Event{}, err
	}
	defenderKey, err := user.KeyByID(c, defenderID)
	if err != nil {
		return event.Event{}, err
	}
	var response AttackEvent
	options := new(datastore.TransactionOptions)
	options.XG = true
	txErr := datastore.RunInTransaction(c, func(c appengine.Context) error {
		attacker := new(user.Player)
		defender := new(user.Player)
		playerStCh := make(chan int, 1)
		go func() {
			if cleanUpAKeys, err := user.Status(c, defenderKey.Encode(), defenderState); err != nil {
				c.Errorf("get player status error %s \n", err)
			}
			playerStCh <- 0
		}()
		if cleanUpDKeys, err := user.Status(c, cfg.Pkey, attackerState); err != nil {
			return err
		}
		<-playerStCh
		cleanUpKeys := append(cleanUpAKeys, cleanUpDKeys...)
		cleanUp := len(cleanUpKeys)
		cleanUpCh := make(chan int)
		if cleanUp {
			go func() {
				if err := datastore.DeleteMulti(c, cleanUpKeys); err != nil {
					c.Errorf("error cleanup while allocating")
				}
				cleanUpCh <- 0
			}()
		}
		warCh := make(chan int, 1)
		connKeys := make([]*datastore.Key, 2)
		if attackerState.Player.ClanMember != nil && defenderState.Player.ClanMember != nil {
			if attackerState.Player.ClanMember.Parent().Equal(defenderState.Player.ClanMember.Parent()) {
				return errors.New("Can't attack your own team members")
			}
			loadWar(c, attackerState.Player.ClanMember, defenderState.Player.ClanMember, connKeys, warCh)
		} else {
			warCh <- 0
		}
		now := time.Now()
		defenseEvent := &event.Event{
			Created:    now,
			Player:     defenderKey,
			Direction:  utils.IN,
			EventType:  "Attack",
			Target:     akey,
			AttackType: AttackType[cfg.AttackType],
			PlayerName: attackerState.Player.Nick,
			PlayerID:   attackerState.Player.PlayerID,
			TargetName: defenderState.Player.Nick,
			TargetID:   defenderState.PlayerID,
		}
		if defenderState.Player.ClanMember != nil {
			defenseEvent.ClanMember = defenderState.Player.ClanMember
		}
		attackEvent := &event.Event{
			Created:    now,
			Player:     akey,
			Direction:  utils.OUT,
			EventType:  "Attack",
			Target:     defenderKey,
			AttackType: AttackType[cfg.AttackType],
			PlayerName: defenderState.Player.Nick,
			PlayerID:   defenderState.Player.PlayerID,
			TargetName: attackerState.Player.Nick,
			TargetID:   attackerProfile.PlayerID,
		}
		if attackerState.Player.ClanMember != nil {
			attackEvent.ClanMember = attackerState.Player.ClanMember
		}
		attack := &AttackWindow{
			AttackEvent:  attackEvent,
			DefenseEvent: defenseEvent,
			BattleMap:    make(map[int64]*AttackFrame),
		}
		defense := &AttackWindow{
			AttackEvent:  defenseEvent,
			DefenseEvent: attackEvent,
			BattleMap:    make(map[int64]*AttackFrame),
		}
		//TODO check attacker status
		if err := renderAttack(cfg, attackerState, defenderState, attack, defense); err != nil {
			return err
		}
		if attackerState.Player.BandwidthUsage < defenderState.Player.BandwidthUsage {
			attackEvent.Memory = 2
		} else {
			attackEvent.Memory = 3
		}
		diffLoss := defenseEvent.BwLost - attackEvent.BwLost
		vicCon := attackEvent.BwLost * 0.1
		war := <-warCh
		attackEvent.ClanConnection = connKeys[0]
		defenseEvent.ClanConnection = connKeys[1]
		if diffLoss <= 0 || diffLoss <= vicCon {
			attackEvent.Result, defenseEvent.Result = false, true
			defenseEvent.ApsGained = 1
		} else {
			attackEvent.Result, defenseEvent.Result = true, false
			attackEvent.ApsGained = 1
			if war > 0 {
				pct := (attackEvent.BwKilled / defenderState.Player.BandwidthUsage) * 100
				c.Debugf("pct killed : %.2f \n", pct)
				if pct > 10.0 {
					pct = 10
				}
				hardpts := (math.Sqrt(attackEvent.BwKilled) + 200) / 200
				c.Debugf("hardpts: %.2f \n", hardpts)
				if hardpts > 10.0 {
					hardpts = 10
				}
				cps := ((pct / 2) + (hardpts / 2)) / float64(war)
				attackEvent.CpsGained = int64(cps)
				attackEvent.ApsGained = int64(2 / war)
			}
			transferCycles := int64(float64(defenderState.Player.Cycles) * 0.1)
			attackEvent.CyclesGained = transferCycles
			defenseEvent.Cycles = transferCycles
		}
		var keys []*datastore.Key
		var models []interface{}
		for _, aep := range attackEvent.EventPrograms {
			if aep.Amount > 0 {
				aep.Owned = false
				defenseEvent.EventPrograms = append(defenseEvent.EventPrograms, aep)
			}
			keys = append(keys, aep.PlayerProgram.Key)
			models = append(models, aep.PlayerProgram)
		}
		for _, daep := range defenseEvent.EventPrograms {
			if daep.Amount > 0 {
				daep.Owned = false
				attackEvent.EventPrograms = append(attackEvent.EventPrograms, daep)
			}
			keys = append(keys, daep.PlayerProgram.Key)
			models = append(models, daep.PlayerProgram)
		}
		attackerState.Player.Cycles += attackEvent.CyclesGained
		defenderState.Player.Cycles -= defenseEvent.Cycles
		attackerState.Player.Cps += attackEvent.CpsGained
		attackerState.Player.Aps += attackEvent.ApsGained
		defenderState.Player.Aps += defenseEvent.ApsGained
		defenderState.Player.NewLocals++
		attackerState.Player.Memory -= attackEvent.Memory
		attackerState.Player.ActiveMemory -= attackEvent.Memory
		keys = append(keys, akey, defenderKey)
		models = append(models, attackerState.Player, defenderState.Player)
		if _, err := datastore.PutMulti(c, keys, models); err != nil {
			return err
		}
		msg := event.Message([]*event.Event{attackEvent, defenseEvent})
		if err := msg.Send(c, AttackEvent); err != nil {
			return err
		}
		<-cleanUpCh
		response = attackEvent
		return nil
	}, options)
	if txErr != nil {
		c.Debugf("error %s \n", txErr)
		return event.Event{}, txErr
	}
	return response, nil
}

func AttackEvent(c appengine.Context, events event.Message) error {
	attackEvent := events[0]
	defenseEvent := events[1]
	attackerMember := new(clan.ClanMember)
	defenderMember := new(clan.ClanMember)
	attackerConnection := new(clan.ClanConnection)
	defenderConnection := new(clan.ClanConnection)
	aClanKey := new(datastore.Key)
	dClanKey := new(datastore.Key)
	var war int
	var notifyCnt int = 1
	keys := make([]*datastore.Key, 0)
	models := make([]interface{}, 0)
	if attackEvent.ClanMember != nil {
		keys = append(keys, attackEvent.ClanMember)
		attackerMember = new(clan.ClanMember)
		models = append(models, attackerMember)
		aClanKey = attackEvent.ClanMember.Parent()
		notifyCnt++
	}
	if defenseEvent.ClanMember != nil {
		keys = append(keys, defenseEvent.ClanMember)
		defenderMember = new(clan.ClanMember)
		models = append(models, defenderMember)
		dClanKey = defenseEvent.ClanMember.Parent()
		notifyCnt++
	}
	if attackerMember != nil && defenderMember != nil {
		if attackEvent.ClanConnection != nil {
			keys = append(keys, attackEvent.ClanConnection)
			attackerConnection = new(clan.ClanConnection)
			models = append(models, attackerConnection)
			war++
		}
		if defenseEvent.ClanConnection != nil {
			keys = append(keys, defenseEvent.ClanConnection)
			defenderConnection = new(clan.ClanConnection)
			models = append(models, defenderConnection)
			war++
		}
	}
	cntCh := make(chan int64, 1)
	notifyCh := make(chan int, notifyCnt)
	go NewEventID(c, cntCh)
	if err := datastore.GetMulti(c, keys, models); err != nil {
		return err
	}
	eventID := <-cntCh
	attackEvent.ID = eventID
	defenseEvent.ID = eventID
	if attackerMember != nil {
		attackerMember.BwKilled += attackEvent.BwKilled
		attackerMember.BwLost += attackEvent.BwLost
		attackerMember.AttacksMade++
		if attackerConnection != nil {
			attackerMember.AttacksMadeWC++
			if defenderMember != nil {
				defenderMember.AttacksSufferedWC++
			}
			updateClanConnection(war, attackEvent, attackerConnection)
		}

		go attackEvent.NotifyClan(c, attackEvent.Player, notifyCh)
	}
	if defenderMember != nil {
		defenderMember.BwKilled += defenseEvent.BwKilled
		defenderMember.BwLost += defenseEvent.BwLost
		defenderMember.AttacksSuffered++
		if defenderConnection != nil {
			updateClanConnection(war, defenseEvent, defenderConnection)
		}
		go defenseEvent.NotifyClan(c, defenseEvent.Player, notifyCh)
	}
	//go NotifyPlayer(c, attackerProfile, &receiverEvent, notifyCh)
	go defenseEvent.NotifyPlayer(c, notifyCh)
	aEventGuid, err := guid.GenUUID()
	dEventGuid, err := guid.GenUUID()
	if err != nil {
		return err
	}
	dEventKey := datastore.NewKey(c, "Event", dEventGuid, 0, nil)
	aEventKey := datastore.NewKey(c, "Event", aEventGuid, 0, nil)
	models = append(models, attackEvent, defenseEvent)
	keys = append(keys, aEventKey, dEventKey)
	if _, err := datastore.PutMulti(c, keys, models); err != nil {
		return err
	}
	for j := 0; j < notifyCnt; j++ {
		<-notifyCh
	}
	return nil
}

func updateClanConnection(war int, event *Event, conn *clan.ClanConnection) {
	switch event.Direction {
	case utils.IN:
		conn.InEvents++
		switch war {
		case attack.SIW:
			conn.SIWInAttacks++
		case attack.MUW:
			conn.MUWInAttacks++
		}
	case utils.OUT:
		conn.OutEvents++
		switch war {
		case attack.SIW:
			conn.SIWOutAttacks++
		case attack.MUW:
			conn.MUWOutAttacks++
		}

	}
}
