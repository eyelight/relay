package relay

import (
	"machine"
	"strconv"
	"strings"
	"time"

	"github.com/eyelight/trigger"
)

type relay struct {
	name       string
	pin        machine.Pin
	onTime     time.Time
	duration   time.Duration
	durationCh chan time.Duration
	off        chan struct{}
	working    bool
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
		onTime:     time.Time{},
		duration:   0 * time.Second,
		durationCh: make(chan time.Duration, 1),
	}
}

// Configure sets up the Relay for use, beginning in the "Off" state
func (r *relay) Configure() {
	r.pin.Configure(machine.PinConfig{Mode: machine.PinOutput})
	r.Off()
	r.onTime = time.Now()
}

func (r *relay) DurationCh() chan time.Duration {
	return r.durationCh
}

// Execute acts on input from a trigger and along with relay.Name() implements the Triggerable interface
func (r *relay) Execute(t trigger.Trigger) {
	println("relay.Execute()...")
	if t.Target != r.name {
		t.Error = true
		println("error - " + r.name + " received a trigger intended for " + t.Target)
		t.Message = string("error - " + r.name + " received a trigger intended for " + t.Target)
		t.ReportCh <- t
		return
	}
	switch t.Action {
	case "On", "on", "ON":
		t.Error = false
		if !r.working {
			r.working = true
			go func() {
				r.durationCh = make(chan time.Duration, 1)
				r.off = make(chan struct{}, 1)
				defer r.reset()
				defer println("	relay.Execute() routine exiting.")
				defer println("	" + r.name + " working: " + strconv.FormatBool(r.working))
				defer println("	" + r.name + " duration: " + r.duration.String())
				defer println("	" + r.name + " onTime: " + r.onTime.Format(time.RFC822))

				r.onTime = time.Now()
				r.pin.High()

				// determined duration or indeterminate
				if t.Duration <= 0 { // sending a command with a negative or omitted duration will be treated as "indefinite on"
					t.Message = string(r.name + " - On indefinitely at " + r.onTime.Local().Format(time.RFC822))
					t.ReportCh <- t
					// return
				} else {
					r.duration = t.Duration
					t.Message = string(r.name + " - On for " + t.Duration.String() + " at " + r.onTime.Local().Format(time.RFC822))
					t.ReportCh <- t
				}

				// wait for communication or off time
				for {
					select {
					case <-r.off:
						r.pin.Low()
						t.Message = string(r.name + " - Forced Off after " + time.Since(r.onTime).String() + " at " + time.Now().Local().Format(time.RFC822))
						t.ReportCh <- t
						return
					case newDuration := <-r.durationCh:
						if t.Duration <= 0 {
							r.pin.Low()
							t.Message = string(r.name + " - Off after " + time.Since(r.onTime).String() + " at " + time.Now().Local().Format(time.RFC822))
							t.ReportCh <- t
							return
						}
						t.Message = string(r.name + " - Changing On duration to " + newDuration.String() + " (after " + time.Since(r.onTime).String() + " of a scheduled " + r.duration.String() + ") at " + time.Now().Local().Format(time.RFC822))
						r.duration = newDuration
						t.ReportCh <- t
					default:
						if r.duration > 0 {
							if time.Since(r.onTime) > r.duration {
								r.pin.Low()
								t.Message = string(r.name + " - Off after " + time.Since(r.onTime).String() + " at " + time.Now().Local().Format(time.RFC822))
								time.Sleep(100 * time.Millisecond)
								t.ReportCh <- t
								return
							}
						}
					}
				}
			}()
		}
		if t.Duration != r.duration {
			println("Sending new duration of " + t.Duration.String() + " to " + r.name)
			r.durationCh <- t.Duration
		}
		return
	case "Off", "off", "OFF":
		if r.pin.Get() { // if the "on" routine hasn't done so, force it off
			println("sending off signal to " + r.name)
			r.off <- struct{}{} // an existing "on" goroutine should be canceled & the relay reset
			time.Sleep(15 * time.Millisecond)
		}
		if r.pin.Get() {
			r.pin.Low()
			println("Off handler forcing " + r.name + " off")
			t.Message = string(r.name + " - Off! after " + time.Since(r.onTime).String() + " at " + time.Now().Local().Format(time.RFC822))
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
	r.onTime = time.Now()
	time.Sleep(5 * time.Millisecond)
	return r.pin.Get()
}

// On brings the Relays's pin high and returns a subsequent, measured confirmation
func (r *relay) On() bool {
	r.pin.High()
	r.onTime = time.Now()
	time.Sleep(5 * time.Millisecond)
	return r.pin.Get()
}

// Off brings the Relay's pin low and reutrns a subsequent, measured confirmation
func (r *relay) Off() bool {
	r.pin.Low()
	r.onTime = time.Now()
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
	return r.Get(), r.onTime
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
	ss.WriteString(r.onTime.String())
	return ss.String()
}

// Name returns the relay's name and along with relay.Execute() implements the Triggerable interface
func (r *relay) Name() string {
	return r.name
}

// reset zeroes the timing fields of a relay struct
func (r *relay) reset() {
	close(r.off)
	close(r.durationCh)
	r.duration = time.Duration(0)
	r.onTime = time.Time{}
	r.working = false
	println("					resetting " + r.name)
}
