// DISCLAIMER
//
// Copyright 2020-2025 ArangoDB GmbH, Cologne, Germany
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

package shared

import (
	"errors"
	"fmt"
	"reflect"
)

// readReader represents a reader that can read a single document at a time.
//
// Implementations should return NoMoreDocumentsError when no more documents
// are available. Any other error is considered non-terminal and may be
// returned together with a document.
type readReader[T any] interface {
	Read() (T, error)
}

// ReadAllReadable represents a reader that can read all remaining documents
// in a single call.
//
// Implementations should continue reading until NoMoreDocumentsError is
// encountered and return all read documents along with any non-terminal errors.
type ReadAllReadable[T any] interface {
	ReadAll() ([]T, []error)
}

// ReadAllReader wraps a readReader and provides a helper to read all documents
// until NoMoreDocumentsError is returned.
//
// It repeatedly calls Read on the underlying Reader, collecting all returned
// documents and errors. Reading stops when NoMoreDocumentsError is encountered.
//
// ReadAllReader is not safe for concurrent use unless the underlying Reader
// implementation explicitly supports concurrent access.
type ReadAllReader[T any, R readReader[T]] struct {
	Reader R
}

// ReadAll reads all remaining documents from the underlying Reader.
//
// It continues calling Read until NoMoreDocumentsError is returned.
// All documents read before termination are returned along with any
// non-terminal errors encountered during reading.
//
// If Read returns both a document and an error, both values are included
// in the returned slices.
func (r *ReadAllReader[T, R]) ReadAll() ([]T, []error) {
	var docs []T
	var errs []error
	for {
		doc, e := r.Reader.Read()
		if errors.Is(e, NoMoreDocumentsError{}) {
			break
		}
		errs = append(errs, e)
		docs = append(docs, doc)
	}
	return docs, errs
}

// readReaderInto represents a reader that reads a single document into a
// user-provided destination object.
//
// The provided value must be a pointer to a compatible type expected by
// the reader implementation.
type readReaderInto[T any] interface {
	Read(i interface{}) (T, error)
}

// ReadAllIntoReadable represents a reader that can read all remaining documents
// into a user-provided destination object.
//
// Implementations should read until NoMoreDocumentsError is encountered and
// return all read documents along with any non-terminal errors.
type ReadAllIntoReadable[T any] interface {
	ReadAll(i interface{}) ([]T, []error)
}

// ReadAllIntoReader wraps a readReaderInto and provides bulk reading support
// into a caller-provided slice.
//
// This type is intended for use cases where documents must be decoded or
// unmarshaled into existing objects for backward compatibility or custom
// processing.
//
// ReadAllIntoReader is not safe for concurrent use unless the underlying
// Reader and the destination object are safe for concurrent access.
type ReadAllIntoReader[T any, R readReaderInto[T]] struct {
	Reader R
}

// ReadAll reads all remaining documents into the provided slice pointer.
//
// The parameter i must be a non-nil pointer to a slice. Each document is
// read into a newly allocated element of the slice's element type, which
// is then appended to the slice.
//
// Reading stops when NoMoreDocumentsError is encountered. All successfully
// read documents (metadata) and any non-terminal errors are returned.
//
// If i is not a pointer to a slice, ReadAll returns an error.
func (r *ReadAllIntoReader[T, R]) ReadAll(i interface{}) ([]T, []error) {
	iVal := reflect.ValueOf(i)
	if !iVal.IsValid() {
		return nil, []error{errors.New("i must be a pointer to a slice, got nil")}
	}
	if iVal.Kind() != reflect.Ptr {
		return nil, []error{fmt.Errorf("i must be a pointer to a slice, got %s", iVal.Kind())}
	}
	if iVal.IsNil() {
		return nil, []error{errors.New("i must be a pointer to a slice, got nil pointer")}
	}
	eVal := iVal.Elem()
	if eVal.Kind() != reflect.Slice {
		return nil, []error{fmt.Errorf("i must be a pointer to a slice, got pointer to %s", eVal.Kind())}
	}

	eType := eVal.Type().Elem()

	var docs []T
	var errs []error

	for {
		res := reflect.New(eType)
		doc, e := r.Reader.Read(res.Interface())
		if errors.Is(e, NoMoreDocumentsError{}) {
			break
		}

		errs = append(errs, e)
		docs = append(docs, doc)
		eVal = reflect.Append(eVal, res.Elem())
	}
	iVal.Elem().Set(eVal)
	return docs, errs
}
