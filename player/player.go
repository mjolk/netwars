package player

import (
	"appengine"
	"appengine/datastore"
	"errors"
	"fmt"
	"mj0lk.be/netwars/cache"
	"mj0lk.be/netwars/counter"
	"mj0lk.be/netwars/guid"
	"mj0lk.be/netwars/program"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	MEMYIELD           = 0.4
	CYCLEYIELD         = 0.5
	TIMEDELIM          = "@"
	TIMETPL            = "%d@%d"
	STARTMEM           = 50
	STARTCYC           = 1000
	ACTIVEMEMMAX       = 10
	LIMIT              = 100
	THUMBSIZE          = 32
	EMAILREGEX         = `(\w[-._\w]*\w@\w[-._\w]*\w\.\w{2,3})`
	NICKREGEX          = `^[a-zA-Z0-9_.-]*$`
	TIMELAYOUT         = "2006-Jan-02"
	USERTYPE     int64 = 1024
	REGULAR      int64 = USERTYPE << iota
	PUSER        int64 = USERTYPE << iota
	PCLAN        int64 = USERTYPE << iota
	MOD          int64 = USERTYPE << iota
	ADMIN        int64 = USERTYPE << iota
	DEAD         int64 = 0x2800
	LIVE         int64 = 0x2800 << 1
	SUSPENDED    int64 = 0x2800 << 2
)

var (
	emailMatcher, _  = regexp.Compile(EMAILREGEX)
	nickMatcher, _   = regexp.Compile(NICKREGEX)
	PlayerStatusName = map[int64]string{
		DEAD:      "Killed",
		LIVE:      "Live",
		SUSPENDED: "Suspended",
	}
	PlayerStatus = map[string]int64{
		"Killed":    DEAD,
		"Live":      LIVE,
		"Suspended": SUSPENDED,
	}
	PlayerTypeName = map[int64]string{
		REGULAR: "Free Player",
		PUSER:   "Subscribed Player",
		PCLAN:   "Subscribed Clan",
		MOD:     "Moderator",
		ADMIN:   "Administrator",
	}
	PlayerType = map[string]int64{
		"Free Player":       REGULAR,
		"Subscribed Player": PUSER,
		"Subscribed Clan":   PCLAN,
		"Moderator":         MOD,
		"Administrator":     ADMIN,
	}
)

type Player struct {
	EncodedKey       string                        `datastore:"-" json:"key"`
	Key              *datastore.Key                `datastore:"-" json:"-"`
	Cps              int64                         `json:"cps"`
	Aps              int64                         `json:"aps"`
	EncodedMember    string                        `datastore:"-" json:"clan_member"`
	ClanMember       *datastore.Key                `json:"-"`
	Cycles           int64                         `datastore:"-" json:"cycles"`
	Memory           int64                         `datastore:"-" json:"mem"`
	ActiveMemory     int64                         `datastore:"-" json:"active_mem"`
	CyclesUpdated    time.Time                     `datastore:"-" json:"-"`
	MemUpdated       time.Time                     `datastore:"-" json:"-"`
	ActiveMemUpdated time.Time                     `datastore:"-" json:"-"`
	Scycles          string                        `datastore:",noindex" json:"-"`
	Smem             string                        `datastore:",noindex" json:"-"`
	SactiveMem       string                        `datastore:",noindex" json:"-"`
	Bandwidth        int64                         `datastore:"-" json:"bandwidth"`
	BandwidthUsage   float64                       `json:"bandwidth_usage"`
	Updated          time.Time                     `json:"updated"`
	NewLocals        int64                         `json:"new_locals"`
	Created          time.Time                     `json:"created"`
	Email            string                        `json:"email"`
	Nick             string                        `json:"nick"`
	Name             string                        `json:"name"`
	Address          string                        `json:"address"`
	Signature        string                        `json:"signature"`
	Birthday         time.Time                     `json:"birthday"`
	AvatarKey        appengine.BlobKey             `json:"-"`
	Avatar           string                        `json:"avatar"`
	AvatarThumb      string                        `datastore:"-" json:"avatar_thumb"`
	PlayerID         int64                         `json:"player_id"`
	Status           int64                         `json:"-"`
	StatusName       string                        `json:"status" datastore:"-"`
	AccessName       string                        `json:"type" datastore:"-"`
	Clan             string                        `json:"clan"`
	ClanTag          string                        `json:"clan_tag"`
	Country          string                        `json:"country"`
	Language         string                        `json:"language"`
	Access           int64                         `json:"-"`
	Verified         bool                          `json:"-" datastore:",noindex"`
	DeviceID         string                        `json:"-" datastore:",noindex"`
	Programs         map[int64]*PlayerProgramGroup `json:"-" datastore:"-"`
	PlayerPrograms   []*PlayerProgramGroup         `json:"programs, omitempty" datastore:"-"`
	Cleanup          []*datastore.Key              `json:"-" datastore:"-"`
}

type PlayerProgramGroup struct {
	Yield    int64            `json:"yield"`
	Usage    float64          `json:"usage"`
	Power    bool             `json:"power"`
	Programs []*PlayerProgram `json:"programs"`
	Type     string           `json:"type"`
}

//parent = player
type PlayerProgram struct {
	Amount     int64          `json:"amount"`
	ProgramKey *datastore.Key `json:"-"`
	Usage      float64        `json:"usage" datastore:"-"`
	Yield      int64          `json:"yield" datastore:"-"`
	Key        *datastore.Key `json:"-" datastore:"-"`
	Expires    time.Time      `datastore:",noindex" json:"expires"`
	Active     bool           `json:"active"`
	*program.Program
}

func RangeForPlayer(player *Player) (float64, float64) {
	lo := player.BandwidthUsage - (player.BandwidthUsage * 20.0 / 100.0)
	hi := player.BandwidthUsage + (player.BandwidthUsage * 30.0 / 100.0)
	return lo, hi
}

func NewPlayer() *Player {
	now := time.Now()
	p := &Player{
		Cycles:           STARTCYC,
		Memory:           STARTMEM,
		ActiveMemory:     ACTIVEMEMMAX,
		CyclesUpdated:    now,
		MemUpdated:       now,
		ActiveMemUpdated: now,
	}
	return p
}

func (pp *PlayerProgram) Load(c <-chan datastore.Property) error {
	if err := datastore.LoadStruct(pp, c); err != nil {
		return err
	}
	if !pp.Expires.IsZero() {
		if pp.Expires.Before(time.Now()) {
			pp.Active = false
		}
	}
	return nil
}

func (pp *PlayerProgram) Save(c chan<- datastore.Property) error {
	if pp.ProgramKey == nil {
		return errors.New("program required")
	}
	return datastore.SaveStruct(pp, c)
}

func (p *Player) Load(c <-chan datastore.Property) error {
	if err := datastore.LoadStruct(p, c); err != nil {
		return err
	}
	p.Cycles, p.CyclesUpdated = timedResource(p.Scycles, 15, 50, 5e4)
	p.Memory, p.MemUpdated = timedResource(p.Smem, 15, 1, 300)
	p.ActiveMemory, p.ActiveMemUpdated = timedResource(p.SactiveMem, 60, 2, ACTIVEMEMMAX)
	if p.ClanMember != nil {
		p.EncodedMember = p.ClanMember.Encode()
	}
	if len(p.Avatar) > 0 {
		p.AvatarThumb = fmt.Sprintf("%s=s%d", p.Avatar, THUMBSIZE)
	}
	if len(p.ClanTag) > 0 {
		p.Nick = fmt.Sprintf("<%s> %s", p.ClanTag, p.Nick)
	}
	p.AccessName = PlayerTypeName[p.Access]
	p.StatusName = PlayerStatusName[p.Status]
	return nil
}

func (p *Player) Save(c chan<- datastore.Property) error {
	p.Scycles = fmt.Sprintf(TIMETPL, p.Cycles, p.CyclesUpdated.Unix())
	p.Smem = fmt.Sprintf(TIMETPL, p.Memory, p.MemUpdated.Unix())
	p.SactiveMem = fmt.Sprintf(TIMETPL, p.ActiveMemory, p.ActiveMemUpdated.Unix())
	p.Updated = time.Now()
	if p.Created.IsZero() {
		p.Created = time.Now()
	}
	return datastore.SaveStruct(p, c)
}

func timedResource(src string, interval, amount, max int64) (int64, time.Time) {
	content := strings.Split(src, TIMEDELIM)
	value, err := strconv.ParseInt(content[0], 10, 64)
	updatedInt, err := strconv.ParseInt(content[1], 10, 64)
	if err != nil {
		panic("unexpected fatal error")
	}
	updated := time.Unix(updatedInt, 0)
	durationMins := int64(time.Now().Sub(updated).Minutes())
	if durationMins > interval && value < max {
		rtt := durationMins - (durationMins % interval)
		value += rtt / interval * amount
		updated = updated.Add(time.Duration(rtt) * time.Minute)
	}
	if value > max {
		value = max
	}
	return value, updated
}

func KeyByID(c appengine.Context, id int64) (*datastore.Key, error) {
	k := fmt.Sprintf("%d", id)
	rk := []*datastore.Key{new(datastore.Key)}
	if !cache.Get(c, k, rk[0]) {
		q := datastore.NewQuery("Player").Filter("PlayerID =", id).KeysOnly()
		result := make([]Player, 0, 1)
		keys, err := q.GetAll(c, &result)
		if err != nil {
			return nil, err
		}
		if len(keys) > 0 {
			rk[0] = keys[0]
			cache.Add(c, k, rk[0])
		}
	}
	return rk[0], nil
}

func Get(c appengine.Context, playerStr string, player *Player) error {
	playerKey, err := datastore.DecodeKey(playerStr)
	if err != nil {
		return nil
	}
	if !cache.Get(c, playerKey.StringID(), player) {
		if err := datastore.Get(c, playerKey, player); err != nil {
			return err
		}
		cache.Add(c, playerKey.StringID(), player)
	}
	return nil
}

func Status(c appengine.Context, playerStr string, player *Player) error {
	playerKey, err := datastore.DecodeKey(playerStr)
	if err != nil {
		return err
	}
	if err := datastore.Get(c, playerKey, player); err != nil {
		return err
	}
	player.Key = playerKey
	player.Programs = make(map[int64]*PlayerProgramGroup)
	player.Cleanup = make([]*datastore.Key, 0)
	q := datastore.NewQuery("PlayerProgram").Ancestor(playerKey)
	for t := q.Run(c); ; {
		var pp PlayerProgram
		key, err := t.Next(&pp)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return err
		}
		if pp.Amount == 0 {
			continue
		}
		pp.Key = key
		pp.Usage = pp.BandwidthUsage * float64(pp.Amount)
		pp.Yield = pp.Bandwidth * pp.Amount
		var group *PlayerProgramGroup
		var yGroup *PlayerProgramGroup
		var mapOk bool
		group, mapOk = player.Programs[pp.Type]
		if !mapOk {
			group = new(PlayerProgramGroup)
			group.Type = program.ProgramName[pp.Type]
			group.Programs = make([]*PlayerProgram, 0)
			player.Programs[pp.Type] = group
		}
		if pp.Active && program.CONN&pp.Type == 0 {
			group.Usage += pp.Usage
			player.BandwidthUsage += pp.Usage
		} else if pp.Type == program.INF {
			player.Cleanup = append(player.Cleanup, pp.Key)
		}
		if pp.Yield > 0 {
			yGroup, mapOk = player.Programs[pp.EffectorTypes] // programs with yield have only one effectortype for now
			if !mapOk {
				yGroup = new(PlayerProgramGroup)
				yGroup.Type = program.ProgramName[pp.EffectorTypes]
				yGroup.Programs = make([]*PlayerProgram, 0)
				yGroup.Usage = 0.0
				player.Programs[pp.EffectorTypes] = yGroup
			}
			yGroup.Power = true
			if pp.Active {
				player.Bandwidth += pp.Yield
				yGroup.Yield += pp.Yield
			} else if pp.Type == program.INF {
				for _, ckey := range player.Cleanup {
					if ckey.Equal(pp.Key) {
						player.Cleanup = append(player.Cleanup, pp.Key)
						break
					}
				}
			}
		}
		group.Power = true
		group.Programs = append(group.Programs, &pp)
	}
	i := len(playerPrograms)
	player.PlayerPrograms = make([]*PlayerProgramGroup, i)
	for cType, cGroup := range player.PlayerPrograms {
		i--
		player.PlayerPrograms[i] = cGroup
		ignoreTypes := program.CONN | program.INF
		if cType == ignoreTypes&cType {
			continue
		}
		usage := float64(player.Bandwidth) - player.BandwidthUsage
		ppusage := float64(cGroup.Yield) - cGroup.Usage
		if usage <= 0.0 || ppusage <= 0.0 {
			cGroup.Power = false
		}
	}
	return nil
}

func badEmail(email string) bool {
	if emailMatcher.MatchString(email) == true {
		return false
	}
	return true
}

func badNick(nick string) bool {
	if nickMatcher.MatchString(nick) == true {
		return false
	}
	return true
}

func ValidPlayer(c appengine.Context, email, nick string) (map[string]int, error) {
	if badEmail(email) {
		return nil, errors.New("Malformed email adres")
	}
	if badNick(nick) {
		return nil, errors.New("Malformed nickname")
	}
	parent := datastore.NewKey(c, "Unique", "UniquePlayer", 0, nil)
	emailKey := datastore.NewKey(c, "Unique", email, 0, parent)
	nickKey := datastore.NewKey(c, "Unique", nick, 0, parent)
	checkKeys := []*datastore.Key{emailKey, nickKey}
	errmap := make(map[string]int)
	errmap["nick"] = 1
	errmap["email"] = 1
	models := []interface{}{new(Email), new(Nick)}
	err := datastore.RunInTransaction(c, func(c appengine.Context) error {
		if err := datastore.GetMulti(c, checkKeys, models); err != nil {
			if multi, ok := err.(appengine.MultiError); ok {
				for i, value := range multi {
					if value == datastore.ErrNoSuchEntity {
						switch i {
						case 0:
							errmap["email"] = 0
						case 1:
							errmap["nick"] = 0
						}
						continue
					}

				}
				if errmap["nick"]+errmap["email"] == 0 {
					if _, err := datastore.PutMulti(c, checkKeys, models); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}, nil)
	if err != nil {
		return nil, err
	}
	return errmap, nil
}

func deleteUniquePlayer(c appengine.Context, email, nick string) error {
	parent := datastore.NewKey(c, "Unique", "UniquePlayer", 0, nil)
	emailKey := datastore.NewKey(c, "Unique", email, 0, parent)
	nickKey := datastore.NewKey(c, "Unique", nick, 0, parent)
	keys := []*datastore.Key{emailKey, nickKey}
	if err := datastore.DeleteMulti(c, keys); err != nil {
		return err
	}
	return nil
}

func Create(c appengine.Context, nick, email string) (string, map[string]int, error) {
	errmap, err := ValidPlayer(c, email, nick)
	if err != nil {
		return "", nil, err
	} else if errmap["email"]+errmap["nick"] > 0 {
		return "", errmap, nil
	}
	keyName, err := guid.GenUUID()
	if err != nil {
		return "", nil, err
	}
	cnt, err := counter.IncrementAndCount(c, "Player")
	if err != nil {
		return "", nil, err
	}
	playerKey := datastore.NewKey(c, "Player", keyName, 0, nil)
	player := NewPlayer()
	player.Nick = nick
	player.Access = ADMIN
	player.Email = email
	player.PlayerID = cnt
	player.Status = LIVE
	if key, err := datastore.Put(c, playerKey, player); err != nil {
		if err := deleteUniquePlayer(c, email, nick); err != nil {
			c.Errorf("error deleting unique property %s", err)
		}
		return nil, nil, err
	}
	return key.Encode(), nil, nil
}

/*func Events(c appengine.Context, playerStr, loc, cursorStr string) (*EventList, error) {
	switch loc {
	case "locals":
		loc = LOCAL
	case "globals":
		loc = GLOBAL
	default:
		return errors.New("invalid event type")
	}
	events := make([]*Event, 20, 20)
	player := new(user.Player)
	if err := Get(c, playerStr, player); err != nil {
		return nil, err
	}
	var queryKey *datastore.Key
	if loc == GLOBAL {
		if player.ClanMember == nil {
			return nil, errors.New("Player not in a clan: no global events")
		}
		queryKey = player.ClanMember.Parent()
	} else {
		queryKey = player.Key
	}
	// reset event trackers
	go func() {
		switch loc {
		case GLOBAL:
			member := new(clan.ClanMember)
			if err := datastore.Get(c, player.ClanMember, member); err != nil {
				c.Errorf("error getting member %s", err)
			}
			if member.NewGlobals > 0 {
				member.NewGlobals = 0
				if _, err := datastore.Put(c, player.ClanMember, member); err != nil {
					c.Errorf("error saving member %s", err)
				}
			}
		case LOCAL:
			if player.NewLocals > 0 {
				player.NewLocals = 0
				if _, err := datastore.Put(c, key, player); err != nil {
					c.Errorf("error saving player %s", err)
				}
			}
		}
		doneCh <- 0
	}()
	filter := fmt.Sprintf("%s =", loc)
	q := datastore.NewQuery("Event").Filter(filter, queryKey).Order("-Created").Limit(20)
	<-profileCh
	//TODO move access to central management
	//only paying users/ moderator/ admin can access more than the last 20 events either global or local
	access := user.PUSER | user.MOD | user.ADMIN
	if len(cursorStr) > 0 && player.Access&access != 0 {
		cursor, err := datastore.DecodeCursor(cursorStr)
		if err != nil {
			return nil, err
		}
		q = q.Start(cursor)
	}
	var ec int
	it := q.Run(c)
	for {
		var event Event
		_, err := it.Next(&event)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		events[ec] = &event
		ec++
	}
	newCursor, err := it.Cursor()
	if err != nil {
		return nil, err
	}
	events = events[:ec]
	list := &EventList{
		Events: events,
		Cursor: newCursor.String(),
	}
	<-doneCh
	return list, nil
}*/
