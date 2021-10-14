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

package test

import (
	"context"
	"net/http"

	"github.com/arangodb/go-driver"
)

type driverErrorCheckFunc func(err error) (bool, error)
type driverErrorChecker func(ctx context.Context, client driver.Client) error

func driverErrorCheck(ctx context.Context, c driver.Client, checker driverErrorChecker, checks ...driverErrorCheckFunc) retryFunc {
	return func() error {
		err := checker(ctx, c)

		for _, check := range checks {
			if valid, err := check(err); err != nil {
				return err
			} else if !valid {
				return nil
			}
		}

		return interrupt{}
	}
}

func driverErrorCheckRetry503(err error) (bool, error) {
	if err == nil {
		return true, nil
	}

	if ae, ok := driver.AsArangoError(err); !ok {
		return false, err
	} else {
		if !ae.HasError {
			return true, nil
		}
		switch ae.Code {
		case http.StatusServiceUnavailable:
			return false, nil
		default:
			return true, err
		}
	}
}
