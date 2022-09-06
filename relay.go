package relay

import (
	"machine"
	"strings"
	"time"

	"github.com/eyelight/trigger"
)

type relay struct {
	name       string
	pin        machine.Pin
	since      time.Time
	duration   time.Duration
	durationCh chan time.Duration
}

type Relay interface {
	Configure()
	Get() bool
	Set(bool) bool
	On() bool
	Off() bool
	Name() string
	Execute(t trigger.Trigger)
	State() (interface{}, time.Time)
	StateString() string
	DurationCh() chan time.Duration
}

// New returns a Relay ready to be configured. The pin you pass here need not be configured.
func New(p machine.Pin, name string) Relay {
	return &relay{
		name:       name,
		pin:        p,
		since:      time.Time{},
		duration:   0 * time.Second,
		durationCh: make(chan time.Duration, 1),
	}
}

// Configure sets up the Relay for use, beginning in the "Off" state
func (r *relay) Configure() {
	r.pin.Configure(machine.PinConfig{Mode: machine.PinOutput})
	r.Off()
	r.since = time.Now()
}

func (r *relay) DurationCh() chan time.Duration {
	return r.durationCh
}

// Execute acts on input from a trigger and along with relay.Name() implements the Triggerable interface
func (r *relay) Execute(t trigger.Trigger) {
	// println("relay.Execute()...")
	if t.Target != r.name {
		t.Error = true
		t.Message = string("error - " + r.name + " received a trigger intended for " + t.Target)
		t.ReportCh <- t
		return
	}
	switch t.Action {
	case "On", "on", "ON":
		t.Error = false
		go func() {
			r.since = time.Now()
			r.pin.High()
			if t.Duration == 0 {
				t.Message = string(r.name + " - On indefinitely at " + time.Now().Local().Format(time.RFC822))
				t.ReportCh <- t
				return
			}
			r.duration = t.Duration
			t.Message = string(r.name + " - On for " + t.Duration.String() + " at " + time.Now().Local().Format(time.RFC822))
			for {
				select {
				case newDuration := <-r.durationCh:
					r.duration = newDuration
					t.Message = string(r.name + " - Changing On duration to " + r.duration.String() + " at " + time.Now().Local().Format(time.RFC822))
					t.ReportCh <- t
				default:
					if time.Since(r.since) > r.duration {
						r.pin.Low()
						t.Message = string(r.name + " - Off at " + time.Now().Local().Format(time.RFC822) + " after " + time.Since(r.since).String())
						t.ReportCh <- t
						r.reset()
						return
					}
				}
			}
		}()
		return
	case "Off", "off", "OFF":
		r.durationCh <- 0                 // an existing "on" goroutine will be canceled by sending a zero duration
		time.Sleep(10 * time.Millisecond) // allow that some time to take effect so the "on" goroutine will exit & send status
		if r.pin.Get() {                  // if the "on" routine hasn't done so, force it off
			r.pin.Low()
			t.Error = false
			t.Message = string(r.name + " - Off at " + time.Now().Local().Format(time.RFC822) + " after " + time.Since(r.since).String())
			t.ReportCh <- t
			r.reset()
			return
		}
		return
	default:
		t.Error = true
		t.Message = string("error - " + r.name + " does not understand Action: '" + t.Action + "' (On, Off)")
		t.ReportCh <- t
		return
	}
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

// Name returns the relay's name and along with relay.Execute() implements the Triggerable interface
func (r *relay) Name() string {
	return r.name
}

// reset zeroes the timing fields of a relay struct
func (r *relay) reset() {
	r.duration = 0
	r.since = time.Time{}
}
