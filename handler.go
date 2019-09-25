package transform

import (
	"context"
	"log"
	"os"
)

// WithErrorHandler wraps the given transformer into a new one
// if the original transformer returns an error, it is passed to the error handler
// 	- if the handler returns nil, that's the same as if the transformer returned nil (it was handled)
//	- if the handler returns an error back (same or different), transform will return an error
func WithErrorHandler(t Transformer, errorHandler func(error) error) Transformer {
	return errorHandlingTransformer{t, errorHandler}
}

// LogErrors wraps the given transformer in a way that it logs all errors from it, but never fails
func LogErrors(t Transformer) Transformer {
	stdLog := log.New(os.Stderr, "transform: ", log.LstdFlags)
	errorHandler := func(err error) error {
		stdLog.Println(err)
		return nil
	}
	return WithErrorHandler(t, errorHandler)
}

type errorHandlingTransformer struct {
	Transformer
	errorHandler func(error) error
}

// Transform is a part of the Transformer interface
func (t errorHandlingTransformer) Transform(ctx context.Context, v interface{}, ch chan<- interface{}) error {
	if err := t.Transformer.Transform(ctx, v, ch); err != nil {
		return t.errorHandler(err)
	}
	return nil
}
