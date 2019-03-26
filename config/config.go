package config

import (
	"fmt"
	"regexp"

	"gopkg.in/go-playground/validator.v9"
	"gopkg.in/yaml.v2"
)

// Config contains setup of the log shipper.
type Config struct {
	Sources  map[string]Source       `yaml:"sources" validate:"required,min=1,dive,keys,alphadash,endkeys,dive"`
	Targets  map[string]Target       `yaml:"targets" validate:"required,min=1,dive,keys,alphadash,endkeys,dive"`
	Alerting map[string]AlertChannel `yaml:"alerting" validate:"required,min=1,dive,keys,alphadash,endkeys,dive"`
}

// Source is the configuration of a source of log items.
type Source struct {
	Provider string            `yaml:"provider" validate:"required,alphadash"`
	Endpoint string            `yaml:"endpoint,omitempty" validate:"omitempty,url"`
	Targets  []string          `yaml:"targets" validate:"min=1"`
	Config   map[string]string `yaml:"config"`
}

// Target is the config of a target to send logs to.
type Target struct {
	Provider      string                `yaml:"provider" validate:"required,alphadash"`
	Endpoint      string                `yaml:"endpoint,omitempty" validate:"omitempty,url"`
	AlertChannels []map[string][]string `yaml:"alert_channels" validate:"required,dive,dive,keys,alphadash,endkeys,dive,alphadash"`
	RateLimit     []RateLimit           `yaml:"rate_limit" validate:"omitempty,dive"`
}

// RateLimit is the config of rate limiting to apply to a target.
type RateLimit struct {
	Throughput      string          `yaml:"throughput" validate:"required,alphanum"`
	Mode            RateLimitMode   `yaml:"mode" validate:"required,dive"`
	BreachBehaviour BreachBehaviour `yaml:"breach_behaviour" validate:"dive"`
}

// RateLimitMode defines how the throughput is calculated.
type RateLimitMode struct {
	Type                string `yaml:"type" validate:"required,oneof=average"`
	Period              string `yaml:"period" validate:"required,alphanum"`
	Duration            string `yaml:"duration" validate:"required,alphanum"`
	PerSourceIdentifier bool   `yaml:"per_source_identifier"`
}

// BreachBehaviour defines how a breach is treated and actioned.
type BreachBehaviour struct {
	Action     string `yaml:"action" validate:"oneof=fallback discard"`
	Target     string `yaml:"target" validate:"omitempty,alphadash"`
	AlertLevel string `yaml:"alert" validate:"oneof=DEBUG INFO WARN ERROR CRITICAL"`
}

// AlertChannel defines a channel to send alerts to.
type AlertChannel struct {
	Provider string            `yaml:"provider" validate:"alphadash"`
	Suppress string            `yaml:"suppress" validate:"alphanum"`
	Config   map[string]string `yaml:"config"`
}

// Create a new config instance.
func NewConfig() *Config {
	return &Config{}
}

// LoadYAML loads a given yaml string into a config struct.
func (c *Config) LoadYAML(data string) error {
	err := yaml.Unmarshal([]byte(data), c)

	if err != nil {
		return fmt.Errorf("error while parsing config file: %s", err)
	}

	return nil
}

// Validate validates the current config.
func (c *Config) Validate() (bool, error) {
	validate := validator.New()

	_ = validate.RegisterValidation("alphadash", func(fl validator.FieldLevel) bool {
		match := regexp.MustCompile(`(?m)^[a-zA-Z0-9-_]+$`)

		return match.MatchString(fl.Field().String())
	})

	err := validate.Struct(c)

	if err != nil {
		return false, fmt.Errorf(err.Error())
	}

	return true, nil
}
