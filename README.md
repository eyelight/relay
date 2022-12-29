# Relay
Package Relay provides functionality for employing an AC relay from a microcontroller, expects to be compiled with [TinyGo](https://tinygo.org), and implements the [Triggerable](https://github.com/eyelight/trigger) interface.

```go
type Triggerable interface {
    Name() string
    Execute(t Trigger)
}
```

### Behavior with package Trigger
When a Dispatcher receives a Trigger intended for a Relay it knows about, it calls the Relay's Execute method, passing along the Trigger.

If the `Trigger.Duration` is omitted, the `Trigger.Action` is interpreted as having indefinite duration. If a duration is included, the Relay's `Execute` method will spawn a goroutine that keeps the Relay's pin *high* for the intended duration. 

Subsequent Triggers received during the Relay's *on* duration will revise the intended duration, and if the new duration would be shorter than the time already elapsed, it will turn off the Relay, bringing its pin *low* and stopping the goroutine.

### Usage
Create & configure a new Relay
```go
r := relay.New(machine.D2, "KitchenLights")
r.Configure()
```

Create a Dispatcher and pass it a channel on which it should listen for Triggers
```go
ch := make(chan Trigger, 1)
d := trigger.NewDispatch(ch)
```

Add your Relay to the Dispatcher and start dispatching
```go
d.AddToDispatch(r)
go d.Dispatch()
```

Triggers can be sent via MQTT. If your MQTT handler sends messages to your Dispatcher, it will determine which of its *Triggerables* should execute the Trigger.

We can emulate this by creating a Trigger and sending it to our Dispatcher's channel:
```go
t := trigger.Trigger{}
t.Target = "KitchenLights"
t.Action = "On"
t.Duration = time.Duration(30 * time.Second)
ch <- t
```

The above snippet may fail because we haven't set `t.ReportCh` – a `chan Trigger` to which the Relay's `Execute` method will send the modified Trigger after taking the requested action. Specifically, `Execute` will typically send the Trigger back to the MQTT handler, having updated `t.Message` and possibly having set `t.Error`.