package firehose

import (
	"time"
	"errors"
)

type FirehoseConfig struct {
	Region             string        `config:"region"`
	DeliveryStreamName string        `config:"stream_name"`
	BatchSize          int           `config:"batch_size"`
	MaxRetries         int           `config:"max_retries"`
	Timeout            time.Duration `config:"timeout"`
	Backoff            backoff       `config:"backoff"`
}

type backoff struct {
	Init time.Duration
	Max  time.Duration
}

const (
	defaultBatchSize = 50
)

var (
	defaultConfig = FirehoseConfig{
		Timeout:    90 * time.Second,
		MaxRetries: 3,
		Backoff: backoff{
			Init: 1 * time.Second,
			Max:  60 * time.Second,
		},
	}
)

func (c *FirehoseConfig) Validate() error {
	if c.Region == "" {
		return errors.New("region is not defined")
	}

	if c.DeliveryStreamName == "" {
		return errors.New("stream_name is not defined")
	}

	if c.BatchSize > 500 || c.BatchSize < 1 {
		return errors.New("invalid batch size")
	}

	return nil
}
