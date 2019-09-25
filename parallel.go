package transform

import (
	"context"
	"reflect"

	"golang.org/x/sync/errgroup"
)

type parallel []Transformer

// InParallel will create a Transformer from all given transformers
// input type of all transformers should be the same, and that's the input type for this transformer
// output type of all of them should be the same, and that's the output type of the transformer
// transform is accomplished by running all given transformers in parallel on the same input data
func InParallel(ts ...Transformer) Transformer {
	switch len(ts) {
	case 0:
		panic("need at least one transformer to chain it")
	case 1:
		return ts[0]
	default:
		typ := ts[0].InputType()
		for _, t := range ts {
			if t.InputType() != typ {
				panic("not all transformers have the same input type")
			}
		}
		return parallel(ts)
	}
}

// InputType is part of the Transformer interface
func (p parallel) InputType() reflect.Type {
	return p[0].InputType()
}

// Transform is part of the Transformer interface
func (p parallel) Transform(ctx context.Context, v interface{}, ch chan<- interface{}) error {
	group, ctx := errgroup.WithContext(ctx)
	for _, t := range p {
		t := t
		group.Go(func() error { return t.Transform(ctx, v, ch) })
	}
	return group.Wait()
}
