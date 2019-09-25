package transform

import (
	"context"
	"fmt"
	"reflect"

	"golang.org/x/sync/errgroup"
)

type chain []Transformer

// Chain will create a Transformer from all given transformers
// input type is the input type of the first transformer in chain
// output type is the output type of the last transformer in chain
// if we say that tx is Transform function of x-th Transformer
// then the result is basically = tN(tN-1(...t2(t1(v))))
func Chain(ts ...Transformer) Transformer {
	switch len(ts) {
	case 0:
		panic("need at least one transformer to chain it")
	case 1:
		return ts[0]
	default:
		return chain(ts)
	}
}

// InputType is part of the Transformer interface
func (c chain) InputType() reflect.Type {
	return c[0].InputType()
}

// Transform is part of the Transformer interface
func (c chain) Transform(ctx context.Context, v interface{}, ch chan<- interface{}) error {
	inCh := make(chan interface{})
	go func(ch chan<- interface{}) {
		defer close(ch)
		ch <- v
	}(inCh)

	group, ctx := errgroup.WithContext(ctx)
	for i := range c {
		tmp := make(chan interface{})
		if i == len(c)-1 {
			group.Go(transformOne(ctx, c[i], inCh, ch, false)) // last transformer
		} else {
			group.Go(transformOne(ctx, c[i], inCh, tmp, true))
		}
		inCh = tmp
	}

	return group.Wait()
}

// runs the transformer on all values from inCh and sends them to outCh
// closes out channel when done if closeOut is true
func transformOne(ctx context.Context, t Transformer, inCh <-chan interface{}, outCh chan<- interface{}, closeOut bool) func() error {
	return func() error {
		if closeOut {
			defer close(outCh)
		}
		for {
			select {
			case iface, more := <-inCh:
				if !more {
					return nil
				}
				if err := t.Transform(ctx, iface, outCh); err != nil {
					return fmt.Errorf("transform failed: %v", err)
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	}
}
