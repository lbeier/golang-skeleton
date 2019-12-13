package config

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

type Env struct {
	DatabaseDSN      string `envconfig:"DB_DSN" required:"true"`
	JaegerHost       string `envconfig:"JAEGER_HOST"`
	Port             string `envconfig:"PORT" required:"true"`
	MigrationsFolder string `envconfig:"MIGRATIONS_FOLDER" required:"true"`
}

func GetEnv() Env {
	var e Env
	err := envconfig.Process("", &e)
	if err != nil {
		log.Fatal(err.Error())
	}

	return e
}
