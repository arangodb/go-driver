//
// DISCLAIMER
//
// Copyright 2020-2023 ArangoDB GmbH, Cologne, Germany
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
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/kkdai/maglev"
	"github.com/pkg/errors"
)

// RequestHashValueExtractor accepts request method and full request path and must return a value which will be used for hash calculation
type RequestHashValueExtractor func(requestMethod, requestPath string) (string, error)

// NewMaglevHashEndpoints returns Endpoint manager which consistently returns the same endpoint based on value
// extracted from request using provided RequestHashValueExtractor
// e.g. if you want to use DB name from URL for hashing you can use RequestDBNameValueExtractor
func NewMaglevHashEndpoints(eps []string, extractor RequestHashValueExtractor) (Endpoint, error) {
	// order of endpoints affects hashing result
	sort.Strings(eps)

	// lookupSize must be equal or greater than len(eps) and it must be a prime number
	lookupSize := findNextPrime(uint64(len(eps)))

	table, err := maglev.NewMaglev(eps, lookupSize)
	if err != nil {
		return nil, err
	}

	return &maglevHashEndpoints{
		extractor:   extractor,
		endpoints:   eps,
		maglevTable: table,
	}, nil
}

func findNextPrime(i uint64) uint64 {
	bigInt := big.NewInt(0).SetUint64(i)

	for {
		if bigInt.ProbablyPrime(1) {
			return bigInt.Uint64()
		}
		i++
		bigInt.SetUint64(i)
	}
}

type maglevHashEndpoints struct {
	extractor   RequestHashValueExtractor
	endpoints   []string
	maglevTable *maglev.Maglev
}

func (e *maglevHashEndpoints) List() []string {
	return e.endpoints
}

func (e *maglevHashEndpoints) Get(providedEp, requestMethod, requestPath string) (string, error) {
	if len(e.endpoints) == 0 {
		return "", errors.New("no endpoints known")
	}

	for _, known := range e.endpoints {
		if known == providedEp {
			return known, nil
		}
	}

	val, err := e.extractor(requestMethod, requestPath)
	if err != nil {
		return "", errors.WithMessagef(err, "could not extract value for method '%s' path '%s'", requestMethod, requestPath)
	}

	r, err := e.maglevTable.Get(val)
	if err != nil {
		return r, errors.WithMessage(err, "failed to lookup Maglev table")
	}

	return r, nil
}

var _ RequestHashValueExtractor = RequestDBNameValueExtractor

// RequestDBNameValueExtractor might be used as RequestHashValueExtractor to use DB name from URL for hashing
// It fallbacks to requestMethod+requestPath concatenation in case if path does not contain DB name
func RequestDBNameValueExtractor(requestMethod, requestPath string) (string, error) {
	// most go-driver requests to ArangoDB are executed against `_db/<db-name>/xxxx/yyy/`) URL pattern
	// we can try to extract db-name to load-balance requests between endpoints

	parts := strings.Split(strings.Trim(strings.TrimSpace(requestPath), "/"), "/")
	if len(parts) >= 3 {
		if parts[0] == "_db" {
			return parts[1], nil
		}
	}
	return fmt.Sprintf("%s_%s", requestMethod, requestPath), nil
}
