package program

import (
	"appengine/aetest"
	"appengine/datastore"
	"testing"
	"time"
)

func TestCreateOrUpdate(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	//create
	jprogram := &Program{
		Name:        "Swarm mark IV",
		Attack:      5,
		Life:        5,
		TypeName:    "Swarm",
		Cycles:      5,
		Memory:      0.05,
		Description: "Swarm mark IV lethal bandwidth threat",
		Effectors:   []string{"Swarm", "Connection"},
	}
	if err := CreateOrUpdate(c, jprogram); err != nil {
		t.Fatalf("error: %s", err)
	}
	programKey := datastore.NewKey(c, "Program", jprogram.Name, 0, nil)
	program, err := Get(c, programKey.Encode())
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	t.Logf("program persisted %s \n", program.Description)
	//update
	jprogram.EncodedKey = programKey.Encode()
	jprogram.Description = "Updated"
	jprogram.Effectors[0] = "Hunter/Killer"
	if err := CreateOrUpdate(c, jprogram); err != nil {
		t.Fatalf("error: %s", err)
	}
	if err := datastore.Get(c, programKey, program); err != nil {
		t.Fatalf("error: %s", err)
	}
	t.Logf("program updated %s ", program.Description)
}

func TestGetAll(t *testing.T) {
	c, err := aetest.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	jprogram0 := &Program{
		Name:        "Swarm mark IV",
		Attack:      65,
		Life:        70,
		TypeName:    "Swarm",
		Cycles:      70,
		Memory:      0.5,
		Description: "Swarm mark IV lethal bandwidth threat",
		Effectors:   []string{"Swarm", "Connection"},
	}
	jprogram1 := &Program{
		Name:        "Mutator mark IV",
		Attack:      20,
		Life:        25,
		TypeName:    "Mutator",
		Cycles:      30,
		Memory:      0.25,
		Description: "Mutator mark IV lethal memory threat",
		Effectors:   []string{"Swarm", "Mutator"},
	}
	jprogram2 := &Program{
		Name:        "D0S mark IV",
		Attack:      5,
		Life:        5,
		TypeName:    "d0s",
		Cycles:      5,
		Memory:      0.12,
		Description: "d0s mark IV lethal bandwidth threat",
		Effectors:   []string{"Hunter/Killer", "Connection"},
	}
	pkey0 := datastore.NewKey(c, "Program", jprogram0.Name, 0, nil)
	pkey1 := datastore.NewKey(c, "Program", jprogram1.Name, 0, nil)
	pkey2 := datastore.NewKey(c, "Program", jprogram2.Name, 0, nil)
	programs := []interface{}{jprogram0, jprogram1, jprogram2}
	programKeys := []*datastore.Key{pkey0, pkey1, pkey2}
	if _, err := datastore.PutMulti(c, programKeys, programs); err != nil {
		t.Fatalf("error: %s", err)
	}
	time.Sleep(1 * time.Second)
	nprograms := make(map[string][]*Program)
	if err := GetAll(c, nprograms); err != nil {
		t.Fatalf("error: %s", err)
	}
	testprog := new(Program)
	if err := datastore.Get(c, pkey0, testprog); err != nil {
		t.Fatalf("error: %s", err)
	}
	t.Logf("tesprogram %v \n", testprog)
	t.Logf("programs loaded: %d", len(nprograms))
}

//test does not work , does work when running in appengine
/*func TestLoadFile(t *testing.T) {
	c, err := appenginetesting.NewContext(nil)
	if err != nil {
		t.Fatalf("NewContext: %v", err)
	}
	defer c.Close()
	if err := LoadFromFile(c); err != nil {
		t.Fatalf("error: %s", err)
	}
}*/
