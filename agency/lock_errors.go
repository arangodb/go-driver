//
// DISCLAIMER
//
// Copyright 2018 ArangoDB GmbH, Cologne, Germany
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
// Author Ewout Prangsma
//

package agency

import (
	"errors"

	driver "github.com/arangodb/go-driver"
)

var (
	// AlreadyLockedError indicates that the lock is already locked.
	AlreadyLockedError = errors.New("already locked")
	// NotLockedError indicates that the lock is not locked when trying to unlock.
	NotLockedError = errors.New("not locked")
)

// IsAlreadyLocked returns true if the given error is or is caused by an AlreadyLockedError.
func IsAlreadyLocked(err error) bool {
	return driver.Cause(err) == AlreadyLockedError
}

// IsNotLocked returns true if the given error is or is caused by an NotLockedError.
func IsNotLocked(err error) bool {
	return driver.Cause(err) == NotLockedError
}
