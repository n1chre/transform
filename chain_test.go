package transform

import (
	"context"
	"fmt"
	"reflect"
	"testing"
)

type chainTest int

func (chainTest) InputType() reflect.Type {
	return reflect.TypeOf(1) // int
}

func (t chainTest) Transform(_ context.Context, v interface{}, ch chan<- interface{}) error {
	// replicate the input value + 1 as many times as the transformer specifies
	v = v.(int) + 1
	for i := 0; i < int(t); i++ {
		ch <- v
	}
	return nil
}

func TestChain(t *testing.T) {
	// each transformer creates as many outputs as the number says, and the output is input+1
	// so chainTest(2) will transform each input x into x+1, and output 2 of those

	for i, chainLen := range []int{1, 2, 3, 5, 8} {
		t.Run(fmt.Sprintf("chain-len-%d", i+1), func(t *testing.T) {

			numOutputs := 1
			ts := make([]Transformer, chainLen)
			for i := 0; i < chainLen; i++ {
				c := i + 1
				ts[i] = chainTest(c)
				numOutputs *= c // why it's like this, is left for the reader to find out
			}
			tt := Chain(ts...)

			for _, input := range []int{2, 4, 8} {
				t.Run(fmt.Sprintf("input-%d", input), func(t *testing.T) {
					ch := make(chan interface{})
					go func() {
						defer close(ch)
						tt.Transform(context.Background(), input, ch)
					}()

					expect := input + chainLen

					outputs := 0
					for out := range ch {
						if out != expect {
							t.Fatalf("got %d, expecting %d", out, expect)
						}
						outputs++
					}

					if outputs != numOutputs {
						t.Fatalf("unexpected number of outputs\ngot\t%d\n\texpect\t%d", outputs, numOutputs)
					}
				})
			}
		})
	}
}
