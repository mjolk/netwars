package player

import (
	"appengine"
	"appengine/datastore"
	"errors"
	"math"
	"mj0lk.be/netwars/event"
	"mj0lk.be/netwars/guid"
	"mj0lk.be/netwars/program"
	"strconv"
	"time"
)

var NotEnoughBandwidthError = errors.New("not enough bandwidth for programtype")
var NoProgramToDeallocate = errors.New("Error: no programs to deallocate")

func PlayerProgramKey(c appengine.Context, playerKey, programKey *datastore.Key) *datastore.Key {
	return datastore.NewKey(c, "PlayerProgram", programKey.StringID(), 0, playerKey)
}

func Allocate(c appengine.Context, playerStr, programStr, amount string) error {
	programKey, err := datastore.DecodeKey(programStr)
	playerKey, err := datastore.DecodeKey(playerStr)
	if err != nil {
		return err
	}
	iAmount, _ := strconv.ParseInt(amount, 10, 64)
	prog, err := program.KeyGet(c, programKey)
	if err != nil {
		return err
	}
	if prog.Type == program.INF {
		return errors.New("Can't alloc INFECT program")
	}
	return datastore.RunInTransaction(c, func(c appengine.Context) error {
		pprogramKey := PlayerProgramKey(c, playerKey, programKey)
		player := new(Player)
		if err := Status(c, playerStr, player); err != nil {
			return err
		}
		mCost := prog.Memory * float64(iAmount)
		cCost := prog.Cycles * iAmount
		if cCost > player.Cycles {
			return errors.New("Error not enough cycles")
		} else if mCost > float64(player.Memory) {
			return errors.New("Error: not enough memory")
		}
		if program.CONN != program.CONN&prog.Type {
			if pProgs, ok := player.Programs[prog.Type]; ok {
				if pProgs.Power {
					available := float64(pProgs.Yield) - pProgs.Usage
					availableNr := (available - math.Mod(available, prog.BandwidthUsage)) / prog.BandwidthUsage
					c.Debugf("available bandwidth: %.2f \navailable #: %d\n program usage: %.2f", available, int64(availableNr), prog.BandwidthUsage)
					if int64(availableNr) < iAmount {
						return NotEnoughBandwidthError
					}
				} else {
					return NotEnoughBandwidthError
				}
			} else {
				return NotEnoughBandwidthError
			}
		}
		pProg := &PlayerProgram{
			Program:    *prog,
			ProgramKey: programKey,
			Key:        pprogramKey,
			Active:     true,
		}
		if pProgs, ok := player.Programs[prog.Type]; ok {
			for _, oPprog := range pProgs.Programs {
				if oPprog.Key.Equal(pprogramKey) {
					pProg = oPprog
				}
			}
		}
		pProg.Amount += iAmount
		cycles := prog.Cycles * iAmount
		memory := int64(prog.Memory * float64(iAmount))
		player.Memory -= memory
		player.Cycles -= cycles
		keys := []*datastore.Key{player.Key, pProg.Key}
		models := []interface{}{player, pProg}
		if _, err := datastore.PutMulti(c, keys, models); err != nil {
			return err
		}
		e := &event.Event{
			Player:            playerKey,
			Created:           time.Now(),
			Direction:         event.OUT,
			EventType:         "Allocate",
			TargetName:        player.Nick,
			TargetID:          player.PlayerID,
			NewBandwidthUsage: player.BandwidthUsage + (float64(iAmount) * pProg.BandwidthUsage),
			Memory:            memory,
			Action:            "Allocate",
			Cycles:            cycles,
			EventPrograms:     []event.EventProgram{event.EventProgram{Name: pProg.Name, Amount: iAmount, Owned: true}},
		}
		if err := event.Send(c, []*event.Event{e}, AllocateEvent); err != nil {
			return err
		}
		return nil
	}, nil)
}

func Deallocate(c appengine.Context, playerStr, programStr, amount string) error {
	programKey, _ := datastore.DecodeKey(programStr)
	playerKey, _ := datastore.DecodeKey(playerStr)
	iAmount, _ := strconv.ParseInt(amount, 10, 64)
	prog, err := program.KeyGet(c, programKey)
	if err != nil {
		return err
	}
	if prog.Type == program.INF {
		return errors.New("Can't dealloc INFECT program")
	}
	return datastore.RunInTransaction(c, func(c appengine.Context) error {
		player := new(Player)
		err := Status(c, playerStr, player)
		if err != nil {
			return err
		}
		var pProg *PlayerProgram
		pprogramKey := PlayerProgramKey(c, playerKey, programKey)
		if pProgs, ok := player.Programs[prog.Type]; ok {
			for _, pp := range pProgs.Programs {
				if pp.Key.Equal(pprogramKey) {
					pProg = pp
				}
			}
		} else {
			return NoProgramToDeallocate
		}
		if pProg == nil {
			return NoProgramToDeallocate
		}
		if pProg.Amount == 0 {
			return NoProgramToDeallocate
		}
		pProg.Amount -= iAmount
		cycles := int64(float64(pProg.Cycles) * float64(iAmount) * CYCLEYIELD)
		memory := int64(pProg.Memory * float64(iAmount) * MEMYIELD)
		player.Cycles += cycles
		player.Memory += memory
		keys := []*datastore.Key{playerKey, pprogramKey}
		models := []interface{}{player, pProg}
		if _, err := datastore.PutMulti(c, keys, models); err != nil {
			return err
		}
		e := &event.Event{
			Player:            playerKey,
			Created:           time.Now(),
			Direction:         event.OUT,
			EventType:         "Allocate",
			TargetName:        player.Nick,
			TargetID:          player.PlayerID,
			NewBandwidthUsage: player.BandwidthUsage - (float64(iAmount) * pProg.BandwidthUsage),
			Memory:            memory,
			Action:            "Deallocate",
			Cycles:            cycles,
			EventPrograms:     []event.EventProgram{event.EventProgram{Name: pProg.Name, Amount: iAmount, Owned: true}},
		}
		if err := event.Send(c, []*event.Event{e}, AllocateEvent); err != nil {
			return err
		}
		return nil
	}, nil)
}

func AllocateEvent(c appengine.Context, events []*event.Event) error {
	e := events[0]
	cntCh := make(chan int64)
	go event.NewEventID(c, cntCh)
	e.Result = true
	leGuid, err := guid.GenUUID()
	if err != nil {
		return err
	}
	localId := datastore.NewKey(c, "Event", leGuid, 0, nil)
	e.ID = <-cntCh
	if _, err := datastore.Put(c, localId, e); err != nil {
		c.Debugf("error saving event %s \n", err)
		return err
	}
	return nil
}
