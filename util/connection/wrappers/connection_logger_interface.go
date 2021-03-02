//
// DISCLAIMER
//
// Copyright 2021 ArangoDB GmbH, Cologne, Germany
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright holder is ArangoDB GmbH, Cologne, Germany
//
// Author Adam Janikowski
//

package wrappers

import (
	"encoding/json"
	"time"

	"github.com/rs/zerolog"
)

type Logger interface {
	Log() Event
}

type Event interface {
	Int(key string, value int) Event
	Str(key, value string) Event
	Time(key string, value time.Time) Event
	Duration(key string, value time.Duration) Event
	Interface(key string, value interface{}) Event

	Msgf(format string, args ...interface{})
}

var _ Logger = &zeroLogLogger{}

func NewZeroLogLogger(l zerolog.Logger) Logger {
	return &zeroLogLogger{log: l}
}

type zeroLogLogger struct {
	log zerolog.Logger
}

func (z zeroLogLogger) Log() Event {
	return newZeroLogEvent(z.log.Info())
}

func newZeroLogEvent(e *zerolog.Event) Event {
	return zeroLogEvent{e}
}

type zeroLogEvent struct {
	event *zerolog.Event
}

func (z zeroLogEvent) Int(key string, value int) Event {
	return newZeroLogEvent(z.event.Int(key, value))
}

func (z zeroLogEvent) Str(key, value string) Event {
	return newZeroLogEvent(z.event.Str(key, value))
}

func (z zeroLogEvent) Time(key string, value time.Time) Event {
	return newZeroLogEvent(z.event.Time(key, value))
}

func (z zeroLogEvent) Duration(key string, value time.Duration) Event {
	return newZeroLogEvent(z.event.Dur(key, value))
}

func (z zeroLogEvent) Interface(key string, value interface{}) Event {
	d, err := json.Marshal(value)
	if err != nil {
		return newZeroLogEvent(z.event.Str(key, err.Error()))
	} else {
		return newZeroLogEvent(z.event.Str(key, string(d)))
	}
}

func (z zeroLogEvent) Msgf(format string, args ...interface{}) {
	z.event.Msgf(format, args...)
}
