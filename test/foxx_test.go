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
// Author Tomasz Mielech
//
package test

import (
	"context"
	"github.com/arangodb/go-driver"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

func getZipFile(url, path string) error {
	respZip, err := http.Get(url)
	if err != nil {
		return err
	}

	zipContent, err := ioutil.ReadAll(respZip.Body)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, zipContent, 0644)
}

func TestFoxxItzpapalotlService(t *testing.T) {

	if getTestMode() != testModeSingle {
		t.Skipf("Not a single")
	}

	zipFilePath := "/tmp/itzpapalotl-v1.2.0.zip"
	c := createClientFromEnv(t, false)

	err := getZipFile("https://github.com/arangodb-foxx/demo-itzpapalotl/archive/v1.2.0.zip", zipFilePath)
	require.NoError(t, err)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Minute*30)
	mountName := "test"
	options := driver.FoxxCreateOptions{
		Mount: "/" + mountName,
	}
	err = c.Foxx().InstallFoxxService(timeoutCtx, zipFilePath, options)
	cancel()
	require.NoError(t, err)

	timeoutCtx, cancel = context.WithTimeout(context.Background(), time.Second*30)
	connection := c.Connection()
	req, err := connection.NewRequest("GET", "_db/_system/"+mountName+"/random")
	require.NoError(t, err)
	resp, err := connection.Do(timeoutCtx, req)
	require.NotNil(t, resp)
	result := make(map[string]interface{}, 0)
	resp.ParseBody("", &result)
	require.NoError(t, err)
	value, ok := result["name"]
	require.Equal(t, true, ok)
	require.NotEmpty(t, value)
	cancel()

	timeoutCtx, cancel = context.WithTimeout(context.Background(), time.Second*30)
	deleteOptions := driver.FoxxDeleteOptions{
		Mount:    "/" + mountName,
		Teardown: true,
	}
	err = c.Foxx().UninstallFoxxService(timeoutCtx, deleteOptions)
	cancel()
	require.NoError(t, err)
}
