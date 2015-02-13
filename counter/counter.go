package counter

import (
	"appengine"
	"appengine/datastore"
	"appengine/memcache"
	"fmt"
	"math/rand"
	"strconv"
)

type CounterShardConfig struct {
	NumShards int
}

type CounterShard struct {
	Name  string
	Count int
}

//TODO memcache does not want to play after being emptied???????? (problems with sequence after server reboot)
func IncrementAndCount(c appengine.Context, name string) (int64, error) {
	cnt, err := GetCount(c, name)
	if err != nil {
		return 0, err
	}
	if err := Increment(c, name); err != nil {
		return 0, err
	}
	cnt++
	return int64(cnt), nil
}

func GetCount(c appengine.Context, name string) (int, error) {
	var total int = 0
	item, err := memcache.Get(c, name)
	if err != nil && err != memcache.ErrCacheMiss {
		return 0, err
	}
	if err == nil {
		tot, err := strconv.Atoi(string(item.Value))
		if err != nil {
			return 0, err
		}
		total = tot
	} else {
		q := datastore.NewQuery("CounterShard").Filter("Name = ", name)
		for t := q.Run(c); ; {
			var cs CounterShard
			_, derr := t.Next(&cs)
			if derr == datastore.Done {
				break
			}
			if derr != nil {
				return 0, derr
			}
			total += cs.Count
		}
		item = &memcache.Item{
			Key:   name,
			Value: []byte(strconv.Itoa(total)),
		}
		if err := memcache.Set(c, item); err != nil {
			return 0, err
		}

	}
	return total, nil
}

func Increment(c appengine.Context, name string) error {
	cfg := &CounterShardConfig{3}
	key := datastore.NewKey(c, "CounterShardConfig", name, 0, nil)
	if err := datastore.Get(c, key, cfg); err != nil {
		if err == datastore.ErrNoSuchEntity {
			if _, err := datastore.Put(c, key, cfg); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	done := make(chan int)
	go func() {
		if _, err := memcache.Increment(c, name, 1, 0); err != nil {
			c.Errorf("error incrementing memcache counter")
		}
		done <- 0
	}()
	err := datastore.RunInTransaction(c, func(c appengine.Context) error {
		index := rand.Intn(cfg.NumShards - 1)
		shardname := fmt.Sprintf("%s%d", name, index)
		shardkey := datastore.NewKey(c, "CounterShard", shardname, 0, nil)
		shard := &CounterShard{Name: name}
		if err := datastore.Get(c, shardkey, shard); err != nil {
			if err != datastore.ErrNoSuchEntity {
				return err
			}
		}
		shard.Count = shard.Count + 1
		if _, err := datastore.Put(c, shardkey, shard); err != nil {
			return err
		}
		return nil
	}, nil)
	<-done
	if err != nil {
		return err
	}
	return nil
}
