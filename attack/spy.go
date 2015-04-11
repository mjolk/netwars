package attack

import (
	"appengine"
	"appengine/datastore"
	"errors"
	"mj0lk.be/netwars/event"
	"mj0lk.be/netwars/player"
	"mj0lk.be/netwars/program"
)

const (
	MAXCH  float64 = 80.0
	MINCH  float64 = 40.0
	MAXVCH float64 = 90.0
	MINVCH float64 = 50.0
)

func Spy(c appengine.Context, playerStr string, cfg AttackCfg) (AttackEvent, error) {
	c.Infof("running spy attack <<<\n")
	ln := len(cfg.ActivePrograms)
	if ln < 1 || ln > 1 {
		return AttackEvent{}, errors.New("Invalid input")
	}
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
		playerStCh := make(chan int)
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
		attackEvent.Memory = 2
		if attacker.ActiveMemory < 2 {
			return errors.New("Not enough active memory")
		}
		attackProgram := &AttackEventProgram{nil, new(event.EventProgram)}
		activeProg := cfg.ActivePrograms[0]
		attackProgramKey, err := datastore.DecodeKey(activeProg.Key)
		if err != nil {
			return err
		}
		aGroupForType := attacker.Programs[program.INT]
		for _, aProg := range aGroupForType.Programs {
			if aProg.ProgramKey.Equal(attackProgramKey) {
				attackProgram.Name = aProg.Name
				attackProgram.AmountBefore = aProg.Amount
				attackProgram.AmountUsed = attackProgram.Amount
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
		var offTotal float64
		var yieldTotal float64
		if attackEvent.Result { // build spy report
			attacker.Aps += 1
			c.Debugf("check FW %s \n", attackProgram.PlayerProgram.Effectors)
			if program.FW&attackProgram.PlayerProgram.EffectorTypes != 0 {
				spGr, ok := defender.Programs[program.FW]
				if ok {
					pType := program.ProgramName[program.FW]
					for _, prog := range spGr.Programs {
						eProg := event.EventProgram{
							Name:     prog.Name,
							Amount:   int64((prog.Usage / spGr.Usage) * 100),
							Owned:    false,
							TypeName: pType,
						}
						attackEvent.EventPrograms = append(attackEvent.EventPrograms, eProg)
					}
				}
			}
			c.Debugf("check BAL %d \n", BAL&attackProgram.PlayerProgram.EffectorTypes)
			if BAL&attackProgram.PlayerProgram.EffectorTypes != 0 {
				var spGrList []event.EventProgram
				for _, ot := range OffensiveTypes {
					spGr, ok := defender.Programs[ot]
					if ok && len(spGr.Programs) > 0 {
						pType := program.ProgramName[ot]
						grEprog := event.EventProgram{
							Name:   pType,
							Amount: int64(spGr.Usage),
							Owned:  false,
							Power:  spGr.Power,
						}
						spGrList = append(spGrList, grEprog)
						for _, prog := range spGr.Programs {
							pEprog := event.EventProgram{
								Name:     prog.Name,
								Amount:   int64((prog.Usage / spGr.Usage) * 100),
								TypeName: pType,
								Owned:    false,
								Power:    spGr.Power,
							}
							attackEvent.EventPrograms = append(attackEvent.EventPrograms, pEprog)
							offTotal += prog.Usage
						}
					}
				}
				for i := range spGrList {
					if spGrList[i].Amount > 0 {
						spGrList[i].Amount = int64((float64(spGrList[i].Amount) / offTotal) * 100)
					}
				}
				attackEvent.EventPrograms = append(attackEvent.EventPrograms, spGrList...)
			}
			c.Debugf("check CONN %d \n", program.CONN&attackProgram.PlayerProgram.EffectorTypes)
			if program.CONN&attackProgram.PlayerProgram.EffectorTypes != 0 {
				for tpe := range defender.Programs {
					if tpe == program.CONN {
						continue
					}
					spGr, ok := defender.Programs[tpe]
					if ok {
						eProg := event.EventProgram{
							Name:     program.ProgramName[tpe],
							Amount:   spGr.Yield,
							Owned:    false,
							TypeName: program.ProgramName[tpe],
						}
						yieldTotal += float64(spGr.Yield)
						attackEvent.EventPrograms = append(attackEvent.EventPrograms, eProg)
					}

				}
				for i := range attackEvent.EventPrograms {
					attackEvent.EventPrograms[i].Amount = int64((float64(attackEvent.EventPrograms[i].Amount) / yieldTotal) * 100)
				}
			}
			c.Debugf("check INF %d \n", program.INF&attackProgram.PlayerProgram.EffectorTypes)
			if program.INF&attackProgram.PlayerProgram.EffectorTypes != 0 {
				spGr, ok := defender.Programs[program.INF]
				if ok {
					pType := program.ProgramName[program.INF]
					for _, prg := range spGr.Programs {
						eProg := event.EventProgram{
							Name:     prg.Name,
							Amount:   int64((prg.Usage / spGr.Usage) * 100),
							TypeName: pType,
							Owned:    false,
							Source:   prg.Source,
							Power:    spGr.Power,
						}
						attackEvent.EventPrograms = append(attackEvent.EventPrograms, eProg)
					}
				}
			}
		}
		attacker.Memory -= attackEvent.Memory
		attacker.ActiveMemory -= attackEvent.Memory
		c.Debugf("attackEvent &+v \n", attackEvent.Event.EventPrograms)
		if result.Killed {
			if attackProgram.PlayerProgram.Amount < 0 {
				return errors.New("program amount < 0 : panic \n")
			}
			attackEvent.EventPrograms = append(attackEvent.EventPrograms, *attackProgram.EventProgram)
			if _, err := datastore.PutMulti(c, []*datastore.Key{attackerKey, attackProgram.PlayerProgram.Key},
				[]interface{}{attacker, attackProgram.PlayerProgram}); err != nil {
				return err
			}
		} else {
			if _, err := datastore.Put(c, attackerKey, attacker); err != nil {
				return err
			}
		}
		evs := []*event.Event{attackEvent.Event}
		if result.Visual || result.Killed || !result.Success {
			defenseEvent := NewAttackEvent(cfg.AttackType, event.IN, defender, attacker)
			var defPr event.EventProgram
			defPr = *attackProgram.EventProgram
			defPr.Owned = false
			defenseEvent.EventPrograms = append(defenseEvent.EventPrograms, defPr)
			evs = append(evs, defenseEvent.Event)
		}
		if err := event.Send(c, evs, event.Func); err != nil {
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
