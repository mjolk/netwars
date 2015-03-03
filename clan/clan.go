package clan

import (
	"appengine"
	"appengine/blobstore"
	"appengine/datastore"
	"appengine/image"
	"errors"
	"fmt"
	"math"
	"mj0lk.be/netwars/cache"
	"mj0lk.be/netwars/counter"
	"mj0lk.be/netwars/event"
	"mj0lk.be/netwars/guid"
	"mj0lk.be/netwars/player"
	"regexp"
	"strconv"
	"time"
)

//TODO implement delete clan

const (
	RANGEUP       = 0.3
	RANGEDOWN     = 0.2
	MAXMEMBER     = 6
	MAXLEADERSHIP = 3
	MAXINVITES    = 3
	THUMBSIZE     = 32
	CLANNAMEREGEX = `^([a-zA-Z0-9]){3,18}$`
	CLANTAGREGEX  = `^([a-zA-Z0-9]){3,4}$`
)

var PlayerAlreadyInvitedError = errors.New("Player already invited \n")
var ClanMemberTypeError = errors.New("Need to be Lieutenant or Leader to perform this action")
var ClanMemberError = errors.New("Player not in a clan")

var clanNameRegex, _ = regexp.Compile(CLANNAMEREGEX)
var clanTagRegex, _ = regexp.Compile(CLANTAGREGEX)

type Clan struct {
	Tag            string `json:"clan_tag"`
	Name           string `json:"clan_name"`
	ClanID         int64
	BandwidthUsage float64           `json:"bw_usage"`
	Cps            int64             `json: "clan_cps"`
	AmountPlayers  int64             `datastore:",noindex"`
	Created        time.Time         `datastore:",noindex"`
	Creator        *datastore.Key    `datastore:",noindex" json:"-"`
	AvatarKey      appengine.BlobKey `datastore:",noindex" json:"-"`
	Avatar         string            `datastore:",noindex" json:"avatar"`
	AvatarThumb    string            `datastore:"-" json:"avatar_thumb"`
	Members        []*player.Player  `datastore:"-" json:"clan_members"`
	Message        string            `datastore:",noindex" json:"message"`
	Profile        string            `datastore:",noindex" json:"profile"`
	Site           string            `datastore:",noindex" json:"clan_site"`
	Description    string            `datastore:",noindex" json:"description"`
}

type Invite struct {
	Player        *datastore.Key
	PlayerName    string
	Expires       time.Time
	Clan          *datastore.Key
	ClanName      string
	InvitedBy     *datastore.Key
	InvitedByName string
	Invited       time.Time
	Key           *datastore.Key `datastore:"-"`
}

//parent clan
type ClanConnection struct {
	Key     *datastore.Key `datastore:"-"`
	Player  *datastore.Key `datastore:",noindex"` //clan leadership initiating connection, blame ...
	Target  *datastore.Key //target clan
	Created time.Time
	Expires time.Time //lock expiration
	Active  bool      //
	Closed  time.Time
}

type MessageUpdate struct {
	Pkey    string
	Content string
}

func (cl *Clan) Load(c <-chan datastore.Property) error {
	if err := datastore.LoadStruct(cl, c); err != nil {
		return err
	}
	if len(cl.Avatar) > 0 {
		cl.AvatarThumb = fmt.Sprintf("%s=s%d", cl.Avatar, THUMBSIZE)
	}
	return nil
}

func (cl *Clan) Save(c chan<- datastore.Property) error {
	return datastore.SaveStruct(cl, c)
}

func (c *Clan) Range() (float64, float64) {
	lo := c.BandwidthUsage - (c.BandwidthUsage * RANGEDOWN)
	hi := c.BandwidthUsage + (c.BandwidthUsage * RANGEUP)
	return lo, hi
}

func isNotInRange(a, d *Clan) bool {
	lo, hi := a.Range()
	if d.BandwidthUsage > hi || d.BandwidthUsage < lo {
		return true
	}
	return false
}

func NewInvite(c appengine.Context, clan *Clan, invitee, player *player.Player) *Invite {
	now := time.Now()
	expires := now.AddDate(0, 0, 2)
	invite := &Invite{
		Expires:       expires,
		Clan:          player.ClanKey,
		ClanName:      clan.Name,
		Player:        invitee.Key,
		PlayerName:    invitee.Nick,
		InvitedBy:     player.Key,
		InvitedByName: player.Nick,
		Invited:       now,
	}
	return invite
}

func InvitesForPlayer(c appengine.Context, playerStr string) ([]*Invite, error) {
	playerKey, err := datastore.DecodeKey(playerStr)
	if err != nil {
		return nil, err
	}
	invMemKey := playerKey.StringID() + "Invite"
	invites := make([]*Invite, 0)
	if !cache.Get(c, invMemKey, invites) {
		q := datastore.NewQuery("Invite").Filter("Player =", playerKey).Filter("Expires >", time.Now())
		for it := q.Run(c); ; {
			var invite Invite
			key, err := it.Next(&invite)
			if err == datastore.Done {
				break
			}
			if err != nil {
				return nil, err
			}
			invite.Key = key
			invites = append(invites, &invite)
		}
		cache.Add(c, invMemKey, invites)
	}
	return invites, nil

}

func Get(c appengine.Context, clanKey *datastore.Key, team *Clan) error {
	cMemKey := clanKey.StringID() + "Clan"
	if !cache.Get(c, cMemKey, team) {
		if err := datastore.Get(c, clanKey, team); err != nil {
			return err
		}
		cache.Add(c, cMemKey, team)
	}
	return nil
}

func Status(c appengine.Context, clanStr string, team *Clan) error {
	clanKey, err := datastore.DecodeKey(clanStr)
	if err != nil {
		return err
	}
	if err := datastore.Get(c, clanKey, team); err != nil {
		return err
	}
	pCnt := team.AmountPlayers
	team.BandwidthUsage = 0
	team.Cps = 0
	team.Members = make([]*player.Player, pCnt, pCnt)
	//TODO  todo project
	var cnt int
	q := datastore.NewQuery("Player").Filter("ClanKey =", clanKey)
	for it := q.Run(c); ; {
		var member player.Player
		key, err := it.Next(&member)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return err
		}
		c.Debugf("count players: %d \n", cnt)
		member.Key = key
		team.Members[cnt] = &member
		team.BandwidthUsage += member.BandwidthUsage
		team.Cps += member.Cps
		cnt++
	}
	return nil
}

func cancelInvite(c appengine.Context, playerStr, inviteStr string) error {
	return nil
}

func activeConnectionsForClan(c appengine.Context, clan *datastore.Key) ([]*ClanConnection, error) {
	connections := make([]*ClanConnection, 3, 3)
	q := datastore.NewQuery("ClanConnection").Ancestor(clan).Filter("Active =", true)
	var cCount int64
	for it := q.Run(c); ; {
		var connection ClanConnection
		_, err := it.Next(&connection)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		connections[cCount] = &connection
		cCount++
	}
	connections = connections[:cCount]
	return connections, nil
}

func DisConnect(c appengine.Context, playerStr, connStr string) error {
	connKey, err := datastore.DecodeKey(connStr)
	playerKey, err := datastore.DecodeKey(playerStr)
	if err != nil {
		return err
	}
	connection := new(ClanConnection)
	iplayer := new(player.Player)
	if err := datastore.GetMulti(c, []*datastore.Key{playerKey, connKey}, []interface{}{iplayer, connection}); err != nil {
		return err
	}
	if iplayer.ClanKey == nil {
		return ClanMemberError
	}
	if iplayer.MemberType < player.LIEUTENANT {
		return ClanMemberTypeError
	}
	if connection.Active {
		if connection.Expires.Before(time.Now()) {
			connection.Closed = time.Now()
			connection.Active = false
		} else {
			toExpire := connection.Expires.Sub(time.Now())
			return errors.New(fmt.Sprintf("\n connection didn't expire yet %s", toExpire))
		}
		if _, err := datastore.Put(c, connKey, connection); err != nil {
			return err
		}
		e := event.Event{
			Created:    time.Now(),
			Player:     playerKey,
			PlayerName: iplayer.Nick,
			PlayerID:   iplayer.PlayerID,
			Direction:  event.OUT,
			EventType:  "Clan",
			Clan:       iplayer.ClanKey,
			Target:     connection.Target,
			Action:     "DisConnect",
		}
		e1 := event.Event{
			Created:   time.Now(),
			EventType: "Clan",
			Direction: event.IN,
			Clan:      connection.Target,
			Target:    iplayer.ClanKey,
			Action:    "DisConnect",
		}
		if err := event.Send(c, []event.Event{e, e1}, DisConnectEvent); err != nil {
			return err
		}
	} else {
		return errors.New("Connection already inactive")
	}
	return nil
}

func DisConnectEvent(c appengine.Context, ev []event.Event) error {
	attackEvent := ev[0]
	defenseEvent := ev[1]
	attackingClan := new(Clan)
	defendingClan := new(Clan)
	cntCh := make(chan int64, 2)
	go event.NewEventID(c, cntCh)
	if err := datastore.GetMulti(c, []*datastore.Key{attackEvent.Clan, defenseEvent.Clan},
		[]interface{}{attackingClan, defendingClan}); err != nil {
		return err
	}
	attackEvent.TargetName = defendingClan.Name
	attackEvent.TargetID = defendingClan.ClanID
	defenseEvent.TargetName = attackingClan.Name
	defenseEvent.TargetID = attackingClan.ClanID
	attackEvent.ClanName = attackingClan.Name
	attackEvent.ClanID = attackingClan.ClanID
	defenseEvent.ClanID = defendingClan.ClanID
	defenseEvent.ClanName = defendingClan.Name
	aGuid, err := guid.GenUUID()
	dGuid, err := guid.GenUUID()
	if err != nil {
		return err
	}
	attackKey := datastore.NewKey(c, "Event", aGuid, 0, nil)
	defenseKey := datastore.NewKey(c, "Event", dGuid, 0, nil)
	clanNotify := make(chan int)
	go attackEvent.Notify(c, clanNotify)
	go defenseEvent.Notify(c, clanNotify)
	id := <-cntCh
	attackEvent.ID = id
	defenseEvent.ID = id
	keys := []*datastore.Key{attackKey, defenseKey}
	models := []interface{}{&attackEvent, &defenseEvent}
	if _, err := datastore.PutMulti(c, keys, models); err != nil {
		return err
	}
	for i := 0; i < 2; i++ {
		<-clanNotify
	}
	return nil
}

func Connect(c appengine.Context, playerStr, target string) error {
	playerKey, err := datastore.DecodeKey(playerStr)
	if err != nil {
		return err
	}
	iplayer := new(player.Player)
	if err := datastore.Get(c, playerKey, iplayer); err != nil {
		return err
	}
	if iplayer.ClanKey == nil {
		return ClanMemberError
	}
	if iplayer.MemberType < player.LIEUTENANT {
		return ClanMemberTypeError
	}
	defendingClanKey, err := datastore.DecodeKey(target)
	if err != nil {
		return err
	}
	clanCh := make(chan int)
	at := new(Clan)
	go func() {
		if err := Status(c, iplayer.ClanKey.Encode(), at); err != nil {
			c.Errorf("error status clan: %s", err)
		}
		clanCh <- 0
	}()
	dt := new(Clan)
	go func() {
		if err := Status(c, target, dt); err != nil {
			c.Errorf("error status clan: %s", err)
		}
		clanCh <- 0
	}()
	attConns, err := activeConnectionsForClan(c, iplayer.ClanKey)
	if err != nil {
		return err
	}
	attConnCount := len(attConns)
	if attConnCount > 0 {
		if attConnCount < 3 {
			for _, conn := range attConns {
				if conn.Target.Equal(defendingClanKey) {
					//already active war with this clan
					return errors.New("Already @ war with this clan")
				}
			}
		} else {
			return errors.New("Already max active wars")
		}
	}
	for i := 0; i < 2; i++ {
		<-clanCh
	}
	if isNotInRange(at, dt) {
		return errors.New("Target Clan is not in range")
	}
	expires := time.Now().AddDate(0, 0, 1)
	connKeyGuid, err := guid.GenUUID()
	if err != nil {
		return err
	}
	newConnKey := datastore.NewKey(c, "ClanConnection", connKeyGuid, 0, iplayer.ClanKey)
	newConnection := &ClanConnection{
		Key:     newConnKey,
		Player:  playerKey,
		Target:  defendingClanKey,
		Created: time.Now(),
		Expires: expires,
		Active:  true,
	}
	if _, err := datastore.Put(c, newConnKey, newConnection); err != nil {
		return err
	}
	created := time.Now()
	e := event.Event{
		Created:    created,
		Player:     playerKey,
		PlayerName: iplayer.Nick,
		PlayerID:   iplayer.PlayerID,
		Direction:  event.OUT,
		EventType:  "Clan",
		Clan:       iplayer.ClanKey,
		ClanName:   at.Name,
		ClanID:     at.ClanID,
		Target:     defendingClanKey,
		TargetName: dt.Name,
		TargetID:   dt.ClanID,
		Expires:    newConnection.Expires,
		Action:     "Connect",
	}
	e1 := event.Event{
		Created:    created,
		EventType:  "Clan",
		Direction:  event.IN,
		Clan:       defendingClanKey,
		ClanName:   dt.Name,
		ClanID:     dt.ClanID,
		Target:     iplayer.ClanKey,
		TargetName: at.Name,
		TargetID:   at.ClanID,
		Expires:    newConnection.Expires,
		Action:     "Connect",
	}
	if err := event.Send(c, []event.Event{e, e1}, ClanEvent); err != nil {
		return err
	}
	return nil
}

func leaveInterval(c appengine.Context, playerKey *datastore.Key) (bool, string, error) {
	qt := time.Unix(time.Now().Unix()-int64(time.Duration(24*time.Hour).Seconds()), 0)
	q := datastore.NewQuery("Event").Filter("Player =", playerKey).Filter("Action =", "Leave").
		Filter("Created >", qt).Order("-Created").Limit(1)
	var e event.Event
	var cnt int
	for it := q.Run(c); ; {
		_, err := it.Next(&e)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return false, "", err
		}
		cnt++
	}
	if cnt > 0 {
		toJoin := time.Duration(time.Duration(24*time.Hour).Nanoseconds() - time.Now().Sub(e.Created).Nanoseconds())
		return true, toJoin.String(), nil
	}
	return false, "", nil
}

func Join(c appengine.Context, playerStr, inviteStr string) error {
	inviteKey, err := datastore.DecodeKey(inviteStr)
	playerKey, err := datastore.DecodeKey(playerStr)
	if err != nil {
		return err
	}
	interv, wait, err := leaveInterval(c, playerKey)
	if err != nil {
		return err
	}
	if interv {
		return errors.New(fmt.Sprintf("Wait %s before joining a new clan", wait))
	}
	invite := new(Invite)
	if err := datastore.Get(c, inviteKey, invite); err != nil {
		return err
	}
	//should already be removed
	if invite.Expires.Before(time.Now()) {
		return errors.New("Invite expired")
	}
	options := new(datastore.TransactionOptions)
	options.XG = true
	return datastore.RunInTransaction(c, func(c appengine.Context) error {
		iplayer := new(player.Player)
		team := new(Clan)
		keys := []*datastore.Key{playerKey, invite.Clan}
		models := []interface{}{iplayer, team}
		if err := datastore.GetMulti(c, keys, models); err != nil {
			return err
		}
		if iplayer.ClanKey != nil {
			return errors.New("Already member of a clan")
		}
		if team.AmountPlayers > MAXMEMBER-1 {
			return errors.New("Full Clan")
		}
		iplayer.ClanKey = invite.Clan
		iplayer.ClanTag = team.Tag
		iplayer.Clan = team.Name
		iplayer.MemberType = player.MEMBER
		team.AmountPlayers++
		if _, err := datastore.PutMulti(c, keys, models); err != nil {
			return err
		}
		e := event.Event{
			Created:    time.Now(),
			Player:     playerKey,
			EventType:  "Clan",
			Clan:       invite.Clan,
			Direction:  event.IN,
			Action:     "Join",
			PlayerName: iplayer.Nick,
			PlayerID:   iplayer.PlayerID,
			ClanName:   team.Name,
			ClanID:     team.ClanID,
			TargetName: team.Name,
			TargetID:   team.ClanID,
		}
		if err := event.Send(c, []event.Event{e}, SingleClanEvent); err != nil {
			return err
		}
		return nil
	}, options)
}

func EmailInvite(c appengine.Context, clanStr, playerStr, email string) error {
	return nil
}

func inviteBarrier(c appengine.Context, clanKey *datastore.Key) error {
	q := datastore.NewQuery("Invite").Filter("Expires >", time.Now()).
		Filter("Clan =", clanKey)
	count, err := q.Count(c)
	if err != nil {
		return err
	}
	if count >= MAXINVITES {
		//already enough invites sent
		//wait till some expire
		return errors.New("Too many invites")
	}
	return nil
}

func InvitePlayer(c appengine.Context, playerStr, inviteeIDStr string) error {
	playerKey, err := datastore.DecodeKey(playerStr)
	inviteeID, err := strconv.ParseInt(inviteeIDStr, 10, 64)
	if err != nil {
		return err
	}
	inviteeKey, err := player.KeyByID(c, inviteeID)
	if err != nil {
		return err
	}
	iplayer := new(player.Player)
	invitedPlayer := new(player.Player)
	if err := datastore.GetMulti(c, []*datastore.Key{playerKey, inviteeKey},
		[]interface{}{iplayer, invitedPlayer}); err != nil {
		return err
	}
	iplayer.Key = playerKey
	invitedPlayer.Key = inviteeKey
	if iplayer.MemberType < player.LIEUTENANT {
		return ClanMemberTypeError
	}
	if iplayer.ClanKey == nil || invitedPlayer.ClanKey != nil {
		return ClanMemberError
	}
	if err := inviteBarrier(c, iplayer.ClanKey); err != nil {
		return err
	}
	team := new(Clan)
	if err := datastore.Get(c, iplayer.ClanKey, team); err != nil {
		return err
	}
	if team.AmountPlayers >= MAXMEMBER {
		return errors.New("Already full clan")
	}
	inviteStr := fmt.Sprintf("%d%d", team.ClanID, invitedPlayer.PlayerID)
	inviteKey := datastore.NewKey(c, "Invite", inviteStr, 0, nil)
	invite := NewInvite(c, team, invitedPlayer, iplayer)
	if err := datastore.Get(c, inviteKey, invite); err != nil {
		if err == datastore.ErrNoSuchEntity {
			//never invited this player before: do nothing
		} else if err != nil {
			return err
		}
	} else if invite.Expires.After(time.Now()) {
		return PlayerAlreadyInvitedError
	} else {
		invite.Expires = time.Now().AddDate(0, 0, 2)
		invite.Invited = time.Now()
		invite.Player = inviteeKey
		invite.InvitedBy = playerKey
		invite.Clan = iplayer.ClanKey
		invite.ClanName = team.Name
		invite.InvitedByName = iplayer.Nick
		invite.PlayerName = invitedPlayer.Nick
	}
	if _, err := datastore.Put(c, inviteKey, invite); err != nil {
		return err
	}
	now := time.Now()
	e := event.Event{
		Created:    now,
		Player:     playerKey,
		Direction:  event.OUT,
		EventType:  "Clan",
		Clan:       iplayer.ClanKey,
		Expires:    invite.Expires,
		Target:     inviteeKey,
		Action:     "Invite",
		PlayerName: iplayer.Nick,
		PlayerID:   iplayer.PlayerID,
		TargetName: invitedPlayer.Nick,
		TargetID:   invitedPlayer.PlayerID,
		ClanName:   team.Name,
		ClanID:     team.ClanID,
	}
	e1 := event.Event{
		Created:    now,
		Direction:  event.IN,
		Player:     inviteeKey,
		EventType:  "Clan",
		Expires:    invite.Expires,
		Target:     playerKey,
		Action:     "Invite",
		PlayerName: invitedPlayer.Nick,
		PlayerID:   invitedPlayer.PlayerID,
		TargetName: iplayer.Nick,
		TargetID:   iplayer.PlayerID,
		ClanName:   team.Name,
		ClanID:     team.ClanID,
	}
	if err := event.Send(c, []event.Event{e, e1}, ClanEvent); err != nil {
		return err
	}
	return nil
}

func ClanEvent(c appengine.Context, em []event.Event) error {
	initEvent := em[0]
	subjEvent := em[1]
	cntCh := make(chan int64, 2)
	go event.NewEventID(c, cntCh)
	receiverGuid, err := guid.GenUUID()
	senderGuid, err := guid.GenUUID()
	if err != nil {
		return err
	}
	senderKey := datastore.NewKey(c, "Event", senderGuid, 0, nil)
	receiverKey := datastore.NewKey(c, "Event", receiverGuid, 0, nil)
	notifyCh := make(chan int, 2)
	go initEvent.Notify(c, notifyCh)
	go subjEvent.Notify(c, notifyCh)
	id := <-cntCh
	initEvent.ID = id
	subjEvent.ID = id
	keys := []*datastore.Key{senderKey, receiverKey}
	models := []interface{}{&initEvent, &subjEvent}
	if _, err := datastore.PutMulti(c, keys, models); err != nil {
		return err
	}
	for i := 0; i < 2; i++ {
		<-notifyCh
	}
	return nil
}

func removeClan(c appengine.Context, name, tag string) error {
	parent := datastore.NewKey(c, "Unique", "UniqueClan", 0, nil)
	clanNameKey := datastore.NewKey(c, "Unique", name, 0, parent)
	clanTagKey := datastore.NewKey(c, "Unique", tag, 0, parent)
	delKeys := []*datastore.Key{clanNameKey, clanTagKey}
	if err := datastore.DeleteMulti(c, delKeys); err != nil {
		return err
	}
	return nil
}

func badName(name string) bool {
	if clanNameRegex.MatchString(name) == true {
		return false
	}
	return true
}

func badTag(tag string) bool {
	if clanTagRegex.MatchString(tag) == true {
		return false
	}
	return true
}

func validClan(c appengine.Context, name, tag string) (map[string]int, error) {
	if badName(name) {
		return nil, errors.New("Malformed clan name")
	}
	if badTag(tag) {
		return nil, errors.New("Malformed tag name")
	}
	parent := datastore.NewKey(c, "Unique", "UniqueClan", 0, nil)
	clanNameKey := datastore.NewKey(c, "Unique", name, 0, parent)
	clanTagKey := datastore.NewKey(c, "Unique", tag, 0, parent)
	checkKeys := []*datastore.Key{clanNameKey, clanTagKey}
	errmap := make(map[string]int)
	errmap["clan_name"] = 1
	errmap["clan_tag"] = 1
	clanNameStrct := &player.Unique{time.Now()}
	clanTagStruct := &player.Unique{time.Now()}
	models := []interface{}{clanNameStrct, clanTagStruct}
	err := datastore.RunInTransaction(c, func(c appengine.Context) error {
		if err := datastore.GetMulti(c, checkKeys, models); err != nil {
			if multi, ok := err.(appengine.MultiError); ok {
				for i, value := range multi {
					if value == datastore.ErrNoSuchEntity {
						switch i {
						case 0:
							errmap["clan_name"] = 0
						case 1:
							errmap["clan_tag"] = 0
						}
						continue
					}

				}
				if errmap["clan_name"]+errmap["clan_tag"] == 0 {
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

func Create(c appengine.Context, playerStr, clanName, tag string) (string, map[string]int, error) {
	errmap, err := validClan(c, clanName, tag)
	if err != nil {
		return "", nil, err
	} else if errmap["clan_name"]+errmap["tag"] > 0 {
		return "", errmap, nil
	}
	playerKey, err := datastore.DecodeKey(playerStr)
	if err != nil {
		return "", nil, err
	}
	interv, wait, err := leaveInterval(c, playerKey)
	if err != nil {
		return "", nil, err
	}
	if interv {
		return "", nil, errors.New(fmt.Sprintf("Wait %s before creating a new clan", wait))
	}
	iplayer := new(player.Player)
	clanNr, err := counter.IncrementAndCount(c, "Clan")
	if err != nil {
		return "", nil, err
	}
	clanGuid, err := guid.GenUUID()
	if err != nil {
		return "", nil, err
	}
	clanKey := datastore.NewKey(c, "Clan", clanGuid, 0, nil)
	tracker := new(event.Tracker)
	trackerKey := datastore.NewKey(c, "Tracker", playerKey.StringID(), 0, clanKey)
	options := new(datastore.TransactionOptions)
	options.XG = true
	txErr := datastore.RunInTransaction(c, func(c appengine.Context) error {
		if err := datastore.Get(c, playerKey, iplayer); err != nil {
			return err
		}
		//just to make sure
		if iplayer.ClanKey != nil {
			return errors.New("Already in a clan")
		}
		iplayer.ClanKey = clanKey
		iplayer.ClanTag = tag
		iplayer.Clan = clanName
		iplayer.MemberType = player.LEADER
		clan := &Clan{
			Name:           clanName,
			Tag:            tag,
			ClanID:         clanNr,
			Created:        time.Now(),
			Creator:        playerKey,
			AmountPlayers:  1,
			BandwidthUsage: iplayer.BandwidthUsage,
		}
		keys := []*datastore.Key{playerKey, clanKey, trackerKey}
		models := []interface{}{iplayer, clan, tracker}
		if _, err := datastore.PutMulti(c, keys, models); err != nil {
			return err
		}
		e := event.Event{
			Created:    time.Now(),
			Direction:  event.IN,
			Player:     playerKey,
			EventType:  "Clan",
			Clan:       clanKey,
			ClanName:   clanName,
			ClanID:     clanNr,
			Action:     "Create",
			PlayerName: iplayer.Nick,
			PlayerID:   iplayer.PlayerID,
		}
		if err := event.Send(c, []event.Event{e}, SingleClanEvent); err != nil {
			return err
		}
		return nil
	}, options)
	if txErr != nil {
		if err := removeClan(c, clanName, tag); err != nil {
			errorMsg := fmt.Sprintf("errors creating clan: %s, %s", txErr, err)
			return "", nil, errors.New(errorMsg)
		}
		return "", nil, txErr
	}
	return clanGuid, nil, nil
}

func SingleClanEvent(c appengine.Context, em []event.Event) error {
	initEvent := em[0]
	cntCh := make(chan int64, 1)
	go event.NewEventID(c, cntCh)
	eventKeyString, err := guid.GenUUID()
	if err != nil {
		return err
	}
	eventKey := datastore.NewKey(c, "Event", eventKeyString, 0, nil)
	//TODO change to message code
	notifyCh := make(chan int)
	go initEvent.Notify(c, notifyCh)
	initEvent.ID = <-cntCh
	if _, err := datastore.Put(c, eventKey, &initEvent); err != nil {
		return err
	}
	<-notifyCh
	return nil
}

func Leave(c appengine.Context, playerStr string) error {
	playerKey, err := datastore.DecodeKey(playerStr)
	if err != nil {
		return err
	}
	options := new(datastore.TransactionOptions)
	options.XG = true
	return datastore.RunInTransaction(c, func(c appengine.Context) error {
		iplayer := new(player.Player)
		if err := datastore.Get(c, playerKey, iplayer); err != nil {
			return err
		}
		if iplayer.ClanKey == nil {
			return ClanMemberError
		}
		if iplayer.MemberType == player.LEADER {
			return ClanMemberTypeError
		}
		clan := new(Clan)
		if err := datastore.Get(c, iplayer.ClanKey, clan); err != nil {
			return err
		}
		clan.AmountPlayers--
		if iplayer.Cps > 0 {
			clan.Cps -= int64(math.Ceil(float64(iplayer.Cps) / 3))
		}
		clanKey := iplayer.ClanKey
		iplayer.ClanKey = nil
		iplayer.Clan = ""
		iplayer.ClanTag = ""
		iplayer.MemberType = 0
		keys := []*datastore.Key{playerKey, clanKey}
		models := []interface{}{iplayer, clan}
		if _, err := datastore.PutMulti(c, keys, models); err != nil {
			return err
		}
		e := event.Event{
			Created:    time.Now(),
			Player:     playerKey,
			Direction:  event.OUT,
			EventType:  "Clan",
			Clan:       clanKey,
			Action:     "Leave",
			PlayerName: iplayer.Nick,
			PlayerID:   iplayer.PlayerID,
			ClanName:   clan.Name,
			ClanID:     clan.ClanID,
		}
		if err := event.Send(c, []event.Event{e}, LeaveEvent); err != nil {
			return err
		}
		return nil
	}, options)
}

func LeaveEvent(c appengine.Context, em []event.Event) error {
	initEvent := em[0]
	cntCh := make(chan int64, 1)
	go event.NewEventID(c, cntCh)
	eventKeyString, err := guid.GenUUID()
	if err != nil {
		return err
	}
	eventKey := datastore.NewKey(c, "Event", eventKeyString, 0, nil)
	//TODO change to message code
	notifyCh := make(chan int, 1)
	go initEvent.Notify(c, notifyCh)
	initEvent.ID = <-cntCh
	if _, err := datastore.Put(c, eventKey, &initEvent); err != nil {
		return err
	}
	<-notifyCh
	//wait for notifications to finish : otherwise leaving player doesnt get local event.
	trackerKey := datastore.NewKey(c, "Tracker", initEvent.Player.StringID(), 0, initEvent.Clan)
	if err := datastore.Delete(c, trackerKey); err != nil {
		return err
	}
	return nil
}

func checkLeaderShip(c appengine.Context, clanKey *datastore.Key) (int64, error) {
	count, err := datastore.NewQuery("Player").Filter("ClanKey =", clanKey).Filter("MemberType >", 1).Count(c)
	if err != nil {
		return 0, err
	}
	return int64(count), nil
}

func PromoteOrDemote(c appengine.Context, playerStr, target, rank string) error {
	playerKey, err := datastore.DecodeKey(playerStr)
	promoteID, err := strconv.ParseInt(target, 10, 64)
	if err != nil {
		return err
	}
	rk, ok := player.MemberType[rank]
	if !ok {
		return errors.New("Unknown rank")
	}
	promoteKey, err := player.KeyByID(c, promoteID)
	if err != nil {
		return err
	}
	iplayer := new(player.Player)
	promotePlayer := new(player.Player)
	if err := datastore.GetMulti(c, []*datastore.Key{playerKey, promoteKey},
		[]interface{}{iplayer, promotePlayer}); err != nil {
		return err
	}
	if iplayer.ClanKey == nil || promotePlayer.ClanKey == nil {
		return ClanMemberError
	}
	if !promotePlayer.ClanKey.Equal(iplayer.ClanKey) {
		return errors.New("Illegal operation")
	}
	if iplayer.MemberType < player.LIEUTENANT {
		return ClanMemberTypeError
	}
	if rk == player.LEADER {
		if iplayer.MemberType != player.LEADER {
			return errors.New("Need to be Clan leader to change leadership")
		}
		if promotePlayer.MemberType != player.LIEUTENANT {
			return errors.New("Can only promote Lieutenant to Leader")
		}
	}
	if promotePlayer.MemberType < player.LIEUTENANT {
		count, err := checkLeaderShip(c, iplayer.ClanKey)
		if err != nil {
			return err
		}
		if count == MAXLEADERSHIP {
			return errors.New("Need to demote someone first")
		}
	}
	if promotePlayer.MemberType == rk {
		return errors.New(fmt.Sprintf("Player already has rank : %s", rank))
	}
	var action string
	if rk < promotePlayer.MemberType {
		action = "Demote"
	} else if rk > promotePlayer.MemberType {
		action = "Promote"
	}
	paction := action
	promotePlayer.MemberType = rk
	keys := []*datastore.Key{promoteKey}
	models := []interface{}{promotePlayer}
	if rk == 4 {
		iplayer.MemberType = player.LIEUTENANT
		keys = append(keys, playerKey)
		models = append(models, iplayer)
		paction = "Demote"
	}
	if _, err := datastore.PutMulti(c, keys, models); err != nil {
		return err
	}
	e := event.Event{
		Created:    time.Now(),
		Player:     playerKey,
		EventType:  "Clan",
		Clan:       promotePlayer.ClanKey,
		Target:     promoteKey,
		TargetName: promotePlayer.Nick,
		TargetID:   promotePlayer.PlayerID,
		PlayerName: iplayer.Nick,
		PlayerID:   iplayer.PlayerID,
		Action:     paction,
		Direction:  event.OUT,
	}
	//no need to provide clan -> same clan and otherwise double global event
	e1 := event.Event{
		Created:    time.Now(),
		Player:     promoteKey,
		EventType:  "Clan",
		PlayerName: promotePlayer.Nick,
		PlayerID:   promotePlayer.PlayerID,
		Target:     playerKey,
		TargetName: iplayer.Nick,
		TargetID:   iplayer.PlayerID,
		Action:     action,
		Direction:  event.IN,
	}
	if err := event.Send(c, []event.Event{e, e1}, ClanEvent); err != nil {
		return err
	}
	return nil
}

func UpdateMessage(c appengine.Context, update *MessageUpdate) error {
	playerKey, err := datastore.DecodeKey(update.Pkey)
	if err != nil {
		return err
	}
	iplayer := new(player.Player)
	if err := datastore.Get(c, playerKey, iplayer); err != nil {
		return err
	}
	if iplayer.ClanKey == nil {
		return ClanMemberError
	}
	if iplayer.MemberType < player.LIEUTENANT {
		return ClanMemberTypeError
	}
	team := new(Clan)
	if err := datastore.Get(c, iplayer.ClanKey, team); err != nil {
		return err
	}
	team.Message = update.Content
	if _, err := datastore.Put(c, iplayer.ClanKey, team); err != nil {
		return err
	}
	e := event.Event{
		Created:    time.Now(),
		Player:     playerKey,
		EventType:  "Clan",
		Clan:       iplayer.ClanKey,
		ClanName:   team.Name,
		ClanID:     team.ClanID,
		Action:     "Message",
		PlayerName: iplayer.Nick,
		PlayerID:   iplayer.PlayerID,
		Direction:  event.IN,
	}
	if err := event.Send(c, []event.Event{e}, SingleClanEvent); err != nil {
		return err
	}
	return nil
}

func Delete(c appengine.Context, pkey string) error {
	return nil
}

func Kick(c appengine.Context, playerStr, target string) error {
	playerKey, err := datastore.DecodeKey(playerStr)
	kickedPlayerID, err := strconv.ParseInt(target, 10, 64)
	if err != nil {
		return err
	}
	kickedPlayerKey, err := player.KeyByID(c, kickedPlayerID)
	if err != nil {
		return err
	}
	iplayer := new(player.Player)
	kickedPlayer := new(player.Player)
	if err := datastore.GetMulti(c, []*datastore.Key{playerKey, kickedPlayerKey},
		[]interface{}{iplayer, kickedPlayer}); err != nil {
		return err
	}
	if iplayer.ClanKey == nil || kickedPlayer.ClanKey == nil {
		return ClanMemberError
	}
	if !kickedPlayer.ClanKey.Equal(iplayer.ClanKey) {
		return errors.New("Illegal operation")
	}
	if iplayer.MemberType < player.LIEUTENANT {
		return ClanMemberTypeError
	}
	if kickedPlayer.MemberType > player.MEMBER {
		errors.New("Need to demote player first")
	}
	kickedPlayer.MemberType = 0
	kickedPlayer.Clan = ""
	kickedPlayer.ClanTag = ""
	kickedPlayer.ClanKey = nil
	if _, err := datastore.Put(c, kickedPlayerKey, kickedPlayer); err != nil {
		return err
	}

	e := event.Event{
		Created:    time.Now(),
		Player:     playerKey,
		EventType:  "Clan",
		Clan:       iplayer.ClanKey,
		Target:     kickedPlayerKey,
		PlayerName: iplayer.Nick,
		PlayerID:   iplayer.PlayerID,
		TargetName: kickedPlayer.Nick,
		TargetID:   kickedPlayer.PlayerID,
		Action:     "Kick",
		Direction:  event.OUT,
	}
	e1 := event.Event{
		Created:    time.Now(),
		Player:     kickedPlayerKey,
		PlayerName: kickedPlayer.Nick,
		PlayerID:   kickedPlayer.PlayerID,
		EventType:  "Clan",
		Action:     "Kick",
		Target:     iplayer.ClanKey,
		TargetName: iplayer.Nick,
		TargetID:   iplayer.PlayerID,
		Direction:  event.IN,
	}
	if err := event.Send(c, []event.Event{e, e1}, KickEvent); err != nil {
		return err
	}
	return nil
}

func KickEvent(c appengine.Context, em []event.Event) error {
	initEvent := em[0]
	subjEvent := em[1]
	cntCh := make(chan int64, 2)
	go event.NewEventID(c, cntCh)
	receiverGuid, err := guid.GenUUID()
	senderGuid, err := guid.GenUUID()
	if err != nil {
		return err
	}
	senderKey := datastore.NewKey(c, "Event", senderGuid, 0, nil)
	receiverKey := datastore.NewKey(c, "Event", receiverGuid, 0, nil)
	notifyCh := make(chan int, 2)
	go initEvent.Notify(c, notifyCh)
	go subjEvent.Notify(c, notifyCh)
	id := <-cntCh
	initEvent.ID = id
	subjEvent.ID = id
	keys := []*datastore.Key{senderKey, receiverKey}
	models := []interface{}{&initEvent, &subjEvent}
	if _, err := datastore.PutMulti(c, keys, models); err != nil {
		return err
	}
	for i := 0; i < 2; i++ {
		<-notifyCh
	}
	//wait for notifications to finish : otherwise kicked player doesnt get local event.
	trackerKey := datastore.NewKey(c, "Tracker", subjEvent.Player.StringID(), 0, subjEvent.Clan)
	if err := datastore.Delete(c, trackerKey); err != nil {
		return err
	}
	return nil
}

// TODO check last update of image to avoid too much changing and uploading
func UpdateAvatar(c appengine.Context, playerStr string, img *blobstore.BlobInfo) error {
	playerKey, err := datastore.DecodeKey(playerStr)
	if err != nil {
		return err
	}
	iplayer := new(player.Player)
	if err := datastore.Get(c, playerKey, iplayer); err != nil {
		return err
	}
	if iplayer.ClanKey != nil {
		clan := new(Clan)
		if err := datastore.Get(c, iplayer.ClanKey, clan); err != nil {
			return err
		}
		if iplayer.MemberType < player.LIEUTENANT {
			return ClanMemberTypeError
		}
		if len(clan.Avatar) > 0 {
			if err := image.DeleteServingURL(c, clan.AvatarKey); err != nil {
				return err
			}
		}
		imgURL, err := image.ServingURL(c, img.BlobKey, nil)
		if err != nil {
			return err
		}
		clan.AvatarKey = img.BlobKey
		clan.Avatar = imgURL.String()
		if _, err := datastore.Put(c, iplayer.ClanKey, clan); err != nil {
			return err
		}
	} else {
		return errors.New("Not a member")
	}
	return nil
}
