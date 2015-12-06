package player

import (
	"appengine"

	"encoding/json"
	"mj0lk.be/netwars/app"
)

type GeneratePlayer struct {
	Creation    Creation
	Allocations []Allocation
}

func Generate(c appengine.Context) error {
	file, err := app.LoadFile("players")
	if err != nil {
		return err
	}
	var players []GeneratePlayer
	json.Unmarshal(file, &players)
	inBuffer := make(chan GeneratePlayer, 5)
	outBuffer := make(chan string)
	errCh := make(chan error)
	amount2Generate := len(players)
	amountGenerated := 0
	var errs []error
	go func(amount int) {
		for {
			select {
			case inBuffer <- players[amount2Generate-amount]:
				amount--
			default:
				if amount <= 0 {
					return
				}
			}
		}
	}(amount2Generate)

	for {
		select {
		case p := <-inBuffer:
			go func() {
				playerKey, err := createPlayer(c, p.Creation.Nick, p.Creation.Email, "")
				if err != nil {
					errCh <- err
					return
				}
				pLen := len(p.Allocations)
				playerStr := playerKey.Encode()
				done := make(chan int)
				for _, alloc := range p.Allocations {
					go func(a Allocation) {
						if err := Allocate(c, playerStr, a); err != nil {
							errCh <- err
						}
						done <- 0
					}(alloc)
				}
				for i := 0; i < pLen; i++ {
					<-done
				}
				outBuffer <- playerKey.StringID()
			}()
		case <-outBuffer:
			amountGenerated++
		default:
			if amountGenerated == amount2Generate {
				return nil
			}
		}
	}
	if len(errs) != 0 {
		return errs[0]
	}
	return nil
}
