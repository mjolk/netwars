package clan

import (
	"appengine"
	"appengine/blobstore"
	"appengine/datastore"
	"appengine/image"
	"bytes"
	"encoding/gob"
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
	LIMIT         = 100
)

var PlayerAlreadyInvitedError = errors.New("Player already invited \n")
var ClanMemberTypeError = errors.New("Need to be Lieutenant or Leader to perform this action")
var ClanMemberError = errors.New("Player not in a clan")

var clanNameRegex, _ = regexp.Compile(CLANNAMEREGEX)
var clanTagRegex, _ = regexp.Compile(CLANTAGREGEX)

type SendKey struct {
	Key string `json:"key"` //connection key
}

type SendID struct {
	ID int64 `json:"id"` //player id
}

type Creation struct {
	Tag  string `json:"tag"`
	Name string `json:"name"`
}

type Promotion struct {
	PlayerID int64 `json:"player_id"`
	Rank     int64 `json:"rank"`
}

type Clan struct {
	Tag            string            `json:"clan_tag"`
	Name           string            `json:"clan_name"`
	ID             int64             `json:"clan_id"`
	BandwidthUsage float64           `json:"bandwidth_usage"`
	Cps            int64             `json: "clan_cps"`
	AmountPlayers  int64             `datastore:",noindex" json:"amount_players"`
	Created        time.Time         `datastore:",noindex" json:"created_on"`
	Creator        *datastore.Key    `datastore:",noindex" json:"-"`
	AvatarKey      appengine.BlobKey `datastore:",noindex" json:"-"`
	Avatar         string            `datastore:",noindex" json:"avatar"`
	AvatarThumb    string            `datastore:"-" json:"avatar_thumb"`
	Members        []player.Player   `datastore:"-" json:"clan_members"`
	Message        string            `datastore:",noindex" json:"message"`
	Profile        string            `datastore:",noindex" json:"profile"`
	Site           string            `datastore:",noindex" json:"clan_site"`
	Description    string            `datastore:",noindex" json:"description"`
	Wars           []ClanConnection  `datastore:"-" json:"wars"`
	Connections    []byte            `json:"-"`
}

type ClanList struct {
	Cursor string `json:"cursor"`
	Clans  []Clan `json:"clans"`
}

type Invite struct {
	Player        *datastore.Key `json:"-"`
	PlayerName    string         `json:"player_name"`
	Expires       time.Time      `json:"expires"`
	Clan          *datastore.Key `json:"-"`
	ClanName      string         `json:"clan_name"`
	InvitedBy     *datastore.Key `json:"-"`
	InvitedByName string         `json:"invited_name"`
	Invited       time.Time      `json:"invited_on"`
	DbKey         *datastore.Key `datastore:"-" json:"-"`
	EncodedKey    string         `datastore:"-" json:"key"`
}

//parent clan
type ClanConnection struct {
	Player     int64     `datastore:",noindex" json:"-"` //clan leadership initiating connection, blame ...
	Source     int64     `json:"attacker"`
	SourceName string    `json:"attacker_name"`
	TargetName string    `json:"defender_name"`
	Target     int64     `json:"defender"` //target clan
	Created    time.Time `json:"created_on"`
	Expires    time.Time `json:"expires"` //lock expiration
	Expired    bool      `json:"expired"`
}

type MessageUpdate struct {
	Content string
}

func (cl *Clan) Load(c <-chan datastore.Property) error {
	if err := datastore.LoadStruct(cl, c); err != nil {
		return err
	}
	if len(cl.Avatar) > 0 {
		cl.AvatarThumb = fmt.Sprintf("%s=s%d", cl.Avatar, THUMBSIZE)
	}
	if len(cl.Connections) > 0 {
		var wBytes = bytes.NewBuffer(cl.Connections)
		if err := gob.NewDecoder(wBytes).Decode(&cl.Wars); err != nil {
			return err
		}
	}
	return nil
}

func (cl *Clan) Save(c chan<- datastore.Property) error {
	if len(cl.Wars) > 0 {
		var wBytes bytes.Buffer
		if err := gob.NewEncoder(&wBytes).Encode(&cl.Wars); err != nil {
			return err
		}
		cl.Connections = wBytes.Bytes()
	}
	return datastore.SaveStruct(cl, c)
}

func (c *Clan) Range() (float64, float64) {
	lo := c.BandwidthUsage - (c.BandwidthUsage * RANGEDOWN)
	hi := c.BandwidthUsage + (c.BandwidthUsage * RANGEUP)
	return lo, hi
}

func (c *Clan) ConnectionForID(target int64) ClanConnection {
	for _, conn := range c.Wars {
		if conn.Target == target {
			return conn
		}
	}
	return ClanConnection{}
}

func (c *Clan) RemoveConnection(target int64) {
	for k, conn := range c.Wars {
		if conn.Target == target {
			c.Wars = append(c.Wars[:k], c.Wars[k+1:]...)
			return
		}
	}
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
		Player:        invitee.DbKey,
		PlayerName:    invitee.Nick,
		InvitedBy:     player.DbKey,
		InvitedByName: player.Nick,
		Invited:       now,
	}
	return invite
}

func InvitesForPlayer(c appengine.Context, playerStr string) ([]Invite, error) {
	playerKey, err := datastore.DecodeKey(playerStr)
	if err != nil {
		return nil, err
	}
	invMemKey := playerKey.StringID() + "Invite"
	invites := make([]Invite, 0)
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
			invite.DbKey = key
			invite.EncodedKey = key.Encode()
			invites = append(invites, invite)
		}
		cache.Add(c, invMemKey, invites)
	}
	return invites, nil

}

func KeyByID(c appengine.Context, id int64) (*datastore.Key, error) {
	k := fmt.Sprintf("%d", id)
	rk := new(datastore.Key)
	if !cache.Get(c, k, rk) {
		q := datastore.NewQuery("Clan").Filter("ID =", id).KeysOnly().Limit(1)
		result := make([]Clan, 1, 1)
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

func PublicStatus(c appengine.Context, pkeyStr, clanIdStr string, team *Clan) error {
	clanId, err := strconv.ParseInt(clanIdStr, 10, 64)
	if err != nil {
		return err
	}
	clanKey, err := KeyByID(c, clanId)
	if err != nil {
		return err
	}
	if err := Get(c, clanKey, team); err != nil {
		return err
	}
	return loadClanMembers(c, clanKey, team)
}

func loadClanMembers(c appengine.Context, clanKey *datastore.Key, iclan *Clan) error {
	pCnt := iclan.AmountPlayers
	iclan.BandwidthUsage = 0
	iclan.Cps = 0
	iclan.Members = make([]player.Player, pCnt, pCnt)
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
		member.DbKey = key
		iclan.Members[cnt] = member
		iclan.BandwidthUsage += member.BandwidthUsage
		iclan.Cps += member.Cps
		cnt++
	}
	return nil
}

func Status(c appengine.Context, playerStr string, team *Clan) error {
	iplayer := new(player.Player)
	_, err := player.Get(c, playerStr, iplayer)
	if err != nil {
		return err
	}
	if err := datastore.Get(c, iplayer.ClanKey, team); err != nil {
		return err
	}
	return loadClanMembers(c, iplayer.ClanKey, team)
}

func List(c appengine.Context, playerStr, rangeStr, cursor string) (ClanList, error) {
	q := datastore.NewQuery("Clan").Order("-BandwidthUsage").Limit(LIMIT)
	team := new(Clan)
	iplayer := new(player.Player)
	_, err := player.Get(c, playerStr, iplayer)
	if err != nil {
		return ClanList{}, err
	}
	if err := Get(c, iplayer.ClanKey, team); err != nil {
		return ClanList{}, err
	}
	if len(rangeStr) > 0 {
		rnge, err := strconv.ParseInt(rangeStr, 10, 64)
		if err != nil {
			return ClanList{}, err
		}
		if rnge > 0 {
			rangeLo, rangeHi := team.Range()
			q = q.Filter("BandwidthUsage >", rangeLo).
				Filter("BandwidthUsage <", rangeHi)
		}
	}
	clans := make([]Clan, LIMIT, LIMIT)
	if len(cursor) > 0 {
		cur, err := datastore.DecodeCursor(cursor)
		if err != nil {
			return ClanList{}, err
		}
		q = q.Start(cur)
	}
	t := q.Run(c)
	var cnt int
	for {
		var team Clan
		key, err := t.Next(&team)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return ClanList{}, err
		}
		if !iplayer.ClanKey.Equal(key) {
			clans[cnt] = team
			cnt++
		}
	}
	clans = clans[:cnt]
	newCur, err := t.Cursor()
	if err != nil {
		return ClanList{}, err
	}
	list := ClanList{
		Cursor: newCur.String(),
		Clans:  clans,
	}
	return list, nil
}

func CancelInvite(c appengine.Context, playerStr, inviteStr string) error {
	inviteKey, err := datastore.DecodeKey(inviteStr)
	if err != nil {
		return err
	}
	return datastore.RunInTransaction(c, func(c appengine.Context) error {
		invite := new(Invite)
		if err := datastore.Get(c, inviteKey, invite); err != nil {
			return err
		}
		created := time.Now()
		e1 := &event.Event{
			Player:     invite.Player,
			PlayerName: invite.PlayerName,
			Created:    created,
			EventType:  "Invite",
			Direction:  event.IN,
			Target:     invite.InvitedBy,
			TargetName: invite.InvitedByName,
			Action:     "Cancel",
		}
		e2 := &event.Event{
			Player:     invite.InvitedBy,
			PlayerName: invite.InvitedByName,
			Clan:       invite.Clan,
			ClanName:   invite.ClanName,
			Created:    created,
			EventType:  "Invite",
			Direction:  event.OUT,
			Target:     invite.Player,
			TargetName: invite.PlayerName,
			Action:     "Cancel",
		}
		if err := datastore.Delete(c, inviteKey); err != nil {
			return err
		}
		if err := event.Send(c, []*event.Event{e1, e2}, event.Func); err != nil {
			return err
		}
		return nil
	}, nil)
}

func DisConnect(c appengine.Context, playerStr string, target int64) error {
	iplayer := new(player.Player)
	playerKey, err := player.Get(c, playerStr, iplayer)
	if err != nil {
		return err
	}
	if iplayer.ClanKey == nil {
		return ClanMemberError
	}
	if iplayer.MemberType < player.LIEUTENANT {
		return ClanMemberTypeError
	}
	DclanKey, err := KeyByID(c, target)
	if err != nil {
		return err
	}
	options := new(datastore.TransactionOptions)
	options.XG = true
	return datastore.RunInTransaction(c, func(c appengine.Context) error {
		Aclan := new(Clan)
		Dclan := new(Clan)
		clanCh := make(chan int)
		go func() {
			if err := Get(c, iplayer.ClanKey, Aclan); err != nil {
				c.Errorf("error fetching clan for disconnect %s", err)
			}
			clanCh <- 1
		}()
		if err := Get(c, DclanKey, Dclan); err != nil {
			return err
		}
		connection := Aclan.ConnectionForID(target)
		if connection.Target == 0 {
			return errors.New("no connection found")
		}
		if connection.Expires.Before(time.Now()) {
			toExpire := connection.Expires.Sub(time.Now())
			return errors.New(fmt.Sprintf("\n connection didn't expire yet %s", toExpire))
		} else {

			Aclan.RemoveConnection(target)
			Dclan.RemoveConnection(target)
			if _, err := datastore.PutMulti(c, []*datastore.Key{iplayer.ClanKey, DclanKey},
				[]interface{}{Aclan, Dclan}); err != nil {
				return err
			}
		}

		e := &event.Event{
			Created:    time.Now(),
			Player:     playerKey,
			PlayerName: iplayer.Nick,
			PlayerID:   iplayer.ID,
			Direction:  event.OUT,
			EventType:  "Clan",
			Clan:       iplayer.ClanKey,
			Target:     DclanKey,
			Action:     "DisConnect",
		}
		e1 := &event.Event{
			Created:   time.Now(),
			EventType: "Clan",
			Direction: event.IN,
			Clan:      DclanKey,
			Target:    iplayer.ClanKey,
			Action:    "DisConnect",
		}
		if err := event.Send(c, []*event.Event{e, e1}, event.Func); err != nil {
			return err
		}
		return nil
	}, options)
}

func Connect(c appengine.Context, playerStr string, target int64) error {
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
	defendingClanKey, err := KeyByID(c, target)
	if err != nil {
		return err
	}
	clanCh := make(chan int)
	at := new(Clan)
	go func() {
		if err := Status(c, playerStr, at); err != nil {
			c.Errorf("error status clan: %s", err)
		}
		clanCh <- 0
	}()
	dt := new(Clan)
	go func() {
		nbrStr := strconv.FormatInt(target, 10)
		if err := PublicStatus(c, "", nbrStr, dt); err != nil {
			c.Errorf("error status clan: %s", err)
		}
		clanCh <- 0
	}()
	for i := 0; i < 2; i++ {
		<-clanCh
	}
	attConnCount := len(at.Wars)
	if attConnCount > 0 {
		if attConnCount < 3 {
			for _, conn := range at.Wars {
				if conn.Target == dt.ID {
					//already active war with this clan
					return errors.New("Already @ war with this clan")
				}
			}
		} else {
			return errors.New("Already max active wars")
		}
	}
	if isNotInRange(at, dt) {
		return errors.New("Target Clan is not in range")
	}
	expires := time.Now().AddDate(0, 0, 1)
	newConnection := ClanConnection{
		Player:     iplayer.ID,
		Source:     at.ID,
		SourceName: at.Name,
		Target:     dt.ID,
		TargetName: dt.Name,
		Created:    time.Now(),
		Expires:    expires,
	}
	at.Wars = append(at.Wars, newConnection)
	dt.Wars = append(dt.Wars, newConnection)
	if _, err := datastore.PutMulti(c, []*datastore.Key{iplayer.ClanKey, defendingClanKey},
		[]interface{}{at, dt}); err == nil {
		created := time.Now()
		e := &event.Event{
			Created:    created,
			Player:     playerKey,
			PlayerName: iplayer.Nick,
			PlayerID:   iplayer.ID,
			Direction:  event.OUT,
			EventType:  "Clan",
			Clan:       iplayer.ClanKey,
			ClanName:   dt.Name,
			ClanID:     dt.ID,
			Target:     defendingClanKey,
			TargetName: dt.Name,
			TargetID:   dt.ID,
			Expires:    newConnection.Expires,
			Action:     "Connect",
		}
		e1 := &event.Event{
			Created:    created,
			EventType:  "Clan",
			Direction:  event.IN,
			Clan:       defendingClanKey,
			ClanName:   dt.Name,
			ClanID:     dt.ID,
			Target:     iplayer.ClanKey,
			TargetName: at.Name,
			TargetID:   at.ID,
			Expires:    newConnection.Expires,
			Action:     "Connect",
		}
		if err := event.Send(c, []*event.Event{e, e1}, event.Func); err != nil {
			return err
		}
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
	if err != nil {
		return err
	}
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
		tracker := new(event.Tracker)
		trackerKey := datastore.NewKey(c, "Tracker", playerKey.StringID(), 0, invite.Clan)
		keys = append(keys, trackerKey)
		models = append(models, tracker)
		iplayer.ClanKey = invite.Clan
		iplayer.ClanTag = team.Tag
		iplayer.Clan = team.Name
		iplayer.MemberType = player.MEMBER
		team.AmountPlayers++
		if _, err := datastore.PutMulti(c, keys, models); err != nil {
			return err
		}
		e := &event.Event{
			Created:    time.Now(),
			Player:     playerKey,
			EventType:  "Clan",
			Clan:       invite.Clan,
			Direction:  event.IN,
			Action:     "Join",
			PlayerName: iplayer.Nick,
			PlayerID:   iplayer.ID,
			ClanName:   team.Name,
			ClanID:     team.ID,
			TargetName: team.Name,
			TargetID:   team.ID,
		}
		if err := event.Send(c, []*event.Event{e}, event.Func); err != nil {
			return err
		}
		return nil
	}, options)
}

//TODO add email invite logic
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

func InvitePlayer(c appengine.Context, playerStr string, inviteeID int64) error {
	playerKey, err := datastore.DecodeKey(playerStr)
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
	iplayer.DbKey = playerKey
	invitedPlayer.DbKey = inviteeKey
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
	inviteStr := fmt.Sprintf("%d%d", team.ID, invitedPlayer.ID)
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
	e := &event.Event{
		Created:    now,
		Player:     playerKey,
		Direction:  event.OUT,
		EventType:  "Clan",
		Clan:       iplayer.ClanKey,
		Expires:    invite.Expires,
		Target:     inviteeKey,
		Action:     "Invite",
		PlayerName: iplayer.Nick,
		PlayerID:   iplayer.ID,
		TargetName: invitedPlayer.Nick,
		TargetID:   invitedPlayer.ID,
		ClanName:   team.Name,
		ClanID:     team.ID,
	}
	e1 := &event.Event{
		Created:    now,
		Direction:  event.IN,
		Player:     inviteeKey,
		EventType:  "Clan",
		Expires:    invite.Expires,
		Target:     playerKey,
		Action:     "Invite",
		PlayerName: invitedPlayer.Nick,
		PlayerID:   invitedPlayer.ID,
		TargetName: iplayer.Nick,
		TargetID:   iplayer.ID,
		ClanName:   team.Name,
		ClanID:     team.ID,
	}
	if err := event.Send(c, []*event.Event{e, e1}, event.Func); err != nil {
		return err
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
			ID:             clanNr,
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
		e := &event.Event{
			Created:    time.Now(),
			Direction:  event.IN,
			Player:     playerKey,
			EventType:  "Clan",
			Clan:       clanKey,
			ClanName:   clanName,
			ClanID:     clanNr,
			Action:     "Create",
			PlayerName: iplayer.Nick,
			PlayerID:   iplayer.ID,
		}
		if err := event.Send(c, []*event.Event{e}, event.Func); err != nil {
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
		e := &event.Event{
			Created:    time.Now(),
			Player:     playerKey,
			Direction:  event.OUT,
			EventType:  "Clan",
			Clan:       clanKey,
			Action:     "Leave",
			PlayerName: iplayer.Nick,
			PlayerID:   iplayer.ID,
			ClanName:   clan.Name,
			ClanID:     clan.ID,
		}
		if err := event.Send(c, []*event.Event{e}, func(c appengine.Context, evs []*event.Event) error {
			if err := event.Func(c, evs); err != nil {
				return err
			}
			le := evs[0]
			trackerKey := datastore.NewKey(c, "Tracker", le.Player.StringID(), 0, le.Clan)
			if err := datastore.Delete(c, trackerKey); err != nil {
				return err
			}
			return nil
		}); err != nil {
			return err
		}
		return nil
	}, options)
}

func checkLeaderShip(c appengine.Context, clanKey *datastore.Key) (int64, error) {
	count, err := datastore.NewQuery("Player").Filter("ClanKey =", clanKey).Filter("MemberType >", 1).Count(c)
	if err != nil {
		return 0, err
	}
	return int64(count), nil
}

func PromoteOrDemote(c appengine.Context, playerStr string, promoteID, rk int64) error {
	playerKey, err := datastore.DecodeKey(playerStr)
	if err != nil {
		return err
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
		return errors.New(fmt.Sprintf("Player already has rank : %s", rk))
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
	e := &event.Event{
		Created:    time.Now(),
		Player:     playerKey,
		EventType:  "Clan",
		Clan:       promotePlayer.ClanKey,
		Target:     promoteKey,
		TargetName: promotePlayer.Nick,
		TargetID:   promotePlayer.ID,
		PlayerName: iplayer.Nick,
		PlayerID:   iplayer.ID,
		Action:     paction,
		Direction:  event.OUT,
	}
	//no need to provide clan -> same clan and otherwise double global event
	e1 := &event.Event{
		Created:    time.Now(),
		Player:     promoteKey,
		EventType:  "Clan",
		PlayerName: promotePlayer.Nick,
		PlayerID:   promotePlayer.ID,
		Target:     playerKey,
		TargetName: iplayer.Nick,
		TargetID:   iplayer.ID,
		Action:     action,
		Direction:  event.IN,
	}
	if err := event.Send(c, []*event.Event{e, e1}, func(c appengine.Context, e []*event.Event) error {
		aEvent := e[0]
		bEvent := e[1]
		team := new(Clan)
		if err := Get(c, aEvent.Clan, team); err != nil {
			c.Errorf("error getting clan %s", err)
		}
		aEvent.ClanName = team.Name
		aEvent.ClanID = team.ID
		bEvent.ClanName = team.Name
		bEvent.ClanID = team.ID
		return event.Func(c, e)
	}); err != nil {
		return err
	}
	return nil
}

func UpdateMessage(c appengine.Context, playerKeyStr string, update *MessageUpdate) error {
	playerKey, err := datastore.DecodeKey(playerKeyStr)
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
	e := &event.Event{
		Created:    time.Now(),
		Player:     playerKey,
		EventType:  "Clan",
		Clan:       iplayer.ClanKey,
		ClanName:   team.Name,
		ClanID:     team.ID,
		Action:     "Message",
		PlayerName: iplayer.Nick,
		PlayerID:   iplayer.ID,
		Direction:  event.IN,
	}
	if err := event.Send(c, []*event.Event{e}, event.Func); err != nil {
		return err
	}
	return nil
}

func Delete(c appengine.Context, pkey string) error {
	return nil
}

func Kick(c appengine.Context, playerStr string, kickedPlayerID int64) error {
	playerKey, err := datastore.DecodeKey(playerStr)
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

	e := &event.Event{
		Created:    time.Now(),
		Player:     playerKey,
		EventType:  "Clan",
		Clan:       iplayer.ClanKey,
		Target:     kickedPlayerKey,
		PlayerName: iplayer.Nick,
		PlayerID:   iplayer.ID,
		TargetName: kickedPlayer.Nick,
		TargetID:   kickedPlayer.ID,
		Action:     "Kick",
		Direction:  event.OUT,
	}
	e1 := &event.Event{
		Created:    time.Now(),
		Player:     kickedPlayerKey,
		PlayerName: kickedPlayer.Nick,
		PlayerID:   kickedPlayer.ID,
		EventType:  "Clan",
		Action:     "Kick",
		Target:     iplayer.ClanKey,
		TargetName: iplayer.Nick,
		TargetID:   iplayer.ID,
		Direction:  event.IN,
	}
	if err := event.Send(c, []*event.Event{e, e1}, func(c appengine.Context, evs []*event.Event) error {
		e1 := evs[0]
		e2 := evs[1]
		cl := new(Clan)
		if err := Get(c, e1.Clan, cl); err != nil {
			c.Errorf("error getting clan %s", err)
			return err
		}
		e1.ClanName = cl.Name
		e1.ClanID = cl.ID
		e2.ClanName = cl.Name
		e2.ClanID = cl.ID
		if err := event.Func(c, evs); err != nil {
			return err
		}
		trackerKey := datastore.NewKey(c, "Tracker", e2.Player.StringID(), 0, e2.Clan)
		if err := datastore.Delete(c, trackerKey); err != nil {
			return err
		}
		return nil
	}); err != nil {
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
