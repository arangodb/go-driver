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

package connection

import (
	"context"
	"fmt"
	"time"
)

func WithTransactionID(transactionID string) RequestModifier {
	return func(r Request) error {
		r.AddHeader("x-arango-trx-id", transactionID)
		return nil
	}
}

func WithFragment(s string) RequestModifier {
	return func(r Request) error {
		r.SetFragment(s)
		return nil
	}
}

func WithQuery(s, value string) RequestModifier {
	return func(r Request) error {
		r.AddQuery(s, value)
		return nil
	}
}

// applyGlobalSettings applies the settings configured in the context to the given request.
func applyGlobalSettings(ctx context.Context) RequestModifier {
	return func(r Request) error {

		// Enable Queue timeout
		if v := ctx.Value(keyUseQueueTimeout); v != nil {
			if useQueueTimeout, ok := v.(bool); ok && useQueueTimeout {
				if v := ctx.Value(keyMaxQueueTime); v != nil {
					if timeout, ok := v.(time.Duration); ok {
						r.AddHeader("x-arango-queue-time-seconds", fmt.Sprint(timeout.Seconds()))
					}
				} else if deadline, ok := ctx.Deadline(); ok {
					timeout := deadline.Sub(time.Now())
					r.AddHeader("x-arango-queue-time-seconds", fmt.Sprint(timeout.Seconds()))
				}
			}
		}

		return nil
	}
}
