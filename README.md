# conf

Extends https://github.com/caarlos0/env to support arbitrary data providers, e.g. secrets manager tools like Vault or AWS Secrets Manager.

# Usage

Implement the config Provider interface

```go
type Provider interface {
	GetValue(field reflect.StructField) (string, error)
}
```

Create custom struct tags

```go
type Config struct {
	MyEnvVar string `env:"MY_ENV"`
	MySecret string `secret:"MY_SECRET"`
}
```

Pass config providers to `conf.Parse(...)` like so

```go
var cfg Config
err := env.Parse(&cfg, env.DefaultProvider, myCustomProvider)
```

where `env.DefaultProvider` is the default environment variable parser from `caarlos0/env` and `myCustomProvider` is the custom provider.
