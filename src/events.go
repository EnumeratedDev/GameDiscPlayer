package main

var events = make(map[string][]func(args ...any))

func EventSubscribe(eventID string, f func(args ...any)) {
	events[eventID] = append(events[eventID], f)
}

func EventEmit(eventID string, args ...any) {
	for k, v := range events {
		if k != eventID {
			continue
		}

		for _, f := range v {
			f(args...)
		}
	}
}
