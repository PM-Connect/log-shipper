package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	c := NewConfig()

	assert.IsType(t, &Config{}, c)
}

func TestLoadYAML(t *testing.T) {
	var data = `
sources:
  my_source:
    provider: some_provider
    targets:
      - a_target
    config:
      some_value: abc

targets:
  a_target:
    provider: some_provider
    alert_channels:
      - channel1: ~
      - channel2: ["CRITICAL"]
    rate_limit:
      - throughput: 10KB
        mode:
          type: average
          period: 1s
          duration: 10
          per_source_identifier: false
        breach_behaviour:
          action: discard
          alert: CRITICAL
          
alerting:
  channel1:
    provider: some_provider
    suppress: 5m
    config:
      some_key: some-value
  channel2:
    provider: other_provider
    suppress: 10m
    config:
      some_key: some-value
`
	c := NewConfig()

	err := c.LoadYAML(data)

	assert.Nil(t, err)
	assert.IsType(t, &Config{}, c)
	assert.IsType(t, map[string]Source{}, c.Sources)
	assert.IsType(t, map[string]Target{}, c.Targets)
	assert.IsType(t, map[string]AlertChannel{}, c.Alerting)

	assert.IsType(t, Source{}, c.Sources["my_source"])
	assert.Equal(t, "some_provider", c.Sources["my_source"].Provider)
	assert.Equal(t, "", c.Sources["my_source"].Endpoint)
	assert.Equal(t, []string{"a_target"}, c.Sources["my_source"].Targets)
	assert.Equal(t, map[string]string{"some_value": "abc"}, c.Sources["my_source"].Config)

	assert.IsType(t, Target{}, c.Targets["a_target"])
	assert.Equal(t, "some_provider", c.Targets["a_target"].Provider)
	assert.Equal(t, []map[string][]string{{"channel1": []string(nil)}, {"channel2": []string{"CRITICAL"}}}, c.Targets["a_target"].AlertChannels)
	assert.Equal(t, []RateLimit{
		{
			Throughput: "10KB",
			Mode: RateLimitMode{
				Type:     "average",
				Period:   "1s",
				Duration: 10,
			},
			BreachBehaviour: BreachBehaviour{
				Action:     "discard",
				AlertLevel: "CRITICAL",
			},
		},
	}, c.Targets["a_target"].RateLimit)

	assert.IsType(t, AlertChannel{}, c.Alerting["channel1"])
	assert.Equal(t, "some_provider", c.Alerting["channel1"].Provider)
	assert.Equal(t, "5m", c.Alerting["channel1"].Suppress)
	assert.Equal(t, map[string]string{"some_key": "some-value"}, c.Alerting["channel1"].Config)

	assert.IsType(t, AlertChannel{}, c.Alerting["channel2"])
	assert.Equal(t, "other_provider", c.Alerting["channel2"].Provider)
	assert.Equal(t, "10m", c.Alerting["channel2"].Suppress)
	assert.Equal(t, map[string]string{"some_key": "some-value"}, c.Alerting["channel2"].Config)
}

func TestValidateIsSuccessful(t *testing.T) {
	var data = `
sources:
  my_source:
    provider: some_provider
    targets:
      - a_target
    config:
      some_value: abc

targets:
  a_target:
    provider: some_provider
    alert_channels:
      - channel1: ~
      - channel2: ["CRITICAL"]
    rate_limit:
      - throughput: 10KB
        mode:
          type: average
          period: 1s
          duration: 10
        breach_behaviour:
          action: discard
          alert: CRITICAL
          
alerting:
  channel1:
    provider: some_provider
    suppress: 5m
    config:
      some_key: some-value
  channel2:
    provider: other_provider
    suppress: 10m
    config:
      some_key: some-value
`
	c := NewConfig()

	err := c.LoadYAML(data)

	assert.Nil(t, err)
	assert.IsType(t, &Config{}, c)

	valid, err := c.Validate()

	assert.True(t, valid)
	assert.Nil(t, err)
}

func TestValidateIsFailure(t *testing.T) {
	var data = `
sources:
  test:
    provider: lll@
    targets:
      - lll@
    config:
      some_value: abc

targets:
  test:
    provider: some_provider
    alert_channels:
      - channel1: ~
      - channel2: ["CRITICAL"]
    rate_limit:
      - throughput: 10KB
        mode:
          type: average
          period: 1s
          duration: 10
          per_source_identifier: false
        breach_behaviour:
          action: discard
          alert: POTATO
          
alerting:
  channel1:
    provider: some_provider
    suppress: 5m
    config:
      some_key: some-value
  channel2:
    provider: other_provider
    suppress: 10m
    config:
      some_key: some-value
`
	c := NewConfig()

	err := c.LoadYAML(data)

	assert.Nil(t, err)
	assert.IsType(t, &Config{}, c)

	valid, err := c.Validate()

	assert.False(t, valid)
	assert.Error(t, err)
}
