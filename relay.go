package relay

import (
	"machine"
	"strings"
	"time"
)

type relay struct {
	name  string
	pin   machine.Pin
	since time.Time
}

type Relay interface {
	Configure()
	Get() bool
	Set(bool) bool
	On() bool
	Off() bool
	Name() string
	State() (interface{}, time.Time)
	StateString() string
}

// New returns a Relay ready to be configured. The pin you pass here need not be configured.
func New(p machine.Pin, name string) Relay {
	return &relay{
		name: name,
		pin:  p,
	}
}

// Configure sets up the Relay for use, beginning in the "Off" state
func (r *relay) Configure() {
	r.pin.Configure(machine.PinConfig{Mode: machine.PinOutput})
	r.Off()
	r.since = time.Now()
}

// Get returns a measured reading of the Relay's pin
func (r *relay) Get() bool {
	return r.pin.Get()
}

// Set brings the Relay's pin to the passed-in value and returns a subsequent, measured confirmation
func (r *relay) Set(s bool) bool {
	r.pin.Set(s)
	r.since = time.Now()
	time.Sleep(5 * time.Millisecond)
	return r.pin.Get()
}

// On brings the Relays's pin high and returns a subsequent, measured confirmation
func (r *relay) On() bool {
	r.pin.High()
	r.since = time.Now()
	time.Sleep(5 * time.Millisecond)
	return r.pin.Get()
}

// Off brings the Relay's pin low and reutrns a subsequent, measured confirmation
func (r *relay) Off() bool {
	r.pin.Low()
	r.since = time.Now()
	time.Sleep(5 * time.Millisecond)
	return r.pin.Get()
}

/*
	Statist interface methods
	State() (interface{}, time.Time)
	SetState(interface{}, time.Time)
	StateString() string
	Name() string
*/

// State returns a Relay's state as a bool and the time since this state has been valid
func (r *relay) State() (interface{}, time.Time) {
	return r.Get(), r.since
}

// StateString returns a Relay's state and the time since this has been valid as a string
func (r *relay) StateString() string {
	s := "ON"
	if !r.Get() {
		s = "OFF"
	}
	ss := strings.Builder{}
	ss.Grow(1024)
	ss.WriteString(time.Now().String())
	ss.WriteString(" -- (Relay) ")
	ss.WriteString(r.name)
	ss.WriteString(" ")
	ss.WriteString(s)
	ss.WriteString(" since ")
	ss.WriteString(r.since.String())
	return ss.String()
}

func (r *relay) Name() string {
	return r.name
}
