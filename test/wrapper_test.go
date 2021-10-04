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
	"strings"
	"time"

	"github.com/arangodb/go-driver"
)

func WrapLogger(t testEnv, c driver.Connection) driver.Connection {
	return &logWrapper{
		t: t,
		c: c,
	}
}

type logWrapper struct {
	t testEnv

	c driver.Connection
}

func (l logWrapper) NewRequest(method, path string) (driver.Request, error) {
	return l.c.NewRequest(method, path)
}

func (l logWrapper) Do(ctx context.Context, req driver.Request) (driver.Response, error) {
	t := time.Now()

	if resp, err := l.c.Do(ctx, req); err != nil {
		l.t.Logf("Request (UNKNOWN)\t%s\t%s (%s): Failed %s", strings.ToUpper(req.Method()), req.Path(), time.Now().Sub(t).String(), err.Error())
		return resp, err
	} else {
		l.t.Logf("Request (%s)\t%s\t%s (%s): Code %d", resp.Endpoint(), strings.ToUpper(req.Method()), req.Path(), time.Now().Sub(t).String(), resp.StatusCode())
		return resp, err
	}
}

func (l logWrapper) Unmarshal(data driver.RawObject, result interface{}) error {
	return l.c.Unmarshal(data, result)
}

func (l logWrapper) Endpoints() []string {
	return l.c.Endpoints()
}

func (l logWrapper) UpdateEndpoints(endpoints []string) error {
	return l.c.UpdateEndpoints(endpoints)
}

func (l logWrapper) SetAuthentication(authentication driver.Authentication) (driver.Connection, error) {
	if c, err := l.c.SetAuthentication(authentication); err != nil {
		return nil, err
	} else {
		return WrapLogger(l.t, c), nil
	}
}

func (l logWrapper) Protocols() driver.ProtocolSet {
	return l.c.Protocols()
}
