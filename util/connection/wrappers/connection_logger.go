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

package wrappers

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver"
)

func NewLoggerConnection(c driver.Connection, l Logger, truncate bool) driver.Connection {
	return logConnection{
		ConnectionID: NewID(),
		started:      time.Now(),
		connection:   c,
		logger:       l,
		truncate:     truncate,
	}
}

var _ driver.Connection = &logConnection{}

type logConnection struct {
	ConnectionID ID

	started time.Time

	connection driver.Connection

	truncate bool

	logger Logger
}

func (l logConnection) NewRequest(method, path string) (driver.Request, error) {
	if r, err := l.connection.NewRequest(method, path); err != nil {
		return nil, err
	} else {
		lr := l.wrapRequest(r)
		lr.withLogger().Msgf("Request created")
		return lr, nil
	}
}

func (l logConnection) getJsonFromData(data []byte) ([]byte, error) {
	var i interface{} = struct{}{}

	if err := l.connection.Unmarshal(data, &i); err != nil {
		return nil, err
	}

	return json.Marshal(i)
}

func (l logConnection) Do(ctx context.Context, req driver.Request) (driver.Response, error) {
	if r, ok := req.(*logRequest); !ok {
		return nil, errors.Errorf("Invalid type of request")
	} else {
		t := time.Now()
		r.withLogger().Msgf("Execute request")

		var d []byte

		cCtx := driver.WithRawResponse(ctx, &d)

		resp, err := l.connection.Do(cCtx, r.request)
		if err != nil {
			r.withLogger().Str("Error", err.Error()).Msgf("Request failed")
			return nil, err
		}

		d, err = l.getJsonFromData(d)
		if err != nil {
			return nil, err
		}

		if l.truncate && len(d) > 128 {
			d = d[:128]
		}

		r.withLogger().Int("Code", resp.StatusCode()).Str("Response", string(d)).Duration("DurationOfCall", time.Now().Sub(t)).Msgf("Request completed")

		return resp, nil
	}
}

func (l logConnection) Unmarshal(data driver.RawObject, result interface{}) error {
	return l.connection.Unmarshal(data, result)
}

func (l logConnection) Endpoints() []string {
	return l.connection.Endpoints()
}

func (l logConnection) UpdateEndpoints(endpoints []string) error {
	return l.connection.UpdateEndpoints(endpoints)
}

func (l logConnection) SetAuthentication(authentication driver.Authentication) (driver.Connection, error) {
	c, err := l.connection.SetAuthentication(authentication)
	if err != nil {
		return nil, err
	}

	return l.copyWith(c), nil
}

func (l logConnection) Protocols() driver.ProtocolSet {
	return l.connection.Protocols()
}

func (l logConnection) wrapRequest(r driver.Request) *logRequest {
	return &logRequest{
		ID:                NewID(),
		ConnectionID:      l.ConnectionID,
		request:           r,
		connectionStarted: l.started,
		started:           time.Now(),
		logger:            l.logger,
		truncate:          l.truncate,
	}
}

func (l logConnection) copyWith(c driver.Connection) *logConnection {
	return &logConnection{
		ConnectionID: l.ConnectionID,
		started:      l.started,
		connection:   c,
		logger:       l.logger,
	}
}

var _ driver.Request = &logRequest{}

type logRequest struct {
	ID           ID
	ConnectionID ID

	request driver.Request

	connectionStarted time.Time
	started           time.Time

	logger Logger

	truncate bool
}

func (l logRequest) copyWith(d driver.Request) *logRequest {
	return &logRequest{
		ID:           l.ID,
		ConnectionID: l.ConnectionID,
		request:      d,
		started:      l.started,
		logger:       l.logger,
	}
}

func (l logRequest) withLogger() Event {
	t := time.Now()
	return l.logger.Log().
		Str("ConnectionID", l.ConnectionID.String()).
		Str("RequestID", l.ID.String()).
		Time("CurrentTime", t).
		Duration("ConnectionDuration", t.Sub(l.connectionStarted)).
		Duration("RequestDuration", t.Sub(l.started)).
		Str("Method", l.request.Method()).
		Str("Path", l.request.Path())
}

func (l logRequest) SetQuery(key, value string) driver.Request {
	l.withLogger().Str("Key", key).Str("Value", value).Msgf("Added Query")
	return l.copyWith(l.request.SetQuery(key, value))
}

func (l logRequest) SetBody(body ...interface{}) (driver.Request, error) {
	d, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	if l.truncate && len(d) > 128 {
		d = d[:128]
	}
	l.withLogger().Str("Body", string(d)).Msgf("Body set")

	if r, err := l.request.SetBody(body...); err != nil {
		return nil, err
	} else {
		return l.copyWith(r), nil
	}
}

func (l logRequest) SetBodyArray(bodyArray interface{}, mergeArray []map[string]interface{}) (driver.Request, error) {
	d, err := json.Marshal(struct {
		Body       interface{}
		MergeArray []map[string]interface{}
	}{
		Body:       bodyArray,
		MergeArray: mergeArray,
	})
	if err != nil {
		return nil, err
	}

	if l.truncate && len(d) > 128 {
		d = d[:128]
	}
	l.withLogger().Str("BodyArray", string(d)).Interface("MergeArray", mergeArray).Msgf("Body Array set")
	if r, err := l.request.SetBodyArray(bodyArray, mergeArray); err != nil {
		return nil, err
	} else {
		return l.copyWith(r), nil
	}
}

func (l logRequest) SetBodyImportArray(bodyArray interface{}) (driver.Request, error) {
	l.withLogger().Interface("BodyArray", bodyArray).Msgf("Body Import Array set")
	if r, err := l.request.SetBodyImportArray(bodyArray); err != nil {
		return nil, err
	} else {
		return l.copyWith(r), nil
	}
}

func (l logRequest) SetHeader(key, value string) driver.Request {
	f := l.withLogger().Str("Key", key)
	if strings.ToLower(key) == "authorization" {
		f = f.Str("Value", "hidden")
	} else {
		f = f.Str("Value", value)
	}
	f.Msgf("Added Header")
	return l.copyWith(l.request.SetHeader(key, value))
}

func (l logRequest) Written() bool {
	return l.request.Written()
}

func (l logRequest) Clone() driver.Request {
	return l.copyWith(l.request.Clone())
}

func (l logRequest) Path() string {
	return l.Path()
}

func (l logRequest) Method() string {
	return l.Method()
}
