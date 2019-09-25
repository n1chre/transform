package transform

import (
	"context"
	"fmt"
	"reflect"
)

type function struct {
	// functions input type
	inputType reflect.Type
	// function
	f reflect.Value
	// does the input need context
	inputContext bool
	// does the output contain an error
	outputError bool
}

// InputType is part of the Transformer interface
func (t *function) InputType() reflect.Type {
	return t.inputType
}

// Transform is part of the Transformer interface
func (t *function) Transform(ctx context.Context, v interface{}, ch chan<- interface{}) error {
	in := []reflect.Value{reflect.ValueOf(v)}
	if t.inputContext {
		// prepend context
		in = append([]reflect.Value{reflect.ValueOf(ctx)}, in[0])
	}

	out := t.f.Call(in)
	if t.outputError {
		// check if error occurred, it's the second field in output
		if !out[1].IsNil() {
			return out[1].Interface().(error)
		}
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case ch <- out[0].Interface():
		return nil
	}
}

// FromFunction constructs a Transformer from given function
// function must take a single input, but can also take a context.Context as the first argument
// function must have a single output, but can also output error as the second output
// if it takes ctx as input, it must have error in it's output. without it doesn't make sense
//
// possible function signatures:
//	1) func(any) any
//	2) func(any) (any, error)
//	3) func(ctx, any) (any, error)
func FromFunction(v interface{}) (Transformer, error) {
	f := reflect.ValueOf(v)
	t := f.Type()

	if t.Kind() != reflect.Func {
		return nil, fmt.Errorf("argument should be a function, got %s", t.Kind())
	}

	transformer := &function{f: f}

	errorType := reflect.TypeOf((*error)(nil)).Elem()
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()

	switch t.NumIn() {
	case 1:
		transformer.inputType = t.In(0)
	case 2:
		if !t.In(0).Implements(contextType) {
			return nil, fmt.Errorf("first argument must implement context.Context, got %s", t.In(0))
		}
		transformer.inputType = t.In(1)
		transformer.inputContext = true
	default:
		return nil, fmt.Errorf("function can have either 1 (any) or 2 (ctx, any) input arguments, got %d", t.NumIn())
	}

	switch t.NumOut() {
	case 1:
		if transformer.inputContext {
			return nil, fmt.Errorf("function takes in context as input, but doesn't have error in it's output")
		}
	case 2:
		if !t.Out(1).Implements(errorType) {
			return nil, fmt.Errorf("second output must implement error, got %s", t.Out(1))
		}
		transformer.outputError = true
	default:
		return nil, fmt.Errorf("function can have either 1 (any) or 2 (any, error) outputs, got %d", t.NumOut())
	}

	return transformer, nil
}
