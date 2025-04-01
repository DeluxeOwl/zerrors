package zerrors

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/emirpasic/gods/v2/sets/hashset"
)

type Error[T ~string] struct {
	code       T
	wrappedErr error
	tags       *hashset.Set[string]
	data       map[string]any
	stack      *stack
}

// New creates a new Error instance.
func New[T ~string](code T) *Error[T] {
	return &Error[T]{
		code:       code,
		wrappedErr: nil,
		data:       map[string]any{},
		tags:       hashset.New[string](),
		stack:      captureStack(1),
	}
}

func (e *Error[T]) LogValue() slog.Value {
	// Create base attributes
	attrs := []slog.Attr{
		slog.String("code", string(e.code)),
		slog.String("error", e.Error()),
	}

	// Add data group if there's any custom data
	if len(e.data) > 0 {
		// Convert map entries directly to key-value pairs for slog.Group
		//nolint:mnd // 2 is the pair nr
		dataArgs := make([]any, 0, len(e.data)*2)
		for k, v := range e.data {
			dataArgs = append(dataArgs, k, v)
		}
		attrs = append(attrs, slog.Group("data", dataArgs...))
	}

	if !e.tags.Empty() {
		attrs = append(attrs, slog.Any("tags", e.GetTags()))
	}

	// Handle wrapped error
	if e.wrappedErr != nil {
		if logValuer, ok := e.wrappedErr.(slog.LogValuer); ok {
			attrs = append(attrs, slog.Any("wrapped", logValuer.LogValue()))
		} else {
			attrs = append(attrs, slog.String("wrapped", e.wrappedErr.Error()))
		}
	}

	if e.stack != nil {
		attrs = append(attrs, slog.String("stack", e.stack.String()))
	}

	return slog.GroupValue(attrs...)
}

func (e *Error[T]) Tags(tags ...string) *Error[T] {
	e.tags.Add(tags...)
	return e
}

func (e *Error[T]) HasTags(tags ...string) bool {
	return e.tags.Contains(tags...)
}

func (e *Error[T]) GetTags() []string {
	return e.tags.Values()
}

func (e *Error[T]) With(k string, v any) *Error[T] {
	e.data[k] = v
	return e
}

func (e *Error[T]) Get(key string) (any, bool) {
	val, ok := e.data[key]
	return val, ok
}

// WithError wraps an existing error.
func (e *Error[T]) WithError(err error) *Error[T] {
	e.wrappedErr = err

	// Propagate the tags
	if wrappedErr, ok := err.(interface{ GetTags() []string }); ok {
		e.tags.Add(wrappedErr.GetTags()...)
	}

	return e
}

// Errorf formats and wraps an error message.
func (e *Error[T]) Errorf(format string, a ...any) *Error[T] {
	e.wrappedErr = fmt.Errorf(format, a...)
	return e
}

// Code returns the error code.
//
//nolint:ireturn // This is fine
func (e *Error[T]) Code() T {
	return e.code
}

func (e *Error[T]) CodeString() string {
	return string(e.code)
}

// Error implements the error interface.
func (e *Error[T]) Error() string {
	if e.wrappedErr != nil {
		return fmt.Sprintf("%s: %s", e.code, e.wrappedErr.Error())
	}
	return string(e.code)
}

// Unwrap implements error unwrapping.
func (e *Error[T]) Unwrap() error {
	return e.wrappedErr
}

// Is implements error comparison.
func (e *Error[T]) Is(target error) bool {
	t, ok := target.(*Error[T])
	if !ok {
		return false
	}
	return e.code == t.code
}

// As implements error casting.
func (e *Error[T]) As(target any) bool {
	if targetErr, ok := target.(**Error[T]); ok {
		*targetErr = e
		return true
	}

	if e.wrappedErr != nil {
		if asErr, ok := e.wrappedErr.(interface{ As(any) bool }); ok {
			return asErr.As(target)
		}
	}

	return false
}

// As implements error casting with a callback.
func As[T ~string, V any](err error, fn func(zerr *Error[T]) V) (*V, bool) {
	var zerr *Error[T]
	if errors.As(err, &zerr) {
		val := fn(zerr)
		return &val, true
	}
	var empty *V
	return empty, false
}

func HasCode[T ~string](err error, code T) bool {
	var e *Error[T]
	if errors.As(err, &e) {
		return e.Code() == code
	}
	return false
}
