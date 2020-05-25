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

package connection

import (
	"context"
	"net/http"
	"net/url"
	"path"
)

type RequestModifier func(r Request) error

func Call(ctx context.Context, c Connection, method, url string, output interface{}, modifiers ...RequestModifier) (Response, error) {
	req, err := c.NewRequest(method, url)
	if err != nil {
		return nil, err
	}

	for _, modifier := range modifiers {
		if err = modifier(req); err != nil {
			return nil, err
		}
	}

	return c.DoWithOutput(ctx, req, output)
}

func CallGet(ctx context.Context, c Connection, url string, output interface{}, modifiers ...RequestModifier) (Response, error) {
	return Call(ctx, c, http.MethodGet, url, output, modifiers...)
}

func CallPost(ctx context.Context, c Connection, url string, output interface{}, body interface{}, modifiers ...RequestModifier) (Response, error) {
	return Call(ctx, c, http.MethodPost, url, output, append(modifiers, WithBody(body))...)
}

func CallHead(ctx context.Context, c Connection, url string, output interface{}, modifiers ...RequestModifier) (Response, error) {
	return Call(ctx, c, http.MethodHead, url, output, modifiers...)
}

func CallPut(ctx context.Context, c Connection, url string, output interface{}, body interface{}, modifiers ...RequestModifier) (Response, error) {
	return Call(ctx, c, http.MethodPut, url, output, append(modifiers, WithBody(body))...)
}

func CallDelete(ctx context.Context, c Connection, url string, output interface{}, modifiers ...RequestModifier) (Response, error) {
	return Call(ctx, c, http.MethodDelete, url, output, modifiers...)
}

func WithBody(i interface{}) RequestModifier {
	return func(r Request) error {
		return r.SetBody(i)
	}
}

func NewUrl(parts ...string) string {
	s := make([]string, len(parts))
	for id, part := range parts {
		s[id] = url.PathEscape(part)
	}

	return path.Join(s...)
}
