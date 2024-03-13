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
	"compress/zlib"
	"fmt"
	"io"

	"github.com/arangodb/go-driver/v2/log"
)

type Compression interface {
	ApplyRequestHeaders(r Request)
	ApplyRequestCompression(r *httpRequest, rootWriter io.Writer) (io.WriteCloser, error)
}

func newCompression(config *CompressionConfig) Compression {
	if config == nil {
		return noCompression{}
	} else if config.CompressionType == "gzip" {
		return gzipCompression{config: config}
	} else if config.CompressionType == "deflate" {
		return deflateCompression{config: config}
	} else {
		log.Error(fmt.Errorf("unknown compression type: %s", config.CompressionType), "")
		return noCompression{config: config}
	}
}

type gzipCompression struct {
	config *CompressionConfig
}

func (g gzipCompression) ApplyRequestHeaders(r Request) {
	if g.config != nil && g.config.ResponseCompressionEnabled {
		if g.config.CompressionType == "gzip" {
			r.AddHeader("Accept-Encoding", "gzip")
		}
	}
}

func (g gzipCompression) ApplyRequestCompression(r *httpRequest, rootWriter io.Writer) (io.WriteCloser, error) {
	config := g.config

	if config != nil && config.RequestCompressionEnabled {
		if config.CompressionType == "deflate" {
			r.headers["Content-Encoding"] = "deflate"

			zlibWriter, err := zlib.NewWriterLevel(rootWriter, config.RequestCompressionLevel)
			if err != nil {
				log.Errorf(err, "error creating zlib writer")
				return nil, err
			}

			return zlibWriter, nil
		}
	}

	return nil, nil
}

type deflateCompression struct {
	config *CompressionConfig
}

func (g deflateCompression) ApplyRequestHeaders(r Request) {
	if g.config != nil && g.config.ResponseCompressionEnabled {
		if g.config.CompressionType == "deflate" {
			r.AddHeader("Accept-Encoding", "deflate")
		}
	}
}

func (g deflateCompression) ApplyRequestCompression(r *httpRequest, rootWriter io.Writer) (io.WriteCloser, error) {
	config := g.config

	if config != nil && config.RequestCompressionEnabled {
		if config.CompressionType == "deflate" {
			r.headers["Content-Encoding"] = "deflate"

			zlibWriter, err := zlib.NewWriterLevel(rootWriter, config.RequestCompressionLevel)
			if err != nil {
				log.Errorf(err, "error creating zlib writer")
				return nil, err
			}

			return zlibWriter, nil
		}
	}

	return nil, nil
}

type noCompression struct {
	config *CompressionConfig
}

func (g noCompression) ApplyRequestHeaders(r Request) {
}

func (g noCompression) ApplyRequestCompression(r *httpRequest, rootWriter io.Writer) (io.WriteCloser, error) {
	return nil, nil
}
