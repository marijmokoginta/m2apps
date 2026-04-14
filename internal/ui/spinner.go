package ui

import (
	"fmt"
	"time"
)

type Spinner struct {
	stop chan bool
}

func NewSpinner() *Spinner {
	return &Spinner{stop: make(chan bool)}
}

func (s *Spinner) Start(message string) {
	go func() {
		chars := []string{"|", "/", "-", "\\"}
		i := 0

		for {
			select {
			case <-s.stop:
				return
			default:
				fmt.Printf("\r%s %s", chars[i], message)
				time.Sleep(100 * time.Millisecond)
				i = (i + 1) % len(chars)
			}
		}
	}()
}

func (s *Spinner) Stop(message string) {
	s.stop <- true
	fmt.Printf("\r%s\n", message)
}
