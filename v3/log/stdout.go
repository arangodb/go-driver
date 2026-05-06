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

import "fmt"

func SetStdOutLogger() {
	SetLogger(&StdOutLogger{})
}

type StdOutLogger struct {
}

func (s StdOutLogger) log(level, msg string, args ...interface{}) {
	println(fmt.Sprintf(fmt.Sprintf("%s: %s", level, msg), args...))
}

func (s StdOutLogger) Trace(msg string) {
	s.log("TRACE", msg)
}

func (s StdOutLogger) Tracef(msg string, args ...interface{}) {
	s.log("TRACE", msg, args...)
}

func (s StdOutLogger) Debug(msg string) {
	s.log("DEBUG", msg)
}

func (s StdOutLogger) Debugf(msg string, args ...interface{}) {
	s.log("DEBUG", msg, args...)
}

func (s StdOutLogger) Info(msg string) {
	s.log("INFO", msg)
}

func (s StdOutLogger) Infof(msg string, args ...interface{}) {
	s.log("INFO", msg, args...)
}

func (s StdOutLogger) Error(err error, msg string) {
	s.log("ERROR", msg)
}

func (s StdOutLogger) Errorf(err error, msg string, args ...interface{}) {
	s.log("ERROR", msg, args...)
}
