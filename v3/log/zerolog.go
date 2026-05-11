//
// DISCLAIMER
//
// Copyright 2020 ArangoDB GmbH, Cologne, Germany
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

package log

import "github.com/rs/zerolog"

func SetZeroLogLogger(logger zerolog.Logger) {
	SetLogger(&ZeroLogLogger{log: logger})
}

type ZeroLogLogger struct {
	log zerolog.Logger
}

func (z ZeroLogLogger) Trace(msg string) {
	z.log.Trace().Msgf(msg)
}

func (z ZeroLogLogger) Tracef(msg string, args ...interface{}) {
	z.log.Trace().Msgf(msg, args...)
}

func (z ZeroLogLogger) Debug(msg string) {
	z.log.Debug().Msgf(msg)
}

func (z ZeroLogLogger) Debugf(msg string, args ...interface{}) {
	z.log.Debug().Msgf(msg, args...)
}

func (z ZeroLogLogger) Info(msg string) {
	z.log.Info().Msgf(msg)
}

func (z ZeroLogLogger) Infof(msg string, args ...interface{}) {
	z.log.Info().Msgf(msg, args...)
}

func (z ZeroLogLogger) Error(err error, msg string) {
	z.log.Error().Err(err).Msgf(msg)
}

func (z ZeroLogLogger) Errorf(err error, msg string, args ...interface{}) {
	z.log.Error().Err(err).Msgf(msg, args...)
}
