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

package log

var logger Log

func SetLogger(l Log) {
	logger = l
}

func Trace(msg string) {
	if logger != nil {
		logger.Trace(msg)
	}
}

func Tracef(msg string, args ...interface{}) {
	if logger != nil {
		logger.Tracef(msg, args...)
	}
}

func Debug(msg string) {
	if logger != nil {
		logger.Debug(msg)
	}
}

func Debugf(msg string, args ...interface{}) {
	if logger != nil {
		logger.Debugf(msg, args...)
	}
}

func Info(msg string) {
	if logger != nil {
		logger.Info(msg)
	}
}

func Infof(msg string, args ...interface{}) {
	if logger != nil {
		logger.Infof(msg, args...)
	}
}

func Error(err error, msg string) {
	if logger != nil {
		logger.Error(err, msg)
	}
}

func Errorf(err error, msg string, args ...interface{}) {
	if logger != nil {
		logger.Errorf(err, msg, args...)
	}
}
