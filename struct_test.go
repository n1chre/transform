package transform

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestStructExpander(t *testing.T) {
	type foo struct {
		I int
		B bool
		S struct {
			X float32
			Y float64
		}
	}

	cases := []struct {
		in  map[string]interface{}
		out interface{}
		err bool
	}{
		{
			in:  map[string]interface{}{},
			out: foo{},
			err: true,
		},
		{
			in: map[string]interface{}{
				"doesnt exist": 123,
			},
			out: foo{},
			err: true,
		},
		{
			in: map[string]interface{}{
				"I": 1,
			},
			out: foo{
				I: 1,
			},
		},
		{
			in: map[string]interface{}{
				"S": struct {
					X float32
					Y float64
				}{2, 3},
			},
			out: foo{
				S: struct {
					X float32
					Y float64
				}{2, 3},
			},
		},
		{
			in: map[string]interface{}{
				"S": struct {
					X float32
					Y float64
				}{2, 3},
				"I": 1,
			},
			out: foo{
				I: 1,
				S: struct {
					X float32
					Y float64
				}{2, 3},
			},
		},
	}

	for i, c := range cases {
		c := c
		t.Run(fmt.Sprintf("case-%d", i+1), func(t *testing.T) {
			var names []string
			for name := range c.in {
				names = append(names, name)
			}

			st, err := NewStructExpander(reflect.TypeOf(c.out), names)
			if err != nil {
				if !c.err {
					t.Fatalf("can't create transformer: %v", err)
				}
				return
			}
			if c.err {
				t.Fatalf("shouldn't be able to build a transformer")
			}

			in := reflect.Indirect(reflect.New(st.InputType()))
			for name, v := range c.in {
				in.FieldByNameFunc(testMatchFunc(name)).Set(reflect.ValueOf(v))
			}

			ch := make(chan interface{})
			go func() {
				st.Transform(context.Background(), in.Interface(), ch)
			}()
			out := <-ch

			have := reflect.ValueOf(out).Interface()
			if !reflect.DeepEqual(c.out, have) {
				t.Fatalf("transform %v\n\thave:\t%v\n\twant:\t%v", c.in, have, c.out)
			}
		})
	}
}

func TestStructCollapser(t *testing.T) {
	type foo struct {
		I int
		B bool
		S struct {
			X float32
			Y float64
		}
	}

	f := foo{1, true, struct {
		X float32
		Y float64
	}{2, 3}}

	cases := []struct {
		in    interface{}
		names []string
		vals  map[string]interface{}
		err   bool
	}{
		{
			in:    f,
			names: []string{"dont exist"},
			err:   true, // name doesn't exist
		},
		{
			in:    f,
			names: []string{"i"},
			vals: map[string]interface{}{
				"i": 1,
			},
		},
		{
			in:    f,
			names: []string{"i", "I"},
			err:   true, // foo.I used 2 times
		},
		{
			in:    f,
			names: []string{"s", "B"},
			vals: map[string]interface{}{
				"S": struct {
					X float32
					Y float64
				}{2, 3},
				"b": true,
			},
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("case-%d", i+1), func(t *testing.T) {
			st, err := NewStructCollapser(reflect.TypeOf(c.in), c.names)
			if err != nil {
				if !c.err {
					t.Fatalf("can't create transformer: %v", err)
				}
				return
			}
			if c.err {
				t.Fatalf("shouldn't be able to build a transformer")
			}

			ch := make(chan interface{})
			go func() {
				st.Transform(context.Background(), c.in, ch)
			}()
			out := <-ch

			vOut := reflect.ValueOf(out)
			for name, value := range c.vals {
				sf := vOut.FieldByNameFunc(testMatchFunc(name))
				if reflect.DeepEqual(sf, reflect.Value{}) {
					t.Fatalf("can't find field in output with name %q", name)
				}
				if !reflect.DeepEqual(sf.Interface(), value) {
					t.Fatalf("output field %q mismatch\n\thave:\t%v\n\twant:\t%v", name, sf.Interface(), value)
				}
			}
		})
	}
}

func testMatchFunc(name string) func(string) bool {
	nameLower := strings.ToLower(name)
	return func(fieldName string) bool {
		first := fieldName[0]
		if first < 'A' || first > 'Z' {
			// not exported
			return false
		}
		// check if it matches the name
		return strings.ToLower(fieldName) == nameLower
	}
}

func TestGetStructFieldName(t *testing.T) {
	type foo struct {
		Name  int
		Name2 int `hive:"name"`
	}

	typ := reflect.TypeOf(foo{})
	if name := GetStructFieldName(typ.Field(0)); name != "name" {
		t.Fatalf("expecting %q, got %q", "name", name)
	}
	if name := GetStructFieldName(typ.Field(1)); name != "name" {
		t.Fatalf("expecting %q, got %q", "name", name)
	}
}
