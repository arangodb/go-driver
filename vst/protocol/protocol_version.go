//
// DISCLAIMER
//
// Copyright 2017-2023 ArangoDB GmbH, Cologne, Germany
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

package protocol

import (
	"fmt"
)

// Version indicates the version of the Velocystream protocol
type Version int

const (
	Version1_0 Version = iota // VST 1.0
	Version1_1                // VST 1.1
)

func (v Version) asString() (string, error) {
	switch v {
	case Version1_0:
		return "vst/1.0", nil
	case Version1_1:
		return "vst/1.1", nil
	default:
		return "", fmt.Errorf("unknown protocol version %d", int(v))
	}
}
