package transform

import (
	"context"
	"reflect"
	"testing"
)

type handlerTest struct{}

func (ht handlerTest) InputType() reflect.Type {
	return reflect.TypeOf(ht) // anything
}

func (ht handlerTest) Transform(_ context.Context, _ interface{}, _ chan<- interface{}) error {
	return ht // error
}

func (handlerTest) Error() string {
	return "test"
}

func TestErrorHandler(t *testing.T) {
	testErrorHandler(t, false)
	testErrorHandler(t, true)
}

func testErrorHandler(t *testing.T, shouldError bool) {
	ht := handlerTest{}
	called := false

	tr := WithErrorHandler(ht, func(e error) error {
		called = true
		if e != ht {
			t.Fatal("expecting the error to be ht, got something else")
		}

		if shouldError {
			return e
		}
		return nil
	})

	if err := tr.Transform(nil, nil, nil); err != nil {
		if !shouldError {
			t.Fatal("error was returned, but shouldn't have been")
		}
	} else if shouldError {
		t.Fatal("no error was returned")
	}

	if !called {
		t.Fatal("error handler wasn't called")
	}
}
