package transform

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

// transforms one struct to another
type structTransformer struct {
	inputType  reflect.Type
	outputType reflect.Type
	// idx[i] = j
	// for expander: input field.i will get mapped to output field.j
	// for collapser: input field.j will get mapped to output field.i
	idx []int
	// this is used to implement the logic explained for idx
	idxReversed bool
}

// InputType is part of the Transformer interface
func (t *structTransformer) InputType() reflect.Type {
	return t.inputType
}

// Transform is part of the Transformer interface
func (t *structTransformer) Transform(ctx context.Context, v interface{}, ch chan<- interface{}) error {
	inValue := reflect.ValueOf(v)
	outValue := reflect.Indirect(reflect.New(t.outputType))
	for inIdx, outIdx := range t.idx {
		if t.idxReversed {
			inIdx, outIdx = outIdx, inIdx // swap
		}
		outValue.Field(outIdx).Set(inValue.Field(inIdx))
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case ch <- outValue.Interface():
		return nil
	}
}

// NewStructExpander creates a transformer from the given type and field names
// An anonymus type is created with fields taken from the given type in the order determined by name
// Transform will convert it from that anonymous type to the given type (expand from a subset to the given output type)
func NewStructExpander(outputType reflect.Type, names []string) (Transformer, error) {
	inputType, idx, err := buildSubtypeAndIdx(outputType, names)
	if err != nil {
		return nil, fmt.Errorf("can't build subtype: %v", err)
	}
	t := structTransformer{
		inputType:   *inputType,
		outputType:  outputType,
		idx:         idx,
		idxReversed: false,
	}
	return &t, nil
}

// NewStructCollapser creates a transformer from the given type and field names
// An anonymus type is created with fields taken from the given type in the order determined by name
// Transform will convert it from the given type to the anonymous type (collapse from the given type to a subset)
func NewStructCollapser(inputType reflect.Type, names []string) (Transformer, error) {
	outputType, idx, err := buildSubtypeAndIdx(inputType, names)
	if err != nil {
		return nil, fmt.Errorf("can't build subtype: %v", err)
	}
	t := structTransformer{
		inputType:   inputType,
		outputType:  *outputType,
		idx:         idx,
		idxReversed: true,
	}
	return &t, nil
}

func buildSubtypeAndIdx(typ reflect.Type, names []string) (*reflect.Type, []int, error) {
	if len(names) == 0 {
		return nil, nil, fmt.Errorf("must provide at least 1 name, got 0")
	}
	if typ.Kind() != reflect.Struct {
		return nil, nil, fmt.Errorf("type needs to be struct, got %s", typ.Kind().String())
	}

	type sfIdx struct {
		sf  reflect.StructField
		idx int
	}
	sfs := map[string]sfIdx{}

	for i, n := 0, typ.NumField(); i < n; i++ {
		sf := typ.Field(i)
		if sf.PkgPath != "" {
			continue // not exported
		}
		name := GetStructFieldName(sf)
		if _, exists := sfs[name]; exists {
			return nil, nil, fmt.Errorf("name %q found multiple times, probably name and tag clash", name)
		}
		sfs[name] = sfIdx{sf, i}
	}

	structFields := make([]reflect.StructField, len(names))
	idx := make([]int, len(names))
	for i, name := range names {
		name = strings.ToLower(name)
		sfIdx, ok := sfs[name]
		if !ok {
			return nil, nil, fmt.Errorf("can't find field with name/tag %q", name)
		}
		structFields[i] = sfIdx.sf
		idx[i] = sfIdx.idx
		delete(sfs, name)
	}

	sType := reflect.StructOf(structFields)

	return &sType, idx, nil
}

// GetStructFieldName returns the name to be used in transformations
// if a field is tagged with 'hive', then that name is used
// name is always lowercase
func GetStructFieldName(sf reflect.StructField) string {
	name := sf.Name
	// check if it has a tag
	if tag, ok := sf.Tag.Lookup("hive"); ok {
		name = tag
	}
	return strings.ToLower(name)
}
