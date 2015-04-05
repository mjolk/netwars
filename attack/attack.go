package attack

import (
	"appengine"
	"appengine/datastore"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"mj0lk.be/netwars/clan"
	"mj0lk.be/netwars/event"
	"mj0lk.be/netwars/player"
	"mj0lk.be/netwars/program"
	"time"
)

const (
	BAL = program.MUT | program.HUK | program.D0S | program.SW
	MEM = program.MUT | program.HUK
	BW  = program.D0S | program.SW
	INT = program.INT
	ICE = program.ICE
	INF = program.INF
	SIW = 1
	MUW = 2
)

var (
	TypeBonus      float64 = 0.2
	OffensiveTypes         = []int64{program.MUT, program.HUK, program.D0S, program.SW}
	AttackType             = map[string]int64{
		"Balanced":     BAL,
		"Memory":       MEM,
		"Bandwidth":    BW,
		"Intelligence": INT,
		"Ice":          ICE,
	}

	AttackName = map[int64]string{
		BAL: "Balanced",
		MEM: "Memory",
		BW:  "Bandwidth",
		INT: "Intelligence",
		ICE: "Ice",
	}
)

type ActiveProgram struct {
	Key    string `json:"key"`
	Amount int64  `json:"amount"`
}

type AttackCfg struct {
	AttackType     int64           `json:"attack_type"`
	Target         int64           `json:"target"`
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
	Updated      []*AttackEventProgram
	UpdatedKeys  []*datastore.Key
	ToUpdate     []interface{}
}

type AttackEvent struct {
	AttackType int64 `json:"attack_type"`
	*event.Event
	Connection *datastore.Key `json:"-"`
}

type AttackEventProgram struct {
	PlayerProgram *player.PlayerProgram `datastore:"-" json:"-"`
	*event.EventProgram
}

func (eprog *AttackEventProgram) AttackDamage() float64 {
	fmt.Printf(" << AttackDamage >>\n")
	fmt.Printf(" << Amount used : %d >>\n", eprog.AmountUsed)
	fmt.Printf(" << Program's attack: %d >>\n", eprog.PlayerProgram.Attack)
	fmt.Printf(" << Efficiency %f >>\n", eprog.AttackEfficiency)
	fmt.Printf(" << Damage dealt %f >>\n", float64(eprog.AmountUsed)*float64(eprog.PlayerProgram.Attack)*eprog.AttackEfficiency)
	return float64(eprog.AmountUsed) * float64(eprog.PlayerProgram.Attack) * eprog.AttackEfficiency
}

func (eprog *AttackEventProgram) ReceiveDamage(window *AttackWindow, attackDamage float64) {
	var attackFactor float64 = 0.8
	if !eprog.ActiveDefender {
		attackFactor = 0.4
	} else if window.AttackEvent.AttackType != BAL {
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
	window.DefenseEvent.VDamageReceived += intDamage
	eprog.AmountLost = append(eprog.AmountLost, killedPrograms)
	eprog.Amount += killedPrograms
	eprog.BwLost += float64(killedPrograms) * eprog.PlayerProgram.BandwidthUsage
	fmt.Printf("bwlost: %.2f, killed programs: %d, receiver name: %s, usage: %.2f \n",
		eprog.BwLost, killedPrograms, eprog.PlayerProgram.Name, eprog.PlayerProgram.BandwidthUsage)
	window.DefenseEvent.ProgramsLost += killedPrograms
	window.DefenseEvent.BwLost += eprog.BwLost
	window.AttackEvent.BwKilled += eprog.BwLost
	window.AttackEvent.ProgramsKilled += killedPrograms
	if eprog.PlayerProgram.Bandwidth > 0 {
		eprog.YieldLost += killedPrograms * eprog.PlayerProgram.Bandwidth
		window.DefenseEvent.YieldLost += eprog.YieldLost
	}
	eprog.PlayerProgram.Amount -= killedPrograms
	//TODO  calculate experience here?

}

func (window *AttackWindow) AddReceiver(atype int64, a *AttackEventProgram) {
	if _, ok := window.BattleMap[atype]; ok {
		window.BattleMap[atype].AddReceiver(a)
	} else {
		frame := new(AttackFrame)
		frame.AddReceiver(a)
		frame.Window = window
		window.BattleMap[atype] = frame
	}
	if len(window.UpdatedKeys) > 0 {
		for _, key := range window.UpdatedKeys {
			if key.Equal(a.PlayerProgram.DbKey) {
				return
			}
		}
	}
	window.UpdatedKeys = append(window.UpdatedKeys, a.PlayerProgram.DbKey)
	window.Updated = append(window.Updated, a)
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
	if len(window.Updated) > 0 {
		for i, update := range window.Updated {
			if update.Amount > 0 { // remember Amount is in this case total amount lost programs
				window.ToUpdate = append(window.ToUpdate, update.PlayerProgram)
				window.DefenseEvent.EventPrograms = append(window.DefenseEvent.EventPrograms, *update.EventProgram)
				ae := event.EventProgram{}
				ae = *update.EventProgram
				ae.Owned = false
				window.AttackEvent.EventPrograms = append(window.AttackEvent.EventPrograms, ae)

			} else {
				window.UpdatedKeys = append(window.UpdatedKeys[:i], window.UpdatedKeys[i+1:]...)
			}

		}
	}
}

func (frame *AttackFrame) AddDealer(a *AttackEventProgram) {
	frame.Dealing = append(frame.Dealing, a)
}

func (frame *AttackFrame) AddReceiver(a *AttackEventProgram) {
	for _, rec := range frame.Receiving {
		if rec.PlayerProgram.DbKey.Equal(a.PlayerProgram.DbKey) {
			return
		}
	}
	frame.Receiving = append(frame.Receiving, a)
}

func (frame *AttackFrame) Render() {
	receiverCount := len(frame.Receiving)
	var attackDamage float64
	for _, dealer := range frame.Dealing {
		attackDamage += dealer.AttackDamage()
	}
	attackDamage = attackDamage / float64(receiverCount)
	for _, receiver := range frame.Receiving {
		receiver.ReceiveDamage(frame.Window, attackDamage)
	}
}

func isValidAttack(attacker, defender *player.Player) {
	//	errors := make(map[string]int, 4)

}

func getConnection(c appengine.Context, aKey, bKey *datastore.Key) ([]*datastore.Key, error) {
	q := datastore.NewQuery("ClanConnection").Ancestor(aKey).
		Filter("Active =", true).Filter("Target =", bKey).KeysOnly().Limit(1)
	conns := make([]*clan.ClanConnection, 1, 1)
	connKeys, connErr := q.GetAll(c, &conns)
	if connErr != nil {
		return nil, connErr
	}
	return connKeys, nil
}

func loadWar(c appengine.Context, doneCh chan<- int, attacker, defender *AttackEvent) {
	attackConn := make(chan *datastore.Key)
	defenseConn := make(chan *datastore.Key)
	var war int
	go func() {
		aConnKeys, err := getConnection(c, attacker.Clan, defender.Clan)
		if err != nil {
			c.Errorf("error getting connection %s \n", err)
		}
		if len(aConnKeys) > 0 {
			attackConn <- aConnKeys[0]
		}
	}()
	go func() {
		dConnKeys, err := getConnection(c, defender.Clan, attacker.Clan)
		if err != nil {
			c.Errorf("error getting defending clan %s \n", err)
		}
		if len(dConnKeys) > 0 {
			defenseConn <- dConnKeys[0]
		}
	}()
	for i := 0; i < 2; i++ {
		select {
		case attacker.Connection = <-attackConn:
		case defender.Connection = <-defenseConn:
		}
	}
	doneCh <- war
}

func render(cfg AttackCfg, attacker, defender *player.Player, attack, defense *AttackWindow) error {
	for _, attackProgram := range cfg.ActivePrograms {
		attackProgramKey, err := datastore.DecodeKey(attackProgram.Key)
		if err != nil {
			return err
		}
		for _, offensiveType := range OffensiveTypes {
			if aGroupForType, ok := attacker.Programs[offensiveType]; ok {
				for _, aProg := range aGroupForType.Programs {
					if aProg.ProgramKey.Equal(attackProgramKey) {
						if !aGroupForType.Power {
							return errors.New("Can't use attack program without power")
						}
						if attackProgram.Amount > aProg.Amount {
							return errors.New("Not enough programs for attack")
						}
						if !aProg.Active {
							//return error??
							continue
						}
						aeProgram := &AttackEventProgram{
							aProg,
							&event.EventProgram{
								Name:           aProg.Name,
								AmountBefore:   aProg.Amount,
								AmountUsed:     attackProgram.Amount,
								ProgramActive:  aProg.Active,
								ActiveDefender: true,
								Power:          aGroupForType.Power,
								Owned:          true},
						}
						for defenseType := range defender.Programs {
							if dGroupForType, ok := defender.Programs[defenseType]; ok {
								for _, dProg := range dGroupForType.Programs {
									activeDefender := true
									if dProg.EffectorTypes&cfg.AttackType == 0 {
										activeDefender = false
									}
									daeProgram := &AttackEventProgram{
										dProg,
										&event.EventProgram{
											Name:           dProg.Name,
											AmountBefore:   dProg.Amount,
											AmountUsed:     dProg.Amount,
											ProgramActive:  dProg.Active,
											ActiveDefender: activeDefender,
											Power:          dGroupForType.Power,
											Owned:          true},
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

func NewAttackEvent(t int64, dir int64, player, target *player.Player) *AttackEvent {
	now := time.Now()
	ievent := &AttackEvent{
		t,
		&event.Event{
			Created:    now,
			Player:     player.DbKey,
			Clan:       player.ClanKey,
			ClanName:   target.Clan,
			Direction:  dir,
			EventType:  "Attack",
			Target:     target.DbKey,
			Action:     AttackName[t],
			PlayerName: player.Nick,
			PlayerID:   player.PlayerID,
			TargetName: target.Nick,
			TargetID:   target.PlayerID},
		nil,
	}
	return ievent
}

func Attack(c appengine.Context, playerStr string, cfg AttackCfg) (AttackEvent, error) {
	c.Debugf("running attack  cfg: %+v<<<\n", cfg)
	attackerKey, err := datastore.DecodeKey(playerStr)
	if err != nil {
		return AttackEvent{}, err
	}
	defenderKey, err := player.KeyByID(c, cfg.Target)
	if err != nil {
		return AttackEvent{}, err
	}
	var response AttackEvent
	options := new(datastore.TransactionOptions)
	options.XG = true
	txErr := datastore.RunInTransaction(c, func(c appengine.Context) error {
		attacker := new(player.Player)
		defender := new(player.Player)
		playerStCh := make(chan int, 1)
		go func() {
			if err := player.Status(c, defenderKey.Encode(), defender); err != nil {
				c.Errorf("get player status error %s \n", err)
			}
			playerStCh <- 0
		}()
		if err := player.Status(c, playerStr, attacker); err != nil {
			return err
		}
		<-playerStCh
		attackEvent := NewAttackEvent(cfg.AttackType, event.OUT, attacker, defender)
		defenseEvent := NewAttackEvent(cfg.AttackType, event.IN, defender, attacker)
		warCh := make(chan int, 1)
		if attacker.ClanKey != nil && defender.ClanKey != nil {
			if attacker.ClanKey.Equal(defender.ClanKey) {
				return errors.New("Can't attack your own team members")
			}
			go loadWar(c, warCh, attackEvent, defenseEvent)
		} else {
			warCh <- 0
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
		if attacker.BandwidthUsage < defender.BandwidthUsage {
			attackEvent.Memory = 2
		} else {
			attackEvent.Memory = 3
		}
		//TODO check attacker status
		if err := render(cfg, attacker, defender, attack, defense); err != nil {
			return err
		}
		diffLoss := defenseEvent.BwLost - attackEvent.BwLost
		vicCon := attackEvent.BwLost * 0.1
		war := <-warCh
		if diffLoss <= 0 || diffLoss <= vicCon {
			attackEvent.Result, defenseEvent.Result = false, true
			defenseEvent.ApsGained = 1
		} else {
			attackEvent.Result, defenseEvent.Result = true, false
			attackEvent.ApsGained = 1
			if war > 0 {
				pct := (attackEvent.BwKilled / defender.BandwidthUsage) * 100
				c.Debugf("pct killed : %.2f \n", pct)
				if pct > 10.0 {
					pct = 10
				}
				hardpts := (math.Sqrt(attackEvent.BwKilled) + 200) / 200
				c.Debugf("hardpts: %.2f \n", hardpts)
				if hardpts > 10.0 {
					hardpts = 10
				}
				cps := ((pct + hardpts) * 0.5) * float64(war)
				attackEvent.CpsGained = int64(cps)
				attackEvent.ApsGained = int64(war)
			}
			transferCycles := int64(float64(defender.Cycles) * 0.1)
			attackEvent.CyclesGained = transferCycles
			defenseEvent.Cycles = transferCycles
		}
		attacker.Cycles += attackEvent.CyclesGained
		defender.Cycles -= defenseEvent.Cycles
		attacker.Cps += attackEvent.CpsGained
		attacker.Aps += attackEvent.ApsGained
		defender.Aps += defenseEvent.ApsGained
		attacker.Memory -= attackEvent.Memory
		attacker.ActiveMemory -= attackEvent.Memory
		keys := []*datastore.Key{attackerKey, defenderKey}
		models := []interface{}{attacker, defender}
		keys = append(keys, append(attack.UpdatedKeys, defense.UpdatedKeys...)...)
		models = append(models, append(attack.ToUpdate, defense.ToUpdate...)...)
		if _, err := datastore.PutMulti(c, keys, models); err != nil {
			return err
		}
		attackEvent.NewBandwidthUsage = attacker.BandwidthUsage - attackEvent.BwLost
		defenseEvent.NewBandwidthUsage = defender.BandwidthUsage - defenseEvent.BwLost
		if err := event.Send(c, []*event.Event{attackEvent.Event, defenseEvent.Event}, event.Func); err != nil {
			return err
		}
		response = *attackEvent
		return nil
	}, options)
	if txErr != nil {
		c.Debugf("error %s \n", txErr)
		return AttackEvent{}, txErr
	}
	return response, nil
}

type ProbResult struct {
	Visual  bool
	Success bool
	Killed  bool
}

func buildProbability(actPct, actVpct float64) []ProbResult {
	r := int(math.Ceil(actPct / 10))
	v := int(math.Ceil(actVpct / 10))
	rv := make([]ProbResult, 10, 10)
	var c int
	for c = 0; c < 10; c++ {
		pResult := ProbResult{false, false, false}
		if c < r {
			pResult.Success = true
		}
		if c < v {
			pResult.Visual = true
		}
		if c < r-5 {
			pResult.Killed = true
		}
		rv[c] = pResult
	}
	return rv
}

func renderProb(attackProgram *AttackEventProgram, defender *player.Player) (ProbResult, error) {
	pDef := []int64{program.FW, program.INF}
	var pctDefense float64
	for _, def := range pDef {
		if dGroupForType, ok := defender.Programs[def]; ok {
			if dGroupForType.Power {
				for _, defProg := range dGroupForType.Programs {
					if defProg.Active && defProg.Source != "" {
						if attackProgram.PlayerProgram.Type&defProg.EffectorTypes != 0 {
							pctDefense += defProg.Usage / defender.BandwidthUsage * 100
						}
					}
				}
			}

		} else {
			continue
		}
	}
	result := ProbResult{false, false, false}
	pctDefense -= pctDefense * 0.1 // no negative effect due to numbers on first iteration
	cnt := 1.0
	for attackProgram.PlayerProgram.Amount > 0 {
		pctDefense += pctDefense * (cnt / 10)
		actualPct := MAXCH - pctDefense
		actualVpct := MINVCH + pctDefense
		if actualPct < MINCH {
			actualPct = MINCH
		}
		if actualVpct > MAXVCH {
			actualVpct = MAXVCH
		}
		probs := buildProbability(actualPct, actualVpct)
		rand.Seed(time.Now().UnixNano())
		rd := rand.Int63n(10)
		rs := probs[rd]
		if rs.Killed {
			result.Killed = true
			attackProgram.PlayerProgram.Amount--
			attackProgram.Amount++
		}
		if rs.Visual {
			result.Visual = true
		}
		if rs.Success {
			result.Success = true
			break
		}
		cnt++
	}
	return result, nil
}
