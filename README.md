# conf

Extends https://github.com/caarlos0/env to support arbitrary data providers. This means you can resolve data from anywhere, such as Vault, AWS Secrets manager, S3 or Google Sheets.

# Usage

Implement the config Provider interface

```go
type Provider interface {
	Provide(field reflect.StructField) (string, error)
}
```

Create custom struct tags

```go
type Config struct {
	// github.com/caarlos0/env properties
	Home         string        `env:"HOME"`
	Port         int           `env:"PORT" envDefault:"3000"`
	IsProduction bool          `env:"PRODUCTION"`
	Hosts        []string      `env:"HOSTS" envSeparator:":"`
	Duration     time.Duration `env:"DURATION"`
	TempFolder   string        `env:"TEMP_FOLDER" envDefault:"${HOME}/tmp" envExpand:"true"`
	
	// custom properties
	MySecret     string        `secret:"MY_SECRET,required"`
}
```

Pass config providers to `conf.Parse(...)` like so

```go
var cfg Config
err := env.Parse(&cfg, env.DefaultProvider, myCustomProvider)
```

where `env.DefaultProvider` is the default environment variable parser from `caarlos0/env` and `myCustomProvider` is the custom provider.

# Providers

* [AWS Secrets Manager](https://github.com/steinfletcher/aws-secrets-manager-conf) for resolving secrets from AWS secrets manager.
