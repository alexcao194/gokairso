package realtime

type fanout []Broadcaster

// Fanout returns a Broadcaster that forwards every event to all targets, so
// extra consumers (e.g. outgoing webhooks) see the same stream as the hub.
func Fanout(targets ...Broadcaster) Broadcaster {
	return fanout(targets)
}

func (f fanout) Broadcast(event Event) {
	for _, t := range f {
		t.Broadcast(event)
	}
}
