package attack

import (
	"appengine"
	"appengine/datastore"
	"errors"
	"fmt"
	"math/rand"
	"netwars/guid"
	"netwars/program"
	"netwars/user"
	"netwars/utils"
	"strconv"
	"time"
)

const (
	ICEMAXCH       int64 = 90
	ICEMINCH       int64 = 50
	ICEKILLMINCH   int64 = 20
	ICE_EVENT_PATH       = "/events/ice"
)

type IceResult struct {
	Success bool
	Killed  bool
}

type IceEventProgram struct {
	AmountLost    int64 `json:"amount_after" datastore:",noindex`
	AmountUsed    int64
	BwLost        float64             `json:"bw_lost" datastore:",noindex`
	PlayerProgram *user.PlayerProgram `datastore:"-" json:"-"`
	Power         bool                `datastore:",noindex" json:"power"`
}

type IceEvent struct {
	utils.Event
	Programs       []IceEventProgram `datastore:"-" json:"-"`
	ICEResult      IceResult
	Target         *datastore.Key `json:"-" datastore:",noindex`
	AttackType     int64          `json:"attack_type" datastore:",noindex"`
	ClanMember     *datastore.Key
	ClanConnection *datastore.Key
	ApsGained      int64   `json:"aps_gained"`
	CpsGained      int64   `json:"cps_gained"`
	ActiveMem      int64   `json:"active_mem_cost"`
	BwLost         float64 `json:"bw_lost"`
	ProgramsLost   int64   `json:"programs_lost"`
	YieldLost      int64   `json:"yield_lost"`
}

func buildIceProbability(actPct, killPct int64) []IceResult {
	r := actPct / 10
	k := killPct / 10
	rv := make([]IceResult, 10)
	var c int64
	for c = 0; c < 10; c++ {
		iceResult := IceResult{false, false}
		if c < r {
			iceResult.Success = true
		}
		if c < k {
			iceResult.Killed = true
		}
		rv[c] = iceResult
	}
	return rv
}

func createIceResponse(c appengine.Context, iceEvent IceEvent, defender *user.Player) Response {
	response := Response{
		Event:      iceEvent.Event,
		TargetName: defender.Nick,
		TargetID:   defender.PlayerID,
		Action:     AttackName[iceEvent.AttackType],
	}
	for _, aProg := range iceEvent.Programs {
		eventProgram := EventProgram{
			Name:  aProg.PlayerProgram.Name,
			Owned: true,
		}
		response.EventPrograms = append(response.EventPrograms, eventProgram)

	}
	return response
}

func Ice(c appengine.Context, cfg AttackCfg) (Response, error) {
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
		dIceEvent := IceEvent{
			Event: utils.Event{
				Created:   now,
				EventType: "Attack",
				Direction: utils.IN,
				Player:    defenderKey,
			},
			Target:     akey,
			AttackType: AttackType[cfg.AttackType],
		}
		aIceEvent := IceEvent{
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
			aIceEvent.ClanMember = attackerState.Player.ClanMember
		}
		if defenderState.Player.ClanMember != nil {
			dIceEvent.ClanMember = defenderState.Player.ClanMember
		}
		warCh := make(chan int, 1)
		connKeys := make([]*datastore.Key, 2)
		if attackerState.Player.ClanMember != nil && defenderState.Player.ClanMember != nil {
			loadWar(c, attackerState.Player.ClanMember, defenderState.Player.ClanMember, connKeys, warCh)
		} else {
			warCh <- 0
		}
		var aeIceProgram IceEventProgram
		var pctDefense int64
		var attackProgram *user.PlayerProgram
		for _, eProgram := range cfg.ActivePrograms {
			attackProgramKey, err := datastore.DecodeKey(eProgram.Key)
			if err != nil {
				return err
			}
			if aGroupForType, ok := attackerState.Programs[program.ICE]; ok {
				for _, aProg := range aGroupForType.Programs {
					if aProg.ProgramKey.Equal(attackProgramKey) {
						attackProgram = aProg
						aeIceProgram = IceEventProgram{
							PlayerProgram: attackProgram,
							Power:         aGroupForType.Power,
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
				return errors.New("Fatal error: no Ice type programs ")
			}
		}
		actualPct := ICEMAXCH - pctDefense
		actualKpct := ICEKILLMINCH + pctDefense
		if actualPct < ICEMINCH {
			actualPct = ICEMINCH
		}
		if actualKpct > 100 {
			actualKpct = 100
		}
		fmt.Printf("success pct : %d kill pct : %d \n", actualPct, actualKpct)
		probs := buildIceProbability(actualPct, actualKpct)
		//fmt.Printf("probabilities : %+v \n", probs)
		rand.Seed(time.Now().UnixNano())
		rd := rand.Int63n(10)
		fmt.Printf("result according to pct %d \n", rd)
		result := probs[rd]
		fmt.Printf("result : %+v \n", result)
		aIceEvent.ICEResult = result
		dIceEvent.ICEResult = result
		aIceEvent.Result = result.Success
		dIceEvent.Result = !result.Success
		var infectPprog *user.PlayerProgram
		var infectPprogKey *datastore.Key
		if result.Success {
			infectProg, err := program.KeyGet(c, attackProgram.Infect)
			if err != nil {
				return err
			}
			keyName, err := guid.GenUUID()
			if err != nil {
				return err
			}
			infectPprogKey = datastore.NewKey(c, "PlayerProgram", keyName, 0, defenderKey)
			infectPprog = &user.PlayerProgram{
				Key:        infectPprogKey,
				Amount:     1,
				ProgramKey: attackProgram.Infect,
				Expires:    time.Now().Add(time.Duration(infectProg.Ettl) * time.Second),
				Active:     true,
			}
		}
		attackerState.Player.Memory -= 4
		attackerState.Player.ActiveMemory -= 4
		defenderState.Player.NewLocals++
		war := <-warCh
		aIceEvent.ClanConnection = connKeys[0]
		aIceEvent.ActiveMem = 4
		dIceEvent.ClanConnection = connKeys[1]
		keys := []*datastore.Key{akey, defenderKey}
		models := []interface{}{attackerState.Player, defenderState.Player}
		if result.Success {
			keys = append(keys, infectPprogKey)
			models = append(models, infectPprog)
			if war > 0 {
				aIceEvent.CpsGained = int64(10 / war)
				dIceEvent.ApsGained = int64(2 / war)
			}
		}
		if result.Killed {
			attackProgram.Amount--
			if attackProgram.Amount < 0 {
				return errors.New("program amount < 0 , panic \n")
			}
			aeIceProgram.AmountLost = 1
			aeIceProgram.BwLost = attackProgram.BandwidthUsage
			aIceEvent.ProgramsLost = 1
			aIceEvent.BwLost = attackProgram.BandwidthUsage
			aIceEvent.YieldLost = attackProgram.Bandwidth
			aIceEvent.Programs = append(aIceEvent.Programs, aeIceProgram)
			keys = append(keys, attackProgram.Key)
			models = append(models, attackProgram)
		}
		if _, err := datastore.PutMulti(c, keys, models); err != nil {
			return err
		}
		if err := utils.SendEvent(c, []IceEvent{aIceEvent, dIceEvent}, ICE_EVENT_PATH); err != nil {
			return err
		}
		response = createIceResponse(c, aIceEvent, defenderState.Player)
		return nil
	}, options)
	if txErr != nil {
		return Response{}, err
	}
	return response, nil
}
