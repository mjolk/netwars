package event

import (
	"appengine"
	"appengine/datastore"
	"appengine/delay"
	"appengine/taskqueue"
	"bytes"
	"encoding/gob"
	"fmt"
	"html/template"
	"mj0lk.be/netwars/cache"
	"mj0lk.be/netwars/counter"
	"mj0lk.be/netwars/guid"
	"time"
)

const (
	EMAILNOTIF       = "Email"
	PUSHNOTIF        = "Push"
	IN         int64 = 0
	OUT        int64 = 1
	JSON             = "JSON"
	HTML             = "HTML"
	LOCAL            = "Player"
	GLOBAL           = "Clan"
)

var (
	DirectionName = map[int64]string{
		0: "IN",
		1: "OUT",
	}

	Direction = map[string]int64{
		"IN":  0,
		"OUT": 1,
	}

	invite_tmpl = template.Must(template.ParseFiles("email_templates/Invite_email.tmpl"))
	//invite_tmpl = template.Must(template.ParseFiles("../event/Invite_email.tmpl")) //testing
)

//CLAN parent: clan  key: playerkey
//PLAYER same keyname
type Tracker struct {
	EventCount   int64 `json:"event_count"`
	MessageCount int64 `json:"message_count"`
}

type EventProgram struct {
	Name             string         `json:"name"`
	Source           string         `json:"source"`
	Amount           int64          `json:"amount"`
	Owned            bool           `json:"owned"`
	TypeName         string         `json:"type_name"`
	AmountUsed       int64          `json:"amount_used" datastore:",noindex`
	AmountBefore     int64          `json:"amount_before" datastore:",noindex`
	AmountLost       []int64        `json:"amount_after" datastore:",noindex`
	Lost             int64          `json:"amount_lost" datastore:",noindex"`
	Program          *datastore.Key `json:"program" datastore:",noindex`
	ProgramActive    bool           `json:"program_active" datastore:",noindex`
	BwLost           float64        `json:"bw_lost" datastore:",noindex`
	ActiveDefender   bool           `json:"-" datastore:",noindex`
	AttackEfficiency float64        `json:"-" datastore:"-" datastore:",noindex`
	YieldLost        int64          `json:"yield_lost" datastore:",noindex`
	Power            bool           `datastore:",noindex" json:"power"`
	VDamageReceived  int64          `datastore:",noindex" json:"-"`
}

// convention:
// event.Player is the one who owns the event
// need different events because of no OR queries.(need multiple queries to get all events for one player when using single event entities for both parties)
// direction IN indicates the initiator of the event is TargetName, TargetID
// direction OUT indicates the initiator of the event is Player, PlayerName, PlayerID
type Event struct {
	Target            *datastore.Key `json:"-" datastore:",noindex` //can be clan or player
	TargetName        string         `json:"target_name"`
	TargetID          int64          `json:"target_id"`
	ID                int64          `json:"event_id"` //sequence simultaniously event counter, 1 event has 2 entities (owners) but one id (count)
	Created           time.Time      `json:"created"`
	Player            *datastore.Key `json:"-"`           //OWNER
	PlayerName        string         `json:"player_name"` //owner nick
	PlayerID          int64          `json:"player_id"`   // owner id
	EventType         string         `json:"event_type"`
	Result            bool           `json:"result"`    //result of the event vis a vis owner (success of failure either defending or attacking)
	Direction         int64          `json:"direction"` //incoming or outgoing event
	Clan              *datastore.Key `json:"clan_key"`  //owning clan
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
	VDamageReceived   int64          `datastore:",noindex" json:"-"`
	GUID              string         `datastore:"-" json:"-"`
}

type PlayerNotification struct {
	Player           *datastore.Key
	Created          time.Time
	Thread           string `json:"thread"`
	EventType        string `json:"event_type"`
	NotificationType string `json:"notification_type"`
	DeviceToken      string
	Email            string
}

type Email struct {
	Email   string
	Subject string
	Content string
}

type Notification struct {
	PlayerKey  *datastore.Key
	ClanNotify bool
}

func NewPullTask(notif interface{}, id, path string) (*taskqueue.Task, error) {
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(notif); err != nil {
		return nil, err
	}

	return &taskqueue.Task{
		Path:    path,
		Payload: buf.Bytes(),
		Tag:     id,
	}, nil

}

func (e Event) CreateEmail(email string) (*taskqueue.Task, error) {
	notif := Email{}
	buf := new(bytes.Buffer)
	err := invite_tmpl.ExecuteTemplate(buf, e.Action+"_email.tmpl", e)
	if err != nil {
		return nil, err
	}
	notif.Content = buf.String()
	notif.Subject = "Netwars :" + e.PlayerName
	notif.Email = email
	task, err := NewPullTask(notif, e.PlayerName, "/notif")
	if err != nil {
		return nil, err
	}
	return task, nil

}

func (e Event) CreatePush() {

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

type EventFunc func(c appengine.Context, events []*Event) error

func (event *Event) Email() {

	//load template for eventtype
	/*	msg := &mail.Message{
					Sender:  "Example.com Support <n3twars@jainware.be>",
					To:      []string{email},
					Subject: "Confirm your registration",
					Body:    fmt.Sprintf(confirmMessage, url),
		url		}
				if err := mail.Send(c, msg); err != nil {
					c.Errorf("Couldn't send email: %v", err)
				}*/
	//	fmt.Printf("sending email to %s \n", event.Email)
}

func (event *Event) Push() {
	//	fmt.Printf("Pushing message to %s \n", event.Email)
}

func Func(c appengine.Context, events []*Event) error {
	evCnt := len(events)
	if evCnt > 0 {
		cntCh := make(chan int64, 1)
		NewEventID(c, cntCh)
		keys := make([]*datastore.Key, evCnt)
		models := make([]interface{}, evCnt)
		id := <-cntCh
		for i := range events {
			events[i].ID = id
			keys[i] = datastore.NewKey(c, "Event", events[i].GUID, 0, nil)
			models[i] = events[i]
		}
		if len(keys) == 1 {
			if _, err := datastore.Put(c, keys[0], models[0]); err != nil {
				return err
			}
		} else {
			if _, err := datastore.PutMulti(c, keys, models); err != nil {
				return err
			}
		}
		notifyCh := make(chan int, evCnt)
		for _, ev := range events {
			go ev.Notify(c, notifyCh)
		}
		for n := 0; n < evCnt; n++ {
			<-notifyCh
		}
	}
	return nil

}

func Send(c appengine.Context, em []*Event, e EventFunc) error {
	if len(em) > 0 {
		for i := range em {
			gid, err := guid.GenUUID()
			if err != nil {
				return err
			}
			em[i].GUID = gid
		}
	}
	laterFunc := delay.Func("event", e)
	laterFunc.Call(c, em)
	//t, err := laterFunc.Task(em)
	//if err != nil {
	//	c.Errorf("Failed to create task: %s", err)
	//}
	//hostName, _ := appengine.ModuleHostname(context, "[event]", "", "")
	//t.Header = make(map[string][]string)
	//t.Header.Set("Host", "localhost:8081")
	//if _, err := taskqueue.Add(c, t, ""); err != nil {
	//	return err
	//}
	return nil
}

type EventList struct {
	Events []Event `json:"events"`
	Cursor string  `json:"cursor"`
}

func NewEventID(c appengine.Context, cntCh chan<- int64) {
	cnt, err := counter.IncrementAndCount(c, "Event")
	if err != nil {
		c.Errorf("error event counter %s \n", err)
		cntCh <- 0
	}
	cntCh <- cnt
}

func NotificationsForType(c appengine.Context, playerKey *datastore.Key, eventType string) ([]*PlayerNotification, error) {
	k := fmt.Sprintf("%s_Notifications", playerKey.StringID())
	notif := make([]*PlayerNotification, 10)
	cnt := 0
	if !cache.Get(c, k, notif) {
		q := datastore.NewQuery("PlayerNotification").Filter("Player =", playerKey)
		for t := q.Run(c); ; {
			var pn PlayerNotification
			_, err := t.Next(&pn)
			if err == datastore.Done {
				break
			}
			if err != nil {
				return nil, err
			}
			notif[cnt] = &pn
			cnt++
		}
		if cnt > 0 {
			cache.Add(c, k, notif)
		}
		notif = notif[:cnt]
	}
	notifsForType := make([]*PlayerNotification, 0)
	if cnt > 0 {
		for _, notification := range notif {
			if notification.EventType == eventType {
				notifsForType = append(notifsForType, notification)
			}
		}
	}
	return notifsForType, nil
}

func incrementTracker(c appengine.Context, pl string) error {
	//trName := pl + "local"
	//	done := make(chan int)
	//	go func() {
	//		if _, err := memcache.Increment(c, trName, 1, 0); err != nil {
	//			c.Errorf("error incrementing memcache counter")
	//		}
	//		done <- 0
	//	}()
	err := datastore.RunInTransaction(c, func(c appengine.Context) error {
		trackerKey := datastore.NewKey(c, "Tracker", pl, 0, nil)
		tracker := new(Tracker)
		if err := datastore.Get(c, trackerKey, tracker); err != nil {
			return err
		}
		tracker.EventCount++
		if _, err := datastore.Put(c, trackerKey, tracker); err != nil {
			return err
		}
		return nil
	}, nil)
	//	<-done
	if err != nil {
		return err
	}
	return nil
}

func (e Event) NotifyPlayer(c appengine.Context, notifCh chan<- *taskqueue.Task, key *datastore.Key) {
	c.Debugf("notify payer --------\n")
	localTrackerCh := make(chan int)
	clanNotify := false
	if key != nil {
		clanNotify = true
	}
	playerKey := e.Player
	if clanNotify {
		playerKey = key
	} else {
		go func() {
			if err := incrementTracker(c, playerKey.StringID()); err != nil {
				c.Errorf("error increment local tracker %s", err)
			}
			localTrackerCh <- 1
		}()
	}
	notifications, err := NotificationsForType(c, playerKey, e.Action)
	if err != nil {
		c.Errorf("error getting notifications : %s", err)
	}
	for _, notif := range notifications {

		switch notif.NotificationType {
		case EMAILNOTIF:
			task, err := e.CreateEmail(notif.Email)
			if err != nil {
				c.Errorf("error creating task: %s", err)
			}
			notifCh <- task
		case PUSHNOTIF:
			//	Push()
		}
	}
	if !clanNotify {
		<-localTrackerCh
	}
	notifCh <- &taskqueue.Task{}
}

func (e Event) Notify(c appengine.Context, readyCh chan<- int) {
	notifs := make([]Notification, 0, 20)
	playerNotifyCh := make(chan *taskqueue.Task)
	var chCnt int64
	if e.Clan != nil {
		err := datastore.RunInTransaction(c, func(c appengine.Context) error {
			trackers := make([]interface{}, 20)
			trackerKeys := make([]*datastore.Key, 20)
			var cnt int64
			var pcnt int64
			q := datastore.NewQuery("Tracker").Ancestor(e.Clan)
			for t := q.Run(c); ; {
				var ct Tracker
				tkey, err := t.Next(&ct)
				if err == datastore.Done {
					break
				}
				if err != nil {
					c.Errorf("error notifying : %s", err)
				}
				send := false
				if e.Player == nil {
					send = true
				} else if tkey.StringID() != e.Player.StringID() {
					send = true
				} else {
					pcnt++
					notifs = append(notifs, Notification{nil, false})
				}
				if send {
					ct.EventCount++
					trackers[cnt] = &ct
					trackerKeys[cnt] = tkey
					playerKey := datastore.NewKey(c, "Player", tkey.StringID(), 0, nil)
					cnt++
					notifs = append(notifs, Notification{playerKey, true})
				}

			}
			chCnt = cnt + pcnt
			if cnt > 0 {
				trackers := trackers[:cnt]
				trackerKeys := trackerKeys[:cnt]
				if _, err := datastore.PutMulti(c, trackerKeys, trackers); err != nil {
					return err
				}
			}
			return nil
		}, nil)
		if err != nil {
			c.Errorf("error saving global trackers %s", err)
		}
	} else {
		notifs = append(notifs, Notification{nil, false})
		chCnt = 1
	}
	for _, notif := range notifs {
		if notif.ClanNotify {
			go e.NotifyPlayer(c, playerNotifyCh, notif.PlayerKey)
		} else {
			go e.NotifyPlayer(c, playerNotifyCh, nil)
		}
	}
	var ni int64
	tasks := make([]*taskqueue.Task, chCnt, chCnt)
	for ni = 0; ni < chCnt; {
		task := <-playerNotifyCh
		if task.Path == "" {
			ni++
		} else {
			tasks[ni] = task
			c.Debugf("task for notification: %+v \n", tasks[ni])
		}

	}
	/*	if _, err := taskqueue.AddMulti(c, tasks, ""); err != nil {
		c.Errorf("\n errors adding tasks : %s", err)
	}*/
	readyCh <- 0
}
