package conf

import (
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Code adapted from https://github.com/caarlos0/env

// nolint: gochecknoglobals
var (
	// ErrNotAStructPtr is returned if you pass something that is not a pointer to a
	// Struct to Parse
	ErrNotAStructPtr = errors.New("env: expected a pointer to a Struct")

	defaultBuiltInParsers = map[reflect.Kind]ParserFunc{
		reflect.Bool: func(v string) (interface{}, error) {
			return strconv.ParseBool(v)
		},
		reflect.String: func(v string) (interface{}, error) {
			return v, nil
		},
		reflect.Int: func(v string) (interface{}, error) {
			i, err := strconv.ParseInt(v, 10, 32)
			return int(i), err
		},
		reflect.Int16: func(v string) (interface{}, error) {
			i, err := strconv.ParseInt(v, 10, 16)
			return int16(i), err
		},
		reflect.Int32: func(v string) (interface{}, error) {
			i, err := strconv.ParseInt(v, 10, 32)
			return int32(i), err
		},
		reflect.Int64: func(v string) (interface{}, error) {
			return strconv.ParseInt(v, 10, 64)
		},
		reflect.Int8: func(v string) (interface{}, error) {
			i, err := strconv.ParseInt(v, 10, 8)
			return int8(i), err
		},
		reflect.Uint: func(v string) (interface{}, error) {
			i, err := strconv.ParseUint(v, 10, 32)
			return uint(i), err
		},
		reflect.Uint16: func(v string) (interface{}, error) {
			i, err := strconv.ParseUint(v, 10, 16)
			return uint16(i), err
		},
		reflect.Uint32: func(v string) (interface{}, error) {
			i, err := strconv.ParseUint(v, 10, 32)
			return uint32(i), err
		},
		reflect.Uint64: func(v string) (interface{}, error) {
			i, err := strconv.ParseUint(v, 10, 64)
			return i, err
		},
		reflect.Uint8: func(v string) (interface{}, error) {
			i, err := strconv.ParseUint(v, 10, 8)
			return uint8(i), err
		},
		reflect.Float64: func(v string) (interface{}, error) {
			return strconv.ParseFloat(v, 64)
		},
		reflect.Float32: func(v string) (interface{}, error) {
			f, err := strconv.ParseFloat(v, 32)
			return float32(f), err
		},
	}

	defaultTypeParsers = map[reflect.Type]ParserFunc{
		reflect.TypeOf(url.URL{}): func(v string) (interface{}, error) {
			u, err := url.Parse(v)
			if err != nil {
				return nil, fmt.Errorf("unable parse URL: %v", err)
			}
			return *u, nil
		},
		reflect.TypeOf(time.Nanosecond): func(v string) (interface{}, error) {
			s, err := time.ParseDuration(v)
			if err != nil {
				return nil, fmt.Errorf("unable to parser duration: %v", err)
			}
			return s, err
		},
	}
)

// ParserFunc defines the signature of a function that can be used within `CustomParsers`
type ParserFunc func(v string) (interface{}, error)

// Parse parses a struct containing `env` tags and loads its values from
// environment variables.
func Parse(v interface{}, providers ...Provider) error {
	for _, provider := range providers {
		if err := ParseWithFuncs(v, map[reflect.Type]ParserFunc{}, provider); err != nil {
			return err
		}
	}
	return nil
}

// MustParse is a helper function to ensure the config is valid and there was no  error when calling the Parse function.
func MustParse(v interface{}, providers ...Provider) {
	err := Parse(v, providers...)
	if err != nil {
		panic(err)
	}
}

// ParseWithFuncs is the same as `Parse` except it also allows the user to pass
// in custom parsers.
func ParseWithFuncs(v interface{}, funcMap map[reflect.Type]ParserFunc, provider Provider) error {
	ptrRef := reflect.ValueOf(v)
	if ptrRef.Kind() != reflect.Ptr {
		return ErrNotAStructPtr
	}
	ref := ptrRef.Elem()
	if ref.Kind() != reflect.Struct {
		return ErrNotAStructPtr
	}
	var parsers = defaultTypeParsers
	for k, v := range funcMap {
		parsers[k] = v
	}
	return doParse(ref, parsers, provider)
}

func doParse(ref reflect.Value, funcMap map[reflect.Type]ParserFunc, provider Provider) error {
	var refType = ref.Type()

	for i := 0; i < refType.NumField(); i++ {
		refField := ref.Field(i)
		if !refField.CanSet() {
			continue
		}
		if reflect.Ptr == refField.Kind() && !refField.IsNil() {
			err := ParseWithFuncs(refField.Interface(), funcMap, provider)
			if err != nil {
				return err
			}
			continue
		}
		if reflect.Struct == refField.Kind() && refField.CanAddr() && refField.Type().Name() == "" {
			err := Parse(refField.Addr().Interface(), provider)
			if nil != err {
				return err
			}
			continue
		}
		refTypeField := refType.Field(i)
		value, err := provider.Provide(refTypeField)
		if err != nil {
			return err
		}
		if value == "" {
			if reflect.Struct == refField.Kind() {
				if err := doParse(refField, funcMap, provider); err != nil {
					return err
				}
			}
			continue
		}
		if err := set(refField, refTypeField, value, funcMap); err != nil {
			return err
		}
	}
	return nil
}

func set(field reflect.Value, sf reflect.StructField, value string, funcMap map[reflect.Type]ParserFunc) error {
	if field.Kind() == reflect.Slice {
		return handleSlice(field, value, sf, funcMap)
	}

	var tm = asTextUnmarshaler(field)
	valBytes := []byte(value)
	if tm != nil {
		var err = tm.UnmarshalText(valBytes)
		return newParseError(sf, err)
	}

	var typee = sf.Type
	var fieldee = field
	if typee.Kind() == reflect.Ptr {
		typee = typee.Elem()
		fieldee = field.Elem()
	}

	parserFunc, ok := funcMap[typee]
	if ok {
		val, err := parserFunc(value)
		if err != nil {
			return newParseError(sf, err)
		}

		fieldee.Set(reflect.ValueOf(val))
		return nil
	}

	parserFunc, ok = defaultBuiltInParsers[typee.Kind()]
	if ok {
		val, err := parserFunc(value)
		if err != nil {
			return newParseError(sf, err)
		}

		fieldee.Set(reflect.ValueOf(val).Convert(typee))
		return nil
	}

	if typee.Kind() == reflect.Struct {
		if json.Valid(valBytes) && isJSONObj(valBytes) {
			i := reflect.New(typee)
			newP := i.Interface()
			err := json.Unmarshal(valBytes, newP)
			if err != nil {
				return err
			}
			fieldee.Set(reflect.ValueOf(newP).Elem())
			return nil
		}
	}

	return newNoParserError(sf)
}

func isJSONObj(s []byte) bool {
	var js map[string]interface{}
	return json.Unmarshal(s, &js) == nil
}

func handleSlice(field reflect.Value, value string, sf reflect.StructField, funcMap map[reflect.Type]ParserFunc) error {
	var separator = sf.Tag.Get("envSeparator")
	if separator == "" {
		separator = ","
	}
	var parts = strings.Split(value, separator)

	var typee = sf.Type.Elem()
	if typee.Kind() == reflect.Ptr {
		typee = typee.Elem()
	}

	if _, ok := reflect.New(typee).Interface().(encoding.TextUnmarshaler); ok {
		return parseTextUnmarshalers(field, parts, sf)
	}

	parserFunc, ok := funcMap[typee]
	if !ok {
		parserFunc, ok = defaultBuiltInParsers[typee.Kind()]
		if !ok {
			return newNoParserError(sf)
		}
	}

	var result = reflect.MakeSlice(sf.Type, 0, len(parts))
	for _, part := range parts {
		r, err := parserFunc(part)
		if err != nil {
			return newParseError(sf, err)
		}
		var v = reflect.ValueOf(r).Convert(typee)
		if sf.Type.Elem().Kind() == reflect.Ptr {
			v = reflect.New(typee)
			v.Elem().Set(reflect.ValueOf(r).Convert(typee))
		}
		result = reflect.Append(result, v)
	}
	field.Set(result)
	return nil
}

func asTextUnmarshaler(field reflect.Value) encoding.TextUnmarshaler {
	if reflect.Ptr == field.Kind() {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
	} else if field.CanAddr() {
		field = field.Addr()
	}

	tm, ok := field.Interface().(encoding.TextUnmarshaler)
	if !ok {
		return nil
	}
	return tm
}

func parseTextUnmarshalers(field reflect.Value, data []string, sf reflect.StructField) error {
	s := len(data)
	elemType := field.Type().Elem()
	slice := reflect.MakeSlice(reflect.SliceOf(elemType), s, s)
	for i, v := range data {
		sv := slice.Index(i)
		kind := sv.Kind()
		if kind == reflect.Ptr {
			sv = reflect.New(elemType.Elem())
		} else {
			sv = sv.Addr()
		}
		tm := sv.Interface().(encoding.TextUnmarshaler)
		if err := tm.UnmarshalText([]byte(v)); err != nil {
			return newParseError(sf, err)
		}
		if kind == reflect.Ptr {
			slice.Index(i).Set(sv)
		}
	}

	field.Set(slice)

	return nil
}

func newParseError(sf reflect.StructField, err error) error {
	if err == nil {
		return nil
	}
	return parseError{
		sf:  sf,
		err: err,
	}
}

type parseError struct {
	sf  reflect.StructField
	err error
}

func (e parseError) Error() string {
	return fmt.Sprintf(`env: parse error on field "%s" of type "%s": %v`, e.sf.Name, e.sf.Type, e.err)
}

func newNoParserError(sf reflect.StructField) error {
	return fmt.Errorf(`env: no parser found for field "%s" of type "%s"`, sf.Name, sf.Type)
}
