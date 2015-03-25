package player

import (
	"appengine"
	"appengine/datastore"
	"errors"
	"fmt"
	"mj0lk.be/netwars/cache"
	"mj0lk.be/netwars/counter"
	"mj0lk.be/netwars/event"
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
	BWLOWLIMIT         = 0.2
	BWHILIMIT          = 0.3
	TIMEDELIM          = "@"
	TIMETPL            = "%d@%d"
	STARTMEM           = 50
	STARTCYC           = 1000
	ACTIVEMEMMAX       = 10
	LIMIT              = 100
	THUMBSIZE          = 32
	EMAILREGEX         = `(\w[-._\w]*\w@\w[-._\w]*\w\.\w{2,3})`
	NICKREGEX          = `^[a-zA-Z0-9_.-]*$`
	MEMBER       int64 = 1 << iota
	LIEUTENANT   int64 = 1 << iota
	LEADER       int64 = 1 << iota
)

var (
	emailMatcher, _ = regexp.Compile(EMAILREGEX)
	nickMatcher, _  = regexp.Compile(NICKREGEX)
	MemberName      = map[int64]string{
		4: "Leader",
		1: "Member",
		2: "Lieutenant",
	}

	MemberType = map[string]int64{
		"Leader":     4,
		"Member":     1,
		"Lieutenant": 2,
	}
)

type Unique struct {
	Created time.Time
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
	Source     string         `json:"source"`
	Amount     int64          `json:"amount"`
	ProgramKey *datastore.Key `json:"-"`
	Usage      float64        `json:"usage" datastore:"-"`
	Yield      int64          `json:"yield" datastore:"-"`
	Key        *datastore.Key `json:"-" datastore:"-"`
	Expires    time.Time      `datastore:",noindex" json:"expires"`
	Active     bool           `json:"active"`
	Exp        int64          `json:"experience"`
	program.Program
}

type Player struct {
	EncodedKey       string                        `datastore:"-" json:"key"`
	Key              *datastore.Key                `datastore:"-" json:"-"`
	Cps              int64                         `json:"cps"`
	Aps              int64                         `json:"aps"`
	EncodedClan      string                        `datastore:"-" json:"clan_member"`
	ClanKey          *datastore.Key                `json:"-"`
	Cycles           int64                         `datastore:"-" json:"cycles"`
	Memory           int64                         `datastore:"-" json:"mem"`
	ActiveMemory     int64                         `datastore:"-" json:"active_mem"`
	CyclesUpdated    time.Time                     `datastore:"-" json:"-"`
	MemUpdated       time.Time                     `datastore:"-" json:"-"`
	ActiveMemUpdated time.Time                     `datastore:"-" json:"-"`
	Scycles          string                        `datastore:",noindex" json:"-"`
	Smem             string                        `datastore:",noindex" json:"-"`
	SactiveMem       string                        `datastore:",noindex" json:"-"`
	Bandwidth        int64                         `json:"bandwidth"`
	BandwidthUsage   float64                       `json:"bandwidth_usage"`
	Updated          time.Time                     `json:"updated"`
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
	Clan             string                        `json:"clan" datastore:"-"`
	ClanTag          string                        `json:"clan_tag"`
	MemberType       int64                         `json:"-"`
	Member           string                        `json:"member_type" datastore:"-"`
	Country          string                        `json:"country"`
	Language         string                        `json:"language"`
	Access           int64                         `json:"-"`
	Verified         bool                          `json:"-" datastore:",noindex"`
	DeviceID         string                        `json:"-" datastore:",noindex"`
	Programs         map[int64]*PlayerProgramGroup `json:"-" datastore:"-"`
	PlayerPrograms   []*PlayerProgramGroup         `json:"programs, omitempty" datastore:"-"`
	Tracker          event.Tracker                 `json:"tracker" datastore:"-"`
}

func (p Player) Range() (float64, float64) {
	lo := p.BandwidthUsage - (p.BandwidthUsage * BWLOWLIMIT)
	hi := p.BandwidthUsage + (p.BandwidthUsage * BWHILIMIT)
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
	program.Load(&pp.Program)
	if !pp.Expires.IsZero() {
		if pp.Expires.Before(time.Now()) {
			pp.Active = false
			pp.Expires = time.Time{}
			pp.Amount = 0
		}
	}
	return nil
}

func (pp *PlayerProgram) Save(c chan<- datastore.Property) error {
	if pp.ProgramKey == nil {
		return errors.New("program required")
	}
	program.Save(&pp.Program)
	return datastore.SaveStruct(pp, c)
}

func (p *Player) Load(c <-chan datastore.Property) error {
	if err := datastore.LoadStruct(p, c); err != nil {
		return err
	}
	p.Cycles, p.CyclesUpdated = timedResource(p.Scycles, 15, 50, 5e4)
	p.Memory, p.MemUpdated = timedResource(p.Smem, 15, 1, 300)
	p.ActiveMemory, p.ActiveMemUpdated = timedResource(p.SactiveMem, 60, 2, ACTIVEMEMMAX)
	if len(p.Avatar) > 0 {
		p.AvatarThumb = fmt.Sprintf("%s=s%d", p.Avatar, THUMBSIZE)
	}
	if len(p.ClanTag) > 0 {
		p.Nick = fmt.Sprintf("<%s> %s", p.ClanTag, p.Nick)
	}
	p.Member = MemberName[p.MemberType]
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
	rk := new(datastore.Key)
	if !cache.Get(c, k, rk) {
		q := datastore.NewQuery("Player").Filter("PlayerID =", id).KeysOnly().Limit(1)
		result := make([]Player, 1, 1)
		keys, err := q.GetAll(c, &result)
		if err != nil {
			return nil, err
		}
		if len(keys) > 0 {
			rk = keys[0]
			cache.Add(c, k, rk)
		}
	}
	return rk, nil
}

func Get(c appengine.Context, playerStr string, player *Player) (*datastore.Key, error) {
	playerKey, err := datastore.DecodeKey(playerStr)
	if err != nil {
		return nil, err
	}
	if !cache.Get(c, playerKey.StringID(), player) {
		if err := datastore.Get(c, playerKey, player); err != nil {
			return nil, err
		}
		cache.Add(c, playerKey.StringID(), player)
	}
	return playerKey, nil
}

func Status(c appengine.Context, playerStr string, iplayer *Player) error {
	playerKey, err := datastore.DecodeKey(playerStr)
	if err != nil {
		return err
	}
	if err := datastore.Get(c, playerKey, iplayer); err != nil {
		return err
	}
	trackerCh := make(chan event.Tracker)
	go func() {
		trackerKey := datastore.NewKey(c, "Tracker", playerKey.StringID(), 0, nil)
		tracker := new(event.Tracker)
		if err := datastore.Get(c, trackerKey, tracker); err != nil {
			c.Errorf("error retrieving tracker: %s", err)
		}
		trackerCh <- *tracker
	}()
	iplayer.Key = playerKey
	iplayer.Programs = make(map[int64]*PlayerProgramGroup)
	iplayer.BandwidthUsage = 0
	iplayer.Bandwidth = 0
	var cnt int
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
		group, mapOk = iplayer.Programs[pp.Type]
		if !mapOk {
			group = new(PlayerProgramGroup)
			group.Type = program.ProgramName[pp.Type]
			group.Programs = make([]*PlayerProgram, 0)
			iplayer.Programs[pp.Type] = group
			cnt++
		}
		if pp.Active && program.CONN&pp.Type == 0 {
			group.Usage += pp.Usage
			iplayer.BandwidthUsage += pp.Usage
		}
		if pp.Yield > 0 {
			yGroup, mapOk = iplayer.Programs[pp.EffectorTypes] // programs with yield have only one effectortype for now
			if !mapOk {
				yGroup = new(PlayerProgramGroup)
				yGroup.Type = program.ProgramName[pp.EffectorTypes]
				yGroup.Programs = make([]*PlayerProgram, 0)
				yGroup.Usage = 0.0
				iplayer.Programs[pp.EffectorTypes] = yGroup
				cnt++
			}
			yGroup.Power = true
			if pp.Active {
				iplayer.Bandwidth += pp.Yield
				yGroup.Yield += pp.Yield
			}
		}
		group.Power = true
		group.Programs = append(group.Programs, &pp)
	}
	iplayer.PlayerPrograms = make([]*PlayerProgramGroup, cnt)
	for cType, cGroup := range iplayer.Programs {
		cnt--
		iplayer.PlayerPrograms[cnt] = cGroup
		ignoreTypes := program.CONN | program.INF
		if cType == ignoreTypes&cType {
			continue
		}
		usage := float64(iplayer.Bandwidth) - iplayer.BandwidthUsage
		ppusage := float64(cGroup.Yield) - cGroup.Usage
		if usage <= 0.0 || ppusage <= 0.0 {
			cGroup.Power = false
		}
	}
	iplayer.Tracker = <-trackerCh
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
	uniqueEmail := &Unique{time.Now()}
	uniqueNick := &Unique{time.Now()}
	models := []interface{}{uniqueEmail, uniqueNick}
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
	trackerKey := datastore.NewKey(c, "Tracker", keyName, 0, nil)
	tracker := new(event.Tracker)
	player := NewPlayer()
	player.Nick = nick
	player.Access = ADMIN
	player.Email = email
	player.PlayerID = cnt
	player.Status = LIVE
	storeKeys := []*datastore.Key{playerKey, trackerKey}
	models := []interface{}{player, tracker}
	if _, err = datastore.PutMulti(c, storeKeys, models); err != nil {
		if uerr := deleteUniquePlayer(c, email, nick); uerr != nil {
			c.Errorf("error deleting unique property %s", uerr)
		}
		return "", nil, err
	}
	return playerKey.Encode(), nil, nil
}

func Tracker(c appengine.Context, playerStr, clanStr string) (event.Tracker, error) {
	playerKey, err := datastore.DecodeKey(playerStr)
	if err != nil {
		return event.Tracker{}, err
	}
	//postFix := "local"
	trackerKey := datastore.NewKey(c, "Tracker", playerKey.StringID(), 0, nil)
	if clanStr != "" {
		//postFix = "global"
		clanKey, err := datastore.DecodeKey(clanStr)
		if err != nil {
			return event.Tracker{}, err
		}
		trackerKey = datastore.NewKey(c, "Tracker", playerKey.StringID(), 0, clanKey)
	}
	//memKey := playerKey.StringID() + postFix
	tracker := new(event.Tracker)
	//if !cache.Get(c, memKey, tracker) {
	if err := datastore.Get(c, trackerKey, tracker); err != nil {
		return event.Tracker{}, err
	}
	//	cache.Add(c, memKey, tracker)
	//}
	return *tracker, nil
}

func Events(c appengine.Context, playerStr, loc, cursorStr string) (event.EventList, error) {
	events := make([]*event.Event, 20, 20)
	iplayer := new(Player)
	playerKey, err := Get(c, playerStr, iplayer)
	if err != nil {
		return event.EventList{}, err
	}
	var queryKey *datastore.Key
	if loc == event.GLOBAL {
		if iplayer.ClanKey == nil {
			return event.EventList{}, errors.New("Player not in a clan: no global events")
		}
		queryKey = iplayer.ClanKey
	} else {
		queryKey = playerKey
	}
	// reset event trackers
	doneCh := make(chan int)
	go func() {
		var clanStr string
		var trackerKey *datastore.Key
		switch loc {
		case event.GLOBAL:
			clanStr = iplayer.ClanKey.Encode()
			trackerKey = datastore.NewKey(c, "Tracker", playerKey.StringID(), 0, iplayer.ClanKey)
		case event.LOCAL:
			clanStr = ""
			trackerKey = datastore.NewKey(c, "Tracker", playerKey.StringID(), 0, nil)
		}
		tracker, err := Tracker(c, playerStr, clanStr)
		if err != nil {
			c.Errorf("error fetching tracker %s", err)
		}
		if tracker.EventCount > 0 {
			tracker.EventCount = 0
			if _, err := datastore.Put(c, trackerKey, tracker); err != nil {
				c.Errorf("error saving global tracker %s", err)
			}
		}
		doneCh <- 0
	}()
	filter := fmt.Sprintf("%s =", loc)
	q := datastore.NewQuery("Event").Filter(filter, queryKey).Order("-Created").Limit(20)
	//TODO move access to central management
	//only paying users/ moderator/ admin can access more than the last 20 events either global or local
	access := PUSER | MOD | ADMIN
	if len(cursorStr) > 0 && iplayer.Access&access != 0 {
		cursor, err := datastore.DecodeCursor(cursorStr)
		if err != nil {
			return event.EventList{}, err
		}
		q = q.Start(cursor)
	}
	var ec int
	it := q.Run(c)
	for {
		var e event.Event
		_, err := it.Next(&e)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return event.EventList{}, err
		}
		if loc == event.GLOBAL && e.Player.Equal(playerKey) {
			continue
		}
		events[ec] = &e
		ec++
	}
	newCursor, err := it.Cursor()
	if err != nil {
		return event.EventList{}, err
	}
	events = events[:ec]
	list := event.EventList{
		Events: events,
		Cursor: newCursor.String(),
	}
	<-doneCh
	return list, nil
}
