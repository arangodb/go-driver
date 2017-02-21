//
// DISCLAIMER
//
// Copyright 2017 ArangoDB GmbH, Cologne, Germany
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

package driver

type CreateUserOptions struct {
	// Loginname of the user to be created
	UserName string `json:"user,omitempty"`
	// The user password as a string. If not specified, it will default to an empty string.
	Password string `json:"passwd,omitempty"`
	// A flag indicating whether the user account should be activated or not. The default value is true. If set to false, the user won't be able to log into the database.
	Active *bool `json:"active,omitempty"`
	// A JSON object with extra user information. The data contained in extra will be stored for the user but not be interpreted further by ArangoDB.
	Extra interface{} `json:"extra,omitempty"`
}
