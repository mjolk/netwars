package attack

import (
	"appengine"
	"appengine/datastore"
	"errors"
	"fmt"
	"math/rand"
	"netwars/program"
	"netwars/user"
	"netwars/utils"
	"strconv"
	"time"
)

const (
	MAXCH          int64 = 80
	MINCH          int64 = 40
	MAXVCH         int64 = 90
	MINVCH         int64 = 50
	SPY_EVENT_PATH       = "/events/spy"
)

type SpyResult struct {
	Visual  bool
	Success bool
	Killed  bool
}

type SpyEventProgram struct {
	Name          string
	AmountUsed    int64 `json:"amount_used" datastore:",noindex`
	Amount        float64
	TypeName      string
	AmountLost    int64               `json:"amount_after" datastore:",noindex`
	BwLost        float64             `json:"bw_lost" datastore:",noindex`
	PlayerProgram *user.PlayerProgram `datastore:"-" json:"-"`
	Power         bool                `datastore:",noindex" json:"power"`
	Owned         bool                `json:"owned"`
}

type SpyEvent struct {
	utils.Event
	SPResult   SpyResult
	Programs   []*SpyEventProgram `datastore:"-" json:"-"`
	Target     *datastore.Key     `json:"-" datastore:",noindex`
	AttackType int64              `json:"attack_type" datastore:",noindex"`
	Clan       *datastore.Key
}

func buildSpyProbability(actPct, actVpct int64) []SpyResult {
	r := actPct / 10
	v := actVpct / 10
	rv := make([]SpyResult, 10)
	var c int64
	for c = 0; c < 10; c++ {
		spResult := SpyResult{false, false, false}
		if c < r {
			spResult.Success = true
		}
		if c < v {
			spResult.Visual = true
		}
		if c < r-5 {
			spResult.Killed = true
		}
		rv[c] = spResult
	}
	return rv
}

func newSpyResponse(c appengine.Context, spyEvent SpyEvent, defender *user.Player) Response {
	response := Response{
		Event:      spyEvent.Event,
		TargetName: defender.Nick,
		TargetID:   defender.PlayerID,
		Action:     AttackName[spyEvent.AttackType],
	}
	for _, aProg := range spyEvent.Programs {
		eventProgram := EventProgram{
			Amount:     aProg.Amount,
			AmountLost: aProg.AmountLost,
			Name:       aProg.Name,
			Owned:      aProg.Owned,
			TypeName:   aProg.TypeName,
		}
		response.EventPrograms = append(response.EventPrograms, eventProgram)

	}
	return response
}

func Spy(c appengine.Context, cfg AttackCfg) (Response, error) {
	c.Infof("running spy attack <<<\n")
	akey, err := datastore.DecodeKey(cfg.Pkey)
	defenderID, err := strconv.ParseInt(cfg.Target, 10, 64)
	if err != nil {
		return Response{}, err
	}
	defenderKey, err := user.KeyByID(c, defenderID)
	if err != nil {
		return Response{}, err
	}
	var response Response
	options := new(datastore.TransactionOptions)
	options.XG = true
	txErr := datastore.RunInTransaction(c, func(c appengine.Context) error {
		now := time.Now()
		attackerState := new(user.PlayerState)
		defenderState := new(user.PlayerState)
		playerStCh := make(chan int)
		go func() {
			if _, err := user.Status(c, defenderKey.Encode(), defenderState); err != nil {
				c.Errorf("get player status error %s \n", err)
			}
			playerStCh <- 0
		}()
		if _, err := user.Status(c, cfg.Pkey, attackerState); err != nil {
			return err
		}
		<-playerStCh
		dSpyEvent := SpyEvent{
			Event: utils.Event{
				Created:   now,
				EventType: "Attack",
				Direction: utils.IN,
				Player:    defenderKey,
			},
			Target:     akey,
			AttackType: AttackType[cfg.AttackType],
		}
		aSpyEvent := SpyEvent{
			Event: utils.Event{
				Created:   now,
				EventType: "Attack",
				Direction: utils.OUT,
				Player:    akey,
			},
			Target:     defenderKey,
			AttackType: AttackType[cfg.AttackType],
		}
		//TODO check attacker status
		if attackerState.Player.ClanMember != nil {
			aSpyEvent.Clan = attackerState.Player.ClanMember.Parent()
		}
		if defenderState.Player.ClanMember != nil {
			dSpyEvent.Clan = defenderState.Player.ClanMember.Parent()
		}
		var aeSpyProgram *SpyEventProgram
		var pctDefense int64
		var attackProgram *user.PlayerProgram
		fmt.Printf("active programs : %+v \n", cfg.ActivePrograms)
		for _, eProgram := range cfg.ActivePrograms {
			attackProgramKey, err := datastore.DecodeKey(eProgram.Key)
			if err != nil {
				return err
			}
			if aGroupForType, ok := attackerState.Programs[program.INT]; ok {
				for _, aProg := range aGroupForType.Programs {
					fmt.Printf("aprog.ProgramKey = %s \n attack program key : %s \n", aProg.ProgramKey, attackProgramKey)
					if aProg.ProgramKey.Equal(attackProgramKey) {
						attackProgram = aProg
						aeSpyProgram = &SpyEventProgram{
							AmountUsed:    1,
							Name:          attackProgram.Name,
							PlayerProgram: attackProgram,
							Power:         aGroupForType.Power,
							Owned:         true,
						}
						if fws, ok := defenderState.Programs[program.FW]; ok {
							if fws.Power {
								for _, fw := range fws.Programs {
									if fw.Active {
										if aProg.Type&fw.EffectorTypes != 0 {
											pctDefense = int64(fw.Usage / attackerState.Player.BandwidthUsage * 100)
										}
									}
								}
							}

						}
					}
				}
			} else {
				return errors.New("Fatal error: no intelligence type programs ")
			}
		}
		actualPct := MAXCH - pctDefense
		actualVpct := MINVCH + pctDefense
		if actualPct < MINCH {
			actualPct = MINCH
		}
		if actualVpct > MAXVCH {
			actualVpct = MAXVCH
		}
		fmt.Printf("actual pct : %d actual visual pct : %d \n", actualPct, actualVpct)
		probs := buildSpyProbability(actualPct, actualVpct)
		//fmt.Printf("probabilities : %+v \n", probs)
		rand.Seed(time.Now().UnixNano())
		rd := rand.Int63n(10)
		fmt.Printf("result according to pct %d \n", rd)
		result := probs[rd]
		fmt.Printf("result : %+v \n", result)
		aSpyEvent.SPResult = result
		dSpyEvent.SPResult = result
		aSpyEvent.Result = result.Success
		dSpyEvent.Result = !result.Success
		var offTotal float64
		var yieldTotal float64
		if result.Success {
			fmt.Printf("check FW %d \n", program.FW&attackProgram.EffectorTypes)
			if program.FW&attackProgram.EffectorTypes != 0 {
				spGr, ok := defenderState.Programs[program.FW]
				if ok {
					pType := program.ProgramName[program.FW]
					for _, prog := range spGr.Programs {
						eProg := &SpyEventProgram{
							Name:          prog.Name,
							Amount:        (prog.Usage / spGr.Usage) * 100,
							Owned:         false,
							PlayerProgram: prog,
							TypeName:      pType,
						}
						aSpyEvent.Programs = append(aSpyEvent.Programs, eProg)
					}
				}
			}
			fmt.Printf("check BAL %d \n", BAL&attackProgram.EffectorTypes)
			if BAL&attackProgram.EffectorTypes != 0 {
				var spGrList []*SpyEventProgram
				for _, ot := range OffensiveTypes {
					spGr, ok := defenderState.Programs[ot]
					if ok && len(spGr.Programs) > 0 {
						pType := program.ProgramName[ot]
						grEprog := &SpyEventProgram{
							Name:   pType,
							Amount: spGr.Usage,
							Owned:  false,
							Power:  spGr.Power,
						}
						spGrList = append(spGrList, grEprog)
						for _, prog := range spGr.Programs {
							pEprog := &SpyEventProgram{
								Name:          prog.Name,
								Amount:        (prog.Usage / spGr.Usage) * 100,
								TypeName:      pType,
								Owned:         false,
								Power:         spGr.Power,
								PlayerProgram: prog,
							}
							aSpyEvent.Programs = append(aSpyEvent.Programs, pEprog)
							offTotal += prog.Usage
						}
					}
				}
				for _, spGrc := range spGrList {
					if spGrc.Amount > 0 {
						spGrc.Amount = (spGrc.Amount / offTotal) * 100
					}
				}
				aSpyEvent.Programs = append(aSpyEvent.Programs, spGrList...)
			}
			fmt.Printf("check CONN %d \n", program.CONN&attackProgram.EffectorTypes)
			if program.CONN&attackProgram.EffectorTypes != 0 {
				for tpe := range defenderState.Programs {
					if tpe == program.CONN {
						continue
					}
					spGr, ok := defenderState.Programs[tpe]
					if ok {
						pType := program.ProgramName[tpe]
						eProg := &SpyEventProgram{
							Name:     program.ProgramName[tpe],
							Amount:   float64(spGr.Yield),
							Owned:    false,
							TypeName: pType,
						}
						yieldTotal += float64(spGr.Yield)
						aSpyEvent.Programs = append(aSpyEvent.Programs, eProg)
					}

				}
				for _, eProgc := range aSpyEvent.Programs {
					eProgc.Amount = (eProgc.Amount / yieldTotal) * 100
				}
			}
		}
		attackerState.Player.Memory -= 2
		if result.Killed {
			attackProgram.Amount-- // check for negative numbers but then something is already seriously wrong before we get here
			if attackProgram.Amount < 0 {
				return errors.New("program amount < 0 : panic \n")
			}
			aeSpyProgram.AmountLost = 1
			aSpyEvent.Programs = append(aSpyEvent.Programs, aeSpyProgram)
			if _, err := datastore.PutMulti(c, []*datastore.Key{akey, attackProgram.Key},
				[]interface{}{attackerState.Player, attackProgram}); err != nil {
				return err
			}
		} else {
			if _, err := datastore.Put(c, akey, attackerState.Player); err != nil {
				return err
			}
		}
		if err := utils.SendEvent(c, []SpyEvent{aSpyEvent, dSpyEvent}, SPY_EVENT_PATH); err != nil {
			return err
		}
		response = newSpyResponse(c, aSpyEvent, defenderState.Player)
		return nil
	}, options)
	if txErr != nil {
		return Response{}, err
	}
	return response, nil
}
