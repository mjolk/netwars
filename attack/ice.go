package attack

import (
	"appengine"
	"appengine/datastore"
	"errors"
	"mj0lk.be/netwars/event"
	"mj0lk.be/netwars/guid"
	"mj0lk.be/netwars/player"
	"mj0lk.be/netwars/program"
	"time"
)

const (
	ICEMAXCH     float64 = 90.0
	ICEMINCH     float64 = 50.0
	ICEKILLMINCH float64 = 20.0
)

func Ice(c appengine.Context, cfg AttackCfg) (AttackEvent, error) {
	c.Infof("running spy attack <<<\n")
	ln := len(cfg.ActivePrograms)
	if ln < 1 || ln > 1 {
		return AttackEvent{}, errors.New("Invalid input")
	}
	attackerKey, defenderKey, err := cfg.Keys(c)
	if err != nil {
		return AttackEvent{}, err
	}
	var response AttackEvent
	options := new(datastore.TransactionOptions)
	options.XG = true
	txErr := datastore.RunInTransaction(c, func(c appengine.Context) error {
		attacker := new(player.Player)
		defender := new(player.Player)
		playerStCh := make(chan int)
		go func() {
			if err := player.Status(c, defenderKey.Encode(), defender); err != nil {
				c.Errorf("get player status error %s \n", err)
			}
			playerStCh <- 0
		}()
		if err := player.Status(c, cfg.Pkey, attacker); err != nil {
			return err
		}
		<-playerStCh
		attackEvent := NewAttackEvent(cfg.AttackType, event.OUT, attacker, defender)
		defenseEvent := NewAttackEvent(cfg.AttackType, event.IN, defender, attacker)
		//TODO check attacker status
		warCh := make(chan int, 1)
		if attacker.ClanKey != nil && defender.ClanKey != nil {
			if attacker.ClanKey.Equal(defender.ClanKey) {
				return errors.New("Can't attack your own team members")
			}
			go loadWar(c, warCh, attackEvent, defenseEvent)
		} else {
			warCh <- 0
		}
		attackProgram := &AttackEventProgram{nil, new(event.EventProgram)}
		activeProg := cfg.ActivePrograms[0]
		attackProgramKey, err := datastore.DecodeKey(activeProg.Key)
		if err != nil {
			return err
		}
		aGroupForType := attacker.Programs[program.ICE]
		for _, aProg := range aGroupForType.Programs {
			if aProg.ProgramKey.Equal(attackProgramKey) {
				attackProgram.Name = aProg.Name
				attackProgram.AmountBefore = aProg.Amount
				attackProgram.AmountUsed = activeProg.Amount
				attackProgram.ProgramActive = aProg.Active
				attackProgram.PlayerProgram = aProg
				attackProgram.ActiveDefender = true
				attackProgram.Power = aGroupForType.Power
				attackProgram.Owned = true
			}
		}
		result, err := renderProb(attackProgram, defender)
		if err != nil {
			return err
		}
		c.Debugf("result : %+v \n", result)
		attackEvent.Result = result.Success
		defenseEvent.Result = !result.Success
		var defInfectKey *datastore.Key
		defInfectProg := new(player.PlayerProgram)
		var attInfectKey *datastore.Key
		attInfectProg := new(player.PlayerProgram)
		attacker.Memory -= 4
		attacker.ActiveMemory -= 4
		war := <-warCh
		attackEvent.Memory = 4
		keys := []*datastore.Key{attackerKey, defenderKey}
		models := []interface{}{attacker, defender}
		if result.Success {
			infectProg, err := program.KeyGet(c, attackProgram.PlayerProgram.Infect)
			if err != nil {
				return err
			}
			defKeyName, err := guid.GenUUID()
			attKeyName, err := guid.GenUUID()
			if err != nil {
				return err
			}
			exp := time.Now().Add(time.Duration(infectProg.Ettl) * time.Second)
			defInfectKey = datastore.NewKey(c, "PlayerProgram", defKeyName, 0, defenderKey)
			defInfectProg = &player.PlayerProgram{
				Program:    *infectProg,
				Source:     attacker.Name,
				Key:        defInfectKey,
				Amount:     infectProg.InfectAmount,
				ProgramKey: attackProgram.PlayerProgram.Infect,
				Expires:    exp,
				Active:     true,
			}
			attInfectKey = datastore.NewKey(c, "PlayerProgram", attKeyName, 0, attackerKey)
			attInfectProg = &player.PlayerProgram{
				Program:    *infectProg,
				Key:        attInfectKey,
				Amount:     infectProg.InfectAmount,
				ProgramKey: attackProgram.PlayerProgram.Infect,
				Expires:    exp,
				Active:     true,
			}
			keys = append(keys, attInfectKey, defInfectKey)
			models = append(models, attInfectProg, defInfectProg)
			if war > 0 {
				attackEvent.CpsGained = int64(5 * war)
				defenseEvent.ApsGained = int64(1 * war)
			}
		}
		if result.Killed {
			if attackProgram.Amount < 0 {
				return errors.New("program amount < 0 , panic \n")
			}
			attackProgram.BwLost = attackProgram.PlayerProgram.BandwidthUsage * float64(attackProgram.Amount)
			attackEvent.ProgramsLost = attackProgram.Amount
			attackEvent.BwLost = attackProgram.BwLost
			attackEvent.EventPrograms = append(attackEvent.EventPrograms, *attackProgram.EventProgram)
			keys = append(keys, attackProgram.PlayerProgram.Key)
			models = append(models, attackProgram.PlayerProgram)
		}
		if _, err := datastore.PutMulti(c, keys, models); err != nil {
			return err
		}
		if err := event.Send(c, []*event.Event{attackEvent.Event, defenseEvent.Event}, event.Func); err != nil {
			return err
		}
		response = *attackEvent
		return nil
	}, options)
	if txErr != nil {
		return AttackEvent{}, err
	}
	return response, nil
}
