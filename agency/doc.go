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

//
// The Agency is fault-tolerant and highly-available key-value store
// that is used to store critical, low-level information about
// an ArangoDB cluster.
//
// The API provided in this package gives access to the Agency.
//
// THIS API IS NOT USED FOR NORMAL DATABASE ACCESS.
//
// Reasons for using this API are:
// - You want to make use of an indepent Agency as your own HA key-value store.
// - You want access to low-level information of your database. USE WITH GREAT CARE!
//
// WARNING: Messing around in the Agency can quickly lead to a corrupt database!
//
