package program

import (
	"appengine"
	"appengine/datastore"
	"encoding/json"
	"errors"
	"io/ioutil"
	"mj0lk.be/netwars/cache"
	"strings"
	"time"
)

const (
	SW   int64 = 1 << iota
	MUT  int64 = 1 << iota
	HUK  int64 = 1 << iota
	D0S  int64 = 1 << iota
	FW   int64 = 1 << iota
	CONN int64 = 1 << iota
	INT  int64 = 1 << iota
	ICE  int64 = 1 << iota
	INF  int64 = 1 << iota
)

var ProgramName = map[int64]string{
	1:   "Swarm",
	2:   "Mutator",
	4:   "Hunter/Killer",
	8:   "d0s",
	16:  "Firewall",
	32:  "Connection",
	64:  "Intelligence",
	128: "Ice",
	256: "Infect",
}

var ProgramType = map[string]int64{
	"Swarm":         1,
	"Mutator":       2,
	"Hunter/Killer": 4,
	"d0s":           8,
	"Firewall":      16,
	"Connection":    32,
	"Intelligence":  64,
	"Ice":           128,
	"Infect":        256,
}

type Program struct {
	EncodedKey     string         `datastore:"-" json:"program_key"`
	Key            *datastore.Key `datastore:"-" json:"-"`
	Name           string         `datastore:"-" json:"name"`
	Attack         int64          `json:"attack"`
	Life           int64          `json:"life"`
	Bandwidth      int64          `json:"bandwidth"`
	BandwidthUsage float64        `json:"bandwidth_usage" datastore:"-"`
	Type           int64          `json:"-"`
	TypeName       string         `json:"type" datastore:"-"`
	Cycles         int64          `json:"cycle_cost"`
	Memory         float64        `json:"mem_cost"`
	Description    string         `json:"description"`
	Created        time.Time      `json:"-"`
	Updated        time.Time      `json:"-"`
	EffectorTypes  int64          `datastore:",noindex" json:"-"`
	Effectors      []string       `datastore:"-" json:"effector"`
	Ettl           int64          `json:"ettl"`
	Infect         *datastore.Key `json:"infect"`
	InfectName     string         `json:"infect_name" datastore:"-"`
}

func (p *Program) Load(c <-chan datastore.Property) error {
	if err := datastore.LoadStruct(p, c); err != nil {
		return err
	}
	//load stuff
	for tpeKey, tpe := range ProgramName {
		if tpeKey == tpeKey&p.EffectorTypes {
			p.Effectors = append(p.Effectors, tpe)
		}
	}
	if p.Infect != nil {
		p.InfectName = p.Infect.StringID()
	}
	p.TypeName = ProgramName[p.Type]
	p.BandwidthUsage = (1 / p.Memory) * float64(p.Cycles)
	return nil
}

func (p *Program) Save(c chan<- datastore.Property) error {
	if p.Created.IsZero() {
		p.Created = time.Now()
	}
	p.Type = ProgramType[p.TypeName]
	if len(p.Effectors) > 0 {
		p.EffectorTypes = 0
		for _, tpe := range p.Effectors {
			p.EffectorTypes |= ProgramType[strings.TrimSpace(tpe)]

		}
	}
	p.Updated = time.Now()
	return datastore.SaveStruct(p, c)
}

func KeyGet(c appengine.Context, pKey *datastore.Key) (*Program, error) {
	stringId := pKey.StringID()
	program := new(Program)
	if !cache.Get(c, stringId, program) {
		if err := datastore.Get(c, pKey, program); err != nil {
			c.Debugf("program from store -- %s\n", err)
			return nil, err
		}
		program.Name = stringId
		program.Key = pKey
		program.EncodedKey = pKey.Encode()
		cache.Set(c, stringId, program)
	}
	return program, nil
}

func Get(c appengine.Context, pKeyStr string) (*Program, error) {
	programKey, err := datastore.DecodeKey(pKeyStr)
	if err != nil {
		return nil, err
	}
	program, err := KeyGet(c, programKey)
	if err != nil {
		return nil, err
	}
	return program, nil
}

func GetAll(c appengine.Context, programs map[string][]*Program) error {
	qp := datastore.NewQuery("Program")
	for t := qp.Run(c); ; {
		var p Program
		key, err := t.Next(&p)
		if err == datastore.Done {
			break
		} else if err != nil {
			return err
		}
		p.Key = key
		p.EncodedKey = key.Encode()
		p.Name = key.StringID()
		programs[p.TypeName] = append(programs[p.TypeName], &p)

	}
	return nil
}

func LoadFromFile(c appengine.Context) error {
	file, err := ioutil.ReadFile("programs.json")
	if err != nil {
		return err

	}
	var jsontype []Program
	json.Unmarshal(file, &jsontype)
	for _, program := range jsontype {
		if err := CreateOrUpdate(c, &program); err != nil {
			c.Debugf("error %s", err)
			return err
		}
	}
	return nil
}

func CreateOrUpdate(c appengine.Context, program *Program) error {
	var pkey *datastore.Key
	err := datastore.RunInTransaction(c, func(c appengine.Context) error {
		var err error
		if len(program.EncodedKey) > 0 {
			pkey, err = datastore.DecodeKey(program.EncodedKey)
			if err != nil {
				return err
			}
		} else {
			if len(program.Name) > 0 {
				pkey = datastore.NewKey(c, "Program", program.Name, 0, nil)
			} else {
				return errors.New("Name program required")
			}
		}
		if len(program.InfectName) > 0 {
			program.Infect = datastore.NewKey(c, "Program", program.InfectName, 0, nil)
		}
		if _, err := datastore.Put(c, pkey, program); err != nil {
			c.Debugf("datastore error: %s", err)
			return err
		}

		return nil
	}, nil)
	if err != nil {
		return err
	}
	//get for development
	if _, err := KeyGet(c, pkey); err != nil {
		return err
	}
	return nil
}
