package transform

import (
	"context"
	"fmt"
	"reflect"
)

// Transformer is an interface which knows how to transform one interface to another
type Transformer interface {
	// InputType returns the input type this transformer operates on
	InputType() reflect.Type
	// Transform is the function which transforms interface to another (or multiple)
	// input interface will always be of type t.InputType()
	// write all outputs to the channel, without closing it
	// ctx.Done() should be monitored
	Transform(context.Context, interface{}, chan<- interface{}) error
}

// All will transform all values in the input channel and send them to the output channel
// input channel needs to be created and closed outside of this function
// Since this function is blocking, output channel can be closed when this function finishes
// Returns an error if transforming fails or context is done
func All(ctx context.Context, t Transformer, inCh <-chan interface{}, outCh chan<- interface{}) error {
	for {
		select {
		case v, more := <-inCh:
			if !more {
				return nil
			}
			if err := t.Transform(ctx, v, outCh); err != nil {
				return fmt.Errorf("transform(%+v) error: %v", v, err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
