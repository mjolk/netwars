package attack

import (
	"appengine"
	"appengine/datastore"
	"errors"
	"fmt"
	"math"
	"mj0lk.be/netwars/clan"
	"mj0lk.be/netwars/event"
	"mj0lk.be/netwars/player"
	"mj0lk.be/netwars/program"
	"mj0lk.be/netwars/utils"
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
