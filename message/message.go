package message

import (
	"appengine"
	"appengine/datastore"
	"errors"
	"mj0lk.be/netwars/counter"
	//"mj0lk.be/netwars/event"
	"mj0lk.be/netwars/guid"
	"mj0lk.be/netwars/player"
	"time"
)

//TODO add delete message/board/thread
const (
	PUBLIC  int64 = 1 << iota
	CLAN    int64 = 1 << iota
	PRIVATE int64 = 1 << iota
)

var AccessName = map[int64]string{
	PUBLIC:                            "Public",
	player.ADMIN:                      "Mod",
	player.LEADER | player.LIEUTENANT: "Clan",
}

var AccessType = map[string]int64{
	"Clan":      player.LEADER | player.LIEUTENANT,
	"Public":    PUBLIC,
	"Moderator": player.ADMIN,
}

type MessageList struct {
	Cursor    string    `json:"cursor"`
	Messages  []Message `json:"messages"`
	BoardKey  string    `json:"board_key"`
	ThreadKey string    `json:"thread_key"`
}

type Message struct {
	DbKey       *datastore.Key `datastore:"-" json:"-"`
	EncodedKey  string         `datastore:"-" json:"message_key"`
	PID         int64          `json:"pid" datastore:"-"`  //personal message recipient
	Bkey        string         `datastore:"-" json:"bkey"` // board id
	Tkey        string         `datastore:"-" json:"tkey"` // thread id
	Skey        string         `datastore:"-" json:"skey"`
	Creator     *datastore.Key `json:"-"`
	Clan        *datastore.Key `json:"-"`
	Scope       int64          `json:"scope"` // message container
	Created     time.Time      `json:"created"`
	MessageID   int64          `datastore:",noindex" json:"message_id"`
	Content     string         `datastore:",noindex" json:"content"`
	Signature   string         `datastore:",noindex" json:"signature"`
	AvatarThumb string         `datastore:",noindex" json:"avatar_thumb"`
	PlayerName  string         `datastore:",noindex" json:"player_name"`
	PlayerID    int64          `json:"player_id"`
	Subject     string         `datastore:",noindex" json:"subject"`
	IsThread    bool           `json:"-"`
	IsBoard     bool           `json:"-"`
	IsDeleted   bool           `json:"-"`
	Access      int64          `json:"-"`
	AccessName  string         `datastore:"-" json:"access"`
	Recipient   *datastore.Key `json:"-"`
	Board       *datastore.Key `json:"-"`
}

func newMessageID(c appengine.Context, cntCh chan<- int64) {
	cnt, err := counter.IncrementAndCount(c, "Message")
	if err != nil {
		c.Errorf("error message counter %s \n", err)
	}
	cntCh <- cnt
}

func CreateOrUpdate(c appengine.Context, playerKeyStr string, message Message) error {
	iplayer := new(player.Player)
	playerKey, err := player.Get(c, playerKeyStr, iplayer)
	if err != nil {
		return err
	}
	nmssg := new(Message)
	var threadKey *datastore.Key
	if len(message.EncodedKey) > 0 {
		//update existing message
		mssgKey, err := datastore.DecodeKey(message.EncodedKey)
		if err != nil {
			return err
		}
		if err := datastore.Get(c, mssgKey, nmssg); err != nil {
			return err
		}
		if !nmssg.Creator.Equal(playerKey) && iplayer.Access&player.MOD == 0 {
			return errors.New("Can't update message")
		}
		nmssg.Content = message.Content
		nmssg.Subject = message.Subject
	} else {
		//new message
		idCnt := make(chan int64, 1)
		go newMessageID(c, idCnt)
		newmssgGuid, err := guid.GenUUID()
		if err != nil {
			return err
		}
		if len(message.Bkey) == 0 && len(message.Tkey) == 0 {
			//creating new board
			if message.Scope == 0 {
				return errors.New("No scope provided")
			}
			nmssg.IsBoard = true
			switch message.Scope {
			case CLAN:
				if iplayer.Access&player.MOD == 0 {
					if iplayer.ClanKey == nil {
						return errors.New("cannot create clan board")
					}
					if iplayer.MemberType < player.LIEUTENANT {
						return errors.New("cannot create clan board")
					}
				}

				nmssg.Scope = CLAN
				nmssg.Clan = iplayer.ClanKey
			case PUBLIC:
				access := player.MOD | player.ADMIN
				if iplayer.Access&access == 0 {
					return errors.New("cannot create board")
				}
				nmssg.Scope = PUBLIC
			}
			nmssg.DbKey = datastore.NewKey(c, "Message", newmssgGuid, 0, nil)
		}
		//new thread
		if len(message.Bkey) > 0 {
			// new thread in board
			boardKey, err := datastore.DecodeKey(message.Bkey)
			if err != nil {
				return err
			}
			nmssg.Board = boardKey
			nmssg.IsThread = true
			nmssg.DbKey = datastore.NewKey(c, "Message", newmssgGuid, 0, nil)
		} else if len(message.Tkey) > 0 { // new message in thread
			// new message in thread
			var tErr error
			threadKey, tErr = datastore.DecodeKey(message.Tkey)
			if tErr != nil {
				return tErr
			}
			nmssg.DbKey = datastore.NewKey(c, "Message", newmssgGuid, 0, threadKey)
		} else if len(message.Skey) > 0 {
			subscriberKey, err := datastore.DecodeKey(message.Skey)
			if err != nil {
				return err
			}
			nmssg.DbKey = datastore.NewKey(c, "Message", newmssgGuid, 0, subscriberKey)
		} else if message.PID > 0 {
			playerKey, err := player.KeyByID(c, message.PID)
			if err != nil {
				return err
			}
			nmssg.Recipient = playerKey
			nmssg.Scope = PRIVATE
			nmssg.DbKey = datastore.NewKey(c, "Message", newmssgGuid, 0, nil)
		}
		nmssg.Creator = playerKey
		nmssg.Signature = iplayer.Signature
		nmssg.AvatarThumb = iplayer.AvatarThumb
		nmssg.Created = time.Now()
		nmssg.Content = message.Content
		nmssg.Subject = message.Subject
		nmssg.PlayerName = iplayer.Nick
		nmssg.PlayerID = iplayer.ID
		nmssg.Access = AccessType[message.AccessName]
		nmssg.MessageID = <-idCnt
	}
	if _, err := datastore.Put(c, nmssg.DbKey, nmssg); err != nil {
		return err
	}
	//TODO message events
	//only send event on regular message
	//enable users to register on messages posted on certain threads.
	/*e := &event.Event{
		Player: playerKey,
	}
	if threadKey != nil {
		if err := event.Send(c, nmssg, MESSAGE_EVENT_PATH); err != nil {
			return err
		}
	}*/
	return nil
}

func addCursor(cursorStr string, q *datastore.Query) error {
	if len(cursorStr) > 0 {
		cursor, err := datastore.DecodeCursor(cursorStr)
		if err != nil {
			return err
		}
		q = q.Start(cursor)
	}
	return nil
}

func newMessageList() MessageList {
	return MessageList{
		"",
		make([]Message, 20, 20),
		"",
		"",
	}
}

//TODO check player key
func PublicBoards(c appengine.Context, playerStr, cursorStr string) (MessageList, error) {
	/*playerKey, err := datastore.DecodeKey(playerStr)
	if err != nil {
		return nil, err
	}*/
	list := newMessageList()
	var cnt int
	q := datastore.NewQuery("Message").Filter("Access =", 1).Filter("IsBoard =", true).Filter("Scope =", PUBLIC).
		Filter("IsDeleted =", false).Limit(20)
	if err := addCursor(cursorStr, q); err != nil {
		return MessageList{}, err
	}
	it := q.Run(c)
	for {
		var msg Message
		key, err := it.Next(&msg)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return MessageList{}, err
		}
		msg.EncodedKey = key.Encode()
		list.Messages[cnt] = msg
		cnt++
	}
	newCursor, err := it.Cursor()
	if err != nil {
		return MessageList{}, err
	}
	list.Messages = list.Messages[:cnt]
	list.Cursor = newCursor.String()
	return list, nil
}

func ClanBoards(c appengine.Context, playerStr, cursorStr string) (MessageList, error) {
	iplayer := new(player.Player)
	_, err := player.Get(c, playerStr, iplayer)
	if err != nil {
		return MessageList{}, err
	}
	if iplayer.ClanKey == nil {
		return MessageList{}, errors.New("Cannot load clan boards")
	}
	list := newMessageList()
	var cnt int
	q := datastore.NewQuery("Message").Filter("Clan =", iplayer.ClanKey).Filter("IsBoard =", true).Filter("Scope =", CLAN).
		Filter("IsDeleted =", false).Limit(20)
	if err := addCursor(cursorStr, q); err != nil {
		return MessageList{}, err
	}
	it := q.Run(c)
	for {
		var msg Message
		key, err := it.Next(&msg)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return MessageList{}, err
		}
		msg.EncodedKey = key.Encode()
		list.Messages[cnt] = msg
		cnt++
	}
	newCursor, err := it.Cursor()
	if err != nil {
		return MessageList{}, err
	}
	list.Messages = list.Messages[:cnt]
	list.Cursor = newCursor.String()
	return list, nil
}

func threadQuery(boardKey *datastore.Key) *datastore.Query {
	return datastore.NewQuery("Message").Filter("Board =", boardKey).Filter("IsDeleted =", false).
		Filter("IsThread =", true).Order("-Created").Limit(40)
}

func messageQuery(threadKey *datastore.Key) *datastore.Query {
	return datastore.NewQuery("Message").Ancestor(threadKey).Filter("IsDeleted =", false).
		Filter("IsThread =", false).Order("-Created").Limit(40)
}

func Messages(c appengine.Context, tpe, playerStr, keyStr, cursorStr string) (MessageList, error) {
	//playerKey, err := datastore.DecodeKey(playerStr)
	var cnt int
	tpeKey, err := datastore.DecodeKey(keyStr)
	if err != nil {
		return MessageList{}, err
	}
	list := newMessageList()
	var q *datastore.Query
	switch tpe {
	case "threads":
		q = threadQuery(tpeKey)
		list.BoardKey = tpeKey.Encode()
	case "messages":
		q = messageQuery(tpeKey)
		list.ThreadKey = tpeKey.Encode()
	}
	if err := addCursor(cursorStr, q); err != nil {
		return MessageList{}, err
	}
	it := q.Run(c)
	for {
		var msg Message
		key, err := it.Next(&msg)
		if err == datastore.Done {
			break
		}
		if err != nil {
			return MessageList{}, err
		}
		msg.EncodedKey = key.Encode()
		list.Messages[cnt] = msg
		cnt++
	}
	newCursor, err := it.Cursor()
	if err != nil {
		return MessageList{}, err
	}
	list.Messages = list.Messages[:cnt]
	list.Cursor = newCursor.String()
	return list, nil
}
