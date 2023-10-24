//
// DISCLAIMER
//
// Copyright 2023 ArangoDB GmbH, Cologne, Germany
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

package async

import (
	"errors"
	"fmt"
)

func IsAsyncJobInProgress(err error) (string, bool) {
	if err == nil {
		return "", false
	}

	var v ErrorAsyncJobInProgress
	if errors.As(err, &v) {
		return v.jobID, true
	}
	return "", false
}

type ErrorAsyncJobInProgress struct {
	jobID string
}

func (a ErrorAsyncJobInProgress) Error() string {
	return fmt.Sprintf("Job with ID %s in progress", a.jobID)
}
