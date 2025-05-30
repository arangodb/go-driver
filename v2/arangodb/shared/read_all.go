package shared

import (
	"errors"
	"reflect"
)

type readReader[T any] interface {
	Read() (T, error)
}

type ReadAllReadable[T any] interface {
	ReadAll() ([]T, []error)
}

type ReadAllReader[T any, R readReader[T]] struct {
	Reader R
}

func (r ReadAllReader[T, R]) ReadAll() ([]T, []error) {
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

type readReaderInto[T any] interface {
	Read(i interface{}) (T, error)
}

type ReadAllIntoReadable[T any] interface {
	ReadAll(i interface{}) ([]T, []error)
}

type ReadAllIntoReader[T any, R readReaderInto[T]] struct {
	Reader R
}

func (r ReadAllIntoReader[T, R]) ReadAll(i interface{}) ([]T, []error) {

	iVal := reflect.ValueOf(i)
	if iVal.Kind() != reflect.Ptr || iVal.Elem().Kind() != reflect.Slice {
		panic("i must be a pointer to a slice")
	}

	eVal := iVal.Elem()
	eType := eVal.Type().Elem()

	var docs []T
	var errs []error

	for {
		res := reflect.New(eType)
		doc, e := r.Reader.Read(res.Interface())
		if errors.Is(e, NoMoreDocumentsError{}) {
			break
		}

		iDocVal := reflect.ValueOf(doc)
		if iDocVal.Kind() == reflect.Ptr {
			iDocVal = iDocVal.Elem()
		}
		docCopy := reflect.New(iDocVal.Type()).Elem()
		docCopy.Set(iDocVal)

		errs = append(errs, e)
		docs = append(docs, docCopy.Interface().(T))
		eVal = reflect.Append(eVal, res.Elem())
	}
	iVal.Elem().Set(eVal)
	return docs, errs
}
