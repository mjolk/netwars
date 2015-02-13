package user

import (
	"appengine/datastore"
	"bytes"
	"encoding/gob"
	"time"
)

type EventProgram struct {
	Name             string         `json:"name"`
	Amount           float64        `json: "amount"`
	Owned            bool           `json:"owned"`
	TypeName         string         `json:"type_name"`
	AmountUsed       int64          `json:"amount_used" datastore:",noindex`
	AmountBefore     int64          `json:"amount_before" datastore:",noindex`
	AmountLost       []int64        `json:"amount_after" datastore:",noindex`
	Lost             int64          `json:"amount_lost" datastore:",noindex"`
	Program          *datastore.Key `json:"program" datastore:",noindex`
	ProgramActive    bool           `json:"program_active" datastore:",noindex`
	BwLost           float64        `json:"bw_lost" datastore:",noindex`
	PlayerProgram    *PlayerProgram `datastore:"-" json:"-"`
	ActiveDefender   bool           `json:"-" datastore:",noindex`
	AttackEfficiency float64        `json:"-" datastore:"-" datastore:",noindex`
	YieldLost        int64          `json:"yield_lost" datastore:",noindex`
	Power            bool           `datastore:",noindex" json:"power"`
	VDamageReceived  int64          `datastore:",noindex" json:"-"`
}

// convention:
// event.Player is the one who owns the event
// need different events because of no OR queries.
// direction IN indicates the initiator of the event is TargetName, TargetID
// direction OUT indicates the initiator of the event is Player, PlayerName, PlayerID eg the owner of the event
type Event struct {
	ClanMember        *datastore.Key `json:"-" datastore:",noindex"`
	ClanConnection    *datastore.Key
	Target            *datastore.Key `json:"-" datastore:",noindex`
	ID                int64          `json:"event_id"`
	Created           time.Time      `json:"created"`
	Player            *datastore.Key `json:"-"`
	EventType         string         `json:"event_type"`
	Result            bool           `json:"result"`
	Direction         int64          `json:"direction"`
	Clan              *datastore.Key `json:"clan_key"`
	PlayerName        string         `json:"player_name"`
	PlayerID          int64          `json:"player_id"`
	TargetName        string         `json:"target_name"`
	TargetID          int64          `json:"target_id"`
	ClanName          string         `json:"clan_name"`
	ClanID            int64          `json:"clan_id"`
	Expires           time.Time      `json:"expires"`
	NewBandwidthUsage float64        `json:"new_bandwidth_usage"`
	Memory            int64          `json:"mem_cost" datastore:",noindex"`
	Action            string         `json:"action"`
	BwLost            float64        `json:"bw_lost"`
	ProgramsLost      int64          `json:"programs_lost"`
	BwKilled          float64        `json:"bw_killed"`
	YieldLost         int64          `json:"yield_lost"`
	ProgramsKilled    int64          `json:"programs_killed"`
	ApsGained         int64          `json:"aps_gained" datastore:",noindex`
	CpsGained         int64          `json:"cps_gained" datastore:",noindex`
	CyclesGained      int64          `json:"cycles_gained"`
	Cycles            int64          `json:"cycles_lost"`
	EventPrograms     []EventProgram `json:"active_programs" datastore:"-"`
	Eprogs            []byte         `json:"-" datastore:",noindex"`
	Email             string         `json:"-" datastore:"-"`
	VDamageReceived   int64          `datastore:",noindex" json:"-"`
}

func (e *Event) Load(c <-chan datastore.Property) error {
	if err := datastore.LoadStruct(e, c); err != nil {
		return err
	}
	if len(e.Eprogs) > 0 {
		var epBytes = bytes.NewBuffer(e.Eprogs)
		if err := gob.NewDecoder(epBytes).Decode(&e.EventPrograms); err != nil {
			return err
		}
	}
	return nil
}

func (e *Event) Save(c chan<- datastore.Property) error {
	if len(e.EventPrograms) > 0 {
		var epBytes bytes.Buffer
		if err := gob.NewEncoder(&epBytes).Encode(&e.EventPrograms); err != nil {
			return err
		}
		e.Eprogs = epBytes.Bytes()
	}
	return datastore.SaveStruct(e, c)
}
