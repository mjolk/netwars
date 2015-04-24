package player

import (
	"appengine"
	"appengine/blobstore"
	"appengine/datastore"
	"appengine/image"
	"fmt"
	"mj0lk.be/netwars/cache"
	"mj0lk.be/netwars/event"
	"strconv"
	"time"
)

const (
	TIMELAYOUT       = "2006-Jan-02"
	USERTYPE   int64 = 1024
	REGULAR    int64 = USERTYPE << iota
	PUSER      int64 = USERTYPE << iota
	PCLAN      int64 = USERTYPE << iota
	MOD        int64 = USERTYPE << iota
	ADMIN      int64 = USERTYPE << iota
	DEAD       int64 = 0x2800
	LIVE       int64 = 0x2800 << 1
	SUSPENDED  int64 = 0x2800 << 2
)

var PlayerStatusName = map[int64]string{
	DEAD:      "Killed",
	LIVE:      "Live",
	SUSPENDED: "Suspended",
}

var PlayerStatus = map[string]int64{
	"Killed":    DEAD,
	"Live":      LIVE,
	"Suspended": SUSPENDED,
}

var PlayerTypeName = map[int64]string{
	REGULAR: "Free Player",
	PUSER:   "Subscribed Player",
	PCLAN:   "Subscribed Clan",
	MOD:     "Moderator",
	ADMIN:   "Administrator",
}

var PlayerType = map[string]int64{
	"Free Player":       REGULAR,
	"Subscribed Player": PUSER,
	"Subscribed Clan":   PCLAN,
	"Moderator":         MOD,
	"Administrator":     ADMIN,
}

type PublicPlayer struct {
	DbKey            *datastore.Key                `datastore:"-" json:"-"`
	Cps              int64                         `json:"-"`
	Aps              int64                         `json:"aps"`
	EncodedClan      string                        `datastore:"-" json:"-"`
	ClanKey          *datastore.Key                `json:"-"`
	Cycles           int64                         `datastore:"-" json:"-"`
	Memory           int64                         `datastore:"-" json:"-"`
	ActiveMemory     int64                         `datastore:"-" json:"-"`
	CyclesUpdated    time.Time                     `datastore:"-" json:"-"`
	MemUpdated       time.Time                     `datastore:"-" json:"-"`
	ActiveMemUpdated time.Time                     `datastore:"-" json:"-"`
	Scycles          string                        `datastore:",noindex" json:"-"`
	Smem             string                        `datastore:",noindex" json:"-"`
	SactiveMem       string                        `datastore:",noindex" json:"-"`
	Bandwidth        int64                         `json:"-"`
	BandwidthUsage   float64                       `json:"bandwidth_usage"`
	Updated          time.Time                     `json:"-"`
	Created          time.Time                     `json:"-"`
	Email            string                        `json:"-"`
	Nick             string                        `json:"nick"`
	Name             string                        `json:"-"`
	Address          string                        `json:"-"`
	Signature        string                        `json:"-"`
	Birthday         time.Time                     `json:"-"`
	AvatarKey        appengine.BlobKey             `json:"-"`
	Avatar           string                        `json:"avatar"`
	AvatarThumb      string                        `datastore:"-" json:"avatar_thumb"`
	ID               int64                         `json:"player_id"`
	Status           int64                         `json:"-"`
	StatusName       string                        `json:"status" datastore:"-"`
	AccessName       string                        `json:"-" datastore:"-"`
	Clan             string                        `json:"clan" datastore:"-"`
	ClanTag          string                        `json:"clan_tag"`
	MemberType       int64                         `json:"-"`
	Member           string                        `json:"member_type" datastore:"-"`
	Country          string                        `json:"-"`
	Language         string                        `json:"-"`
	Access           int64                         `json:"-"`
	Verified         bool                          `json:"-" datastore:",noindex"`
	DeviceID         string                        `json:"-" datastore:",noindex"`
	Programs         map[int64]*PlayerProgramGroup `json:"-" datastore:"-"`
	PlayerPrograms   []*PlayerProgramGroup         `json:"-" datastore:"-"`
	Tracker          event.Tracker                 `json:"-" datastore:"-"`
	Pass             []byte                        `json:"-" datastore:"-"`
}

func (p *PublicPlayer) Load(c <-chan datastore.Property) error {
	if err := datastore.LoadStruct(p, c); err != nil {
		return err
	}
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

func (p *PublicPlayer) Save(c chan<- datastore.Property) error {
	return datastore.SaveStruct(p, c)
}

type Profile struct {
	Nick           string  `json:"nick"`
	BandwidthUsage float64 `json:"bandwidth_usage"`
	Status         int64   `json:"-"`
	StatusName     string  `json:"status"`
	Avatar         string  `json:"-"`
	AvatarThumb    string  `json:"avatar"`
	ID             int64   `json:"player_id"`
	ClanTag        string  `json:"clan_tag"`
	Access         int64   `json:"-"`
	AccessName     string  `json:"type"`
}

type ProfileUpdate struct {
	Name      string `json:"name"`
	Birthday  string `json:"birthday"`
	Country   string `json:"country"`
	Language  string `json:"language"`
	Address   string `json:"address"`
	Signature string `json:"signature"`
}

type PlayerList struct {
	Cursor  string    `json:"cursor"`
	Players []Profile `json:"players"`
}

//parent profile
type PlayerSubscription struct {
	Created          time.Time
	Expires          time.Time
	Activated        bool
	PaymentMethod    string
	Origin           *datastore.Key
	SubscriptionType int64
}

func (p *Profile) Load(c <-chan datastore.Property) error {
	if err := datastore.LoadStruct(p, c); err != nil {
		return err
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

func (p *Profile) Save(c chan<- datastore.Property) error {
	return datastore.SaveStruct(p, c)
}

func GetPublic(c appengine.Context, playerStr string, player *PublicPlayer) (*datastore.Key, error) {
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

func Public(c appengine.Context, playerStr string, playerIdStr string, iplayer *PublicPlayer) error {
	playerId, err := strconv.ParseInt(playerIdStr, 10, 64)
	if err != nil {
		return err
	}
	playerKey, err := KeyByID(c, playerId)
	if err != nil {
		return err
	}
	if _, err := GetPublic(c, playerKey.Encode(), iplayer); err != nil {
		return err
	}
	return nil

}

func List(c appengine.Context, pkeyStr, rangeStr, cursor string) (PlayerList, error) {
	q := datastore.NewQuery("Player").Project("Nick", "BandwidthUsage", "Status",
		"Avatar", "ID", "ClanTag", "Access").Order("-BandwidthUsage").Limit(LIMIT)
	iplayer := new(Player)
	playerKey, err := Get(c, pkeyStr, iplayer)
	if err != nil {
		return PlayerList{}, err
	}
	if len(rangeStr) > 0 {
		rnge, err := strconv.ParseInt(rangeStr, 10, 64)
		if err != nil {
			return PlayerList{}, err
		}
		if rnge > 0 {
			rangeLo, rangeHi := iplayer.Range()
			q = q.Filter("BandwidthUsage >", rangeLo).
				Filter("BandwidthUsage <", rangeHi)
		}
	}
	profiles := make([]Profile, LIMIT, LIMIT)
	if len(cursor) > 0 {
		cur, err := datastore.DecodeCursor(cursor)
		if err != nil {
			return PlayerList{}, err
		}
		q = q.Start(cur)
	}
	t := q.Run(c)
	var cnt int
	for {
		var profile Profile
		key, err := t.Next(&profile)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return PlayerList{}, err
		}
		if !playerKey.Equal(key) {
			profiles[cnt] = profile
			cnt++
		}
	}
	profiles = profiles[:cnt]
	newCur, err := t.Cursor()
	if err != nil {
		return PlayerList{}, err
	}
	list := PlayerList{
		Cursor:  newCur.String(),
		Players: profiles,
	}
	return list, nil
}

func UpdateProfile(c appengine.Context, playerStr string, update ProfileUpdate) error {
	iplayer := new(Player)
	playerKey, err := Get(c, playerStr, iplayer)
	if err != nil {
		return err
	}
	if len(update.Birthday) > 0 {
		bd, err := time.Parse(TIMELAYOUT, update.Birthday)
		if err != nil {
			return err
		}
		iplayer.Birthday = bd
	}
	iplayer.Name = update.Name
	iplayer.Country = update.Country
	iplayer.Language = update.Language
	iplayer.Address = update.Address
	iplayer.Signature = update.Signature
	if _, err := datastore.Put(c, playerKey, iplayer); err != nil {
		return err
	}
	return nil
}

// TODO check last update of image to avoid too much changing and uploading
func UpdateAvatar(c appengine.Context, playerStr string, img *blobstore.BlobInfo) error {
	iplayer := new(Player)
	playerKey, err := Get(c, playerStr, iplayer)
	if err != nil {
		return err
	}
	if len(iplayer.Avatar) > 0 {
		if err := image.DeleteServingURL(c, iplayer.AvatarKey); err != nil {
			return err
		}
	}
	imgURL, err := image.ServingURL(c, img.BlobKey, nil)
	if err != nil {
		return err
	}
	iplayer.AvatarKey = img.BlobKey
	iplayer.Avatar = imgURL.String()
	if _, err := datastore.Put(c, playerKey, iplayer); err != nil {
		return err
	}
	return nil
}
