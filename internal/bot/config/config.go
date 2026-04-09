package config

import (
	"errors"
	"os"
)

// Config stores the bot's configuration.
type Config struct {
	DBAddress   string
	BotToken    string
	GigaToken   string
	SaluteToken string
}

func (c *Config) loadFromEnvs() error {
	if v := os.Getenv("DATABASE_DSN"); v != "" {
		c.DBAddress = v
	} else {
		return errors.New("env variable `DATABASE_DSN` is not set")
	}
	if v := os.Getenv("BOT_TOKEN"); v != "" {
		c.BotToken = v
	} else {
		return errors.New("env variable `BOT_TOKEN` is not set")
	}
	if v := os.Getenv("GIGACHAT_TOKEN"); v != "" {
		c.GigaToken = v
	} else {
		return errors.New("env variable `GIGACHAT_TOKEN` is not set")
	}
	if v := os.Getenv("SALUTESPEECH_TOKEN"); v != "" {
		c.SaluteToken = v
	} else {
		return errors.New("env variable `SALUTESPEECH_TOKEN` is not set")
	}
	return nil
}

// New returns a new config.
func New() (Config, error) {
	c := Config{}
	err := c.loadFromEnvs()
	if err != nil {
		return Config{}, err
	}
	return c, nil
}
