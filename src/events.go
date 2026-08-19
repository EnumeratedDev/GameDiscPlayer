package main

import "github.com/diamondburned/gotk4/pkg/core/glib"

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
			glib.IdleAdd(func() {
				f(args...)
			})
		}
	}
}
