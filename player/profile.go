package user

import (
	"appengine"
	"appengine/blobstore"
	"appengine/datastore"
	"appengine/image"
	"fmt"
	"mj0lk.be/netwars/cache"
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

type Profile struct {
	Nick           string  `json:"nick"`
	BandwidthUsage float64 `json:"bandwidth_usage"`
	Status         int64   `json:"-"`
	StatusName     string  `json:"status"`
	Avatar         string  `json:"-"`
	AvatarThumb    string  `json:"avatar"`
	PlayerID       int64   `json:"player_id"`
	ClanTag        string  `json:"clan_tag"`
	Access         int64   `json:"-"`
	AccessName     string  `json:"type"`
}

type ProfileUpdate struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	Birthday  string `json:"birthday"`
	Country   string `json:"country"`
	Language  string `json:"language"`
	Address   string `json:"address"`
	Signature string `json:"signature"`
}

//parent profile
type PlayerNotification struct {
	Thread           string `json:"thread"`
	EventType        string `json:"event_type"`
	NotificationType string `json:"notification_type"`
}

type PlayerList struct {
	Cursor  string     `json:"c"`
	Players []*Profile `json:"players"`
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

func List(c appengine.Context, pkeyStr, rangeStr, cursor string) (*PlayerList, error) {
	playerKey, err := datastore.DecodeKey(pkeyStr)
	attackRange, err := strconv.ParseBool(rangeStr)
	if err != nil {
		return nil, err
	}
	profiles := make([]*Profile, 0, LIMIT)
	q := datastore.NewQuery("Player").Project("Nick", "BandwidthUsage", "Status",
		"Avatar", "PlayerID", "ClanTag", "Access").Order("-BandwidthUsage").Limit(LIMIT)
	if attackRange {
		player := new(Player)
		if err := datastore.Get(c, playerKey, player); err != nil {
			return nil, err
		}
		rangeLo, rangeHi := RangeForPlayer(player)
		q = q.Filter("BandwidthUsage >", rangeLo).
			Filter("BandwidthUsage <", rangeHi)
	}
	if len(cursor) > 0 {
		cur, err := datastore.DecodeCursor(cursor)
		if err != nil {
			return nil, err
		}
		q = q.Start(cur)
	}
	t := q.Run(c)
	for {
		var profile Profile
		_, err := t.Next(&profile)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, &profile)
	}
	newCur, err := t.Cursor()
	if err != nil {
		return nil, err
	}
	list := &PlayerList{
		Cursor:  newCur.String(),
		Players: profiles,
	}
	c.Debugf("players list : %+v \n", list)
	return list, nil
}

func UpdateProfile(c appengine.Context, update ProfileUpdate) error {
	playerKey, err := datastore.DecodeKey(update.Key)
	if err != nil {
		return err
	}
	player := new(Player)
	if err := datastore.Get(c, playerKey, player); err != nil {
		return err
	}
	if len(update.Birthday) > 0 {
		bd, err := time.Parse(TIMELAYOUT, update.Birthday)
		if err != nil {
			return err
		}
		player.Birthday = bd
	}
	player.Name = update.Name
	player.Country = update.Country
	player.Language = update.Language
	player.Address = update.Address
	player.Signature = update.Signature
	if _, err := datastore.Put(c, playerKey, player); err != nil {
		return err
	}
	return nil
}

func UpdateNotifications(c appengine.Context, pkey string, notifications []PlayerNotification) error {
	return nil
}

// TODO check last update of image to avoid too much changing and uploading
func UpdateAvatar(c appengine.Context, playerStr string, img *blobstore.BlobInfo) error {
	playerKey, err := datastore.DecodeKey(playerStr)
	if err != nil {
		return err
	}
	player := new(Player)
	if err := datastore.Get(c, playerKey, player); err != nil {
		return err
	}
	if len(player.Avatar) > 0 {
		if err := image.DeleteServingURL(c, player.AvatarKey); err != nil {
			return err
		}
	}
	imgURL, err := image.ServingURL(c, img.BlobKey, nil)
	if err != nil {
		return err
	}
	player.AvatarKey = img.BlobKey
	player.Avatar = imgURL.String()
	if _, err := datastore.Put(c, playerKey, player); err != nil {
		return err
	}
	return nil
}
