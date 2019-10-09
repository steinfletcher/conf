package conf

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

type Provider interface {
	Provide(field reflect.StructField) (string, error)
}

var EnvProvider = envProvider{tag: "env"}

var SecretEnvProvider = envProvider{tag: "secret"}

type envProvider struct {
	tag string
}

func (o envProvider) Provide(field reflect.StructField) (string, error) {
	var val string
	var err error

	key, opts := parseKeyForOption(field.Tag.Get(o.tag))

	defaultValue := field.Tag.Get("envDefault")
	val = getOr(key, defaultValue)

	expandVar := field.Tag.Get("envExpand")
	if strings.ToLower(expandVar) == "true" {
		val = os.ExpandEnv(val)
	}

	if len(opts) > 0 {
		for _, opt := range opts {
			// The only option supported is "required".
			switch opt {
			case "":
				break
			case "required":
				val, err = getRequired(key)
			default:
				err = fmt.Errorf("env: tag option %q not supported", opt)
			}
		}
	}

	return val, err
}

func getOr(key, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if ok {
		return value
	}
	return defaultValue
}

func parseKeyForOption(key string) (string, []string) {
	opts := strings.Split(key, ",")
	return opts[0], opts[1:]
}

func getRequired(key string) (string, error) {
	if value, ok := os.LookupEnv(key); ok {
		return value, nil
	}
	return "", fmt.Errorf(`env: required environment variable %q is not set`, key)
}
