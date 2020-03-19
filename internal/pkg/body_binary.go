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
// Author Tomasz Mielech
//

package pkg

import (
	"fmt"
	"github.com/arangodb/go-driver"
)

type binaryBody struct {
	body        []byte
	contentType string
}

func NewBinaryBodyBuilder(contentType string) *binaryBody {
	b := binaryBody{
		contentType: contentType,
	}
	return &b
}

// SetBody sets the content of the request.
// The protocol of the connection determines what kinds of marshalling is taking place.
func (b *binaryBody) SetBody(body ...interface{}) error {
	if len(body) == 0 {
		return driver.WithStack(fmt.Errorf("must provide at least 1 body"))
	}

	if data, ok := body[0].([]byte); ok {
		b.body = data
		return nil
	}

	return driver.WithStack(fmt.Errorf("must provide body as a []byte type"))
}

func (b *binaryBody) SetBodyArray(_ interface{}, _ []map[string]interface{}) error {
	return nil
}

func (b *binaryBody) SetBodyImportArray(_ interface{}) error {
	return nil
}

func (b *binaryBody) GetBody() []byte {
	return b.body
}

func (b *binaryBody) GetContentType() string {
	return b.contentType
}

func (b *binaryBody) Clone() driver.BodyBuilder {
	return &binaryBody{
		body:        b.GetBody(),
		contentType: b.GetContentType(),
	}
}
