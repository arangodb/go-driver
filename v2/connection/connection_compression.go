//
// DISCLAIMER
//
// Copyright 2024 ArangoDB GmbH, Cologne, Germany
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
	"compress/gzip"
	"compress/zlib"
	"io"

	"github.com/pkg/errors"

	"github.com/arangodb/go-driver/v2/log"
)

type Compression interface {
	ApplyRequestHeaders(r Request)
	ApplyRequestCompression(r *httpRequest, rootWriter io.Writer) (io.WriteCloser, error)
}

type compression struct {
	config *CompressionConfig
}

func newCompression(config *CompressionConfig) Compression {
	return &compression{
		config: config,
	}
}

func (g compression) ApplyRequestHeaders(r Request) {
	if g.config != nil && g.config.ResponseCompressionEnabled {
		if g.config.CompressionType == "gzip" {
			r.AddHeader("Accept-Encoding", "gzip")
		} else if g.config.CompressionType == "deflate" {
			r.AddHeader("Accept-Encoding", "deflate")
		}
	}
}

func (g compression) ApplyRequestCompression(r *httpRequest, rootWriter io.Writer) (io.WriteCloser, error) {
	config := g.config

	if config != nil && config.RequestCompressionEnabled {
		if config.CompressionType == "gzip" {
			r.headers["Content-Encoding"] = "gzip"

			gzipWriter, err := gzip.NewWriterLevel(rootWriter, config.RequestCompressionLevel)
			if err != nil {
				log.Errorf(err, "error creating gzip writer")
				return nil, err
			}
			return gzipWriter, nil
		} else if config.CompressionType == "deflate" {
			r.headers["Content-Encoding"] = "deflate"

			zlibWriter, err := zlib.NewWriterLevel(rootWriter, config.RequestCompressionLevel)
			if err != nil {
				log.Errorf(err, "error creating zlib writer")
				return nil, err
			}

			return zlibWriter, nil
		} else {
			return nil, errors.Errorf("unsupported compression type: %s", config.CompressionType)
		}
	}

	return nil, nil
}
