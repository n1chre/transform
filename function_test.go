package transform

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"
)

func TestFunction(t *testing.T) {

	square := func(x int) int {
		return x * x
	}

	root := func(x int) (int, error) {
		if x < 0 {
			return 0, fmt.Errorf("negative x")
		}
		return int(math.Sqrt(float64(x))), nil
	}

	// produce x+1 after x*100 ms
	inc := func(ctx context.Context, x int) (int, error) {
		out := make(chan int)
		go func() {
			time.Sleep(time.Millisecond * 100 * time.Duration(x))
			out <- x + 1
		}()

		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case y := <-out:
			return y, nil
		}
	}

	for i, c := range []struct {
		f            interface{}
		in           interface{}
		out          interface{}
		createErr    bool
		transformErr bool
	}{
		// need one input, need one output
		{f: func() {}, createErr: true},
		// no error output when using context
		{f: func(context.Context, int) int { return 0 }, createErr: true},
		// only one output arg, no error
		{f: func(int) (int, int) { return 0, 0 }, createErr: true},
		{f: square, in: 2, out: 4},
		{f: root, in: 25, out: 5},
		{f: root, in: -25, transformErr: true},
		{f: inc, in: 3, out: 4},
		{f: inc, in: 100, transformErr: true},
	} {
		t.Run(fmt.Sprintf("case-%d", i+1), func(t *testing.T) {
			tr, err := FromFunction(c.f)
			if err != nil {
				if !c.createErr {
					t.Errorf("should be able to create transformer from %T: %v", c.f, err)
				}
				return
			}
			if c.createErr {
				t.Errorf("shouldn't be able to create a transformer")
				return
			}

			ch := make(chan interface{})
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()

				if err := tr.Transform(ctx, c.in, ch); err != nil {
					ch <- err
				}
			}()

			out := <-ch
			if err, ok := out.(error); ok {
				if !c.transformErr {
					t.Errorf("transform shouldn't error out, got %v", err)
				}
				return
			}
			if c.transformErr {
				t.Errorf("transform should error out, but didn't")
				return
			}

			if !reflect.DeepEqual(out, c.out) {
				t.Errorf("output mismatch:\n\thave: %v\n\twant: %v", out, c.out)
			}
		})
	}
}
