//
// DISCLAIMER
//
// Copyright 2020-2024 ArangoDB GmbH, Cologne, Germany
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

package connection

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/arangodb/go-driver/v2/version"
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
// Deprecated: use applyArangoDBConfiguration instead
func applyGlobalSettings(ctx context.Context) RequestModifier {
	return func(r Request) error {

		// Set version header
		val := fmt.Sprintf("go-driver-v2/%s", version.DriverVersion())
		if ctx != nil {
			if v := ctx.Value(keyDriverFlags); v != nil {
				if flags, ok := v.([]string); ok {
					val = fmt.Sprintf("%s (%s)", val, strings.Join(flags, ","))
				}
			}
		}
		r.AddHeader("x-arango-driver", val)

		// Enable Queue timeout
		if ctx != nil {
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
		}

		return nil
	}
}

func applyArangoDBConfiguration(config ArangoDBConfiguration, ctx context.Context) RequestModifier {
	return func(r Request) error {
		// Set version header
		val := fmt.Sprintf("go-driver-v2/%s", version.DriverVersion())
		if len(config.DriverFlags) > 0 {
			val = fmt.Sprintf("%s (%s)", val, strings.Join(config.DriverFlags, ","))
		}
		r.AddHeader("x-arango-driver", val)

		if config.ArangoQueueTimeoutEnabled {
			if config.ArangoQueueTimeoutSec > 0 {
				r.AddHeader("x-arango-queue-time-seconds", fmt.Sprint(config.ArangoQueueTimeoutSec))
			} else if deadline, ok := ctx.Deadline(); ok {
				timeout := deadline.Sub(time.Now())
				r.AddHeader("x-arango-queue-time-seconds", fmt.Sprint(timeout.Seconds()))
			}
		}

		if config.Compression != nil && config.Compression.ResponseCompressionEnabled {
			if config.Compression.CompressionType == "gzip" {
				r.AddHeader("Accept-Encoding", "gzip")
			} else if config.Compression.CompressionType == "deflate" {
				r.AddHeader("Accept-Encoding", "deflate")
			}
		}

		return nil
	}
}
