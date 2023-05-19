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

package arangodb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertServerRole(t *testing.T) {
	type args struct {
		arangoDBRole string
	}
	tests := map[string]struct {
		args args
		want ServerRole
	}{
		"coordinator": {
			args: args{
				arangoDBRole: "COORDINATOR",
			},
			want: ServerRoleCoordinator,
		},
		"single": {
			args: args{
				arangoDBRole: "SINGLE",
			},
			want: ServerRoleSingle,
		},
		"agent": {
			args: args{
				arangoDBRole: "AGENT",
			},
			want: ServerRoleAgent,
		},
		"primary": {
			args: args{
				arangoDBRole: "PRIMARY",
			},
			want: ServerRoleDBServer,
		},
		"undefined": {
			args: args{
				arangoDBRole: "invalid",
			},
			want: ServerRoleUndefined,
		},
	}

	for testName, test := range tests {
		t.Run(testName, func(t *testing.T) {
			got := ConvertServerRole(test.args.arangoDBRole)
			assert.Equal(t, test.want, got)
		})
	}
}
