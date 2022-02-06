package streams

import (
	"errors"
	"time"

	"github.com/jpillora/backoff"
)

type StreamsConfig struct {
	Region               string          `config:"region"`
	DeliveryStreamName   string          `config:"stream_name"`
	PartitionKey         string          `config:"partition_key"`
	PartitionKeyProvider string          `config:"partition_key_provider"`
	BatchSize            int             `config:"batch_size"`
	BatchSizeBytes       int             `config:"batch_size_bytes"`
	MaxRetries           int             `config:"max_retries"`
	Timeout              time.Duration   `config:"timeout"`
	Backoff              backoff.Backoff `config:"backoff"`
}

const (
	defaultBatchSize = 50
	// As per https://docs.aws.amazon.com/sdk-for-go/api/service/kinesis/#Kinesis.PutRecords
	maxBatchSize      = 500
	maxBatchSizeBytes = 5*1024*1024 - 1
)

var (
	defaultConfig = StreamsConfig{
		Timeout:    90 * time.Second,
		MaxRetries: 3,
		Backoff: backoff.Backoff{
			Min:    1 * time.Second,
			Max:    60 * time.Second,
			Jitter: true,
		},
		BatchSizeBytes: 5 * 1000 * 1000, // almost 5MB
	}
)

func (c *StreamsConfig) Validate() error {
	if c.Region == "" {
		return errors.New("region is not defined")
	}

	if c.DeliveryStreamName == "" {
		return errors.New("stream_name is not defined")
	}

	if c.BatchSize > maxBatchSize || c.BatchSize < 1 {
		return errors.New("invalid batch size")
	}

	if c.BatchSizeBytes > maxBatchSizeBytes || c.BatchSizeBytes < 1 {
		return errors.New("invalid batch size bytes")
	}

	if c.PartitionKeyProvider != "" && c.PartitionKeyProvider != "xid" {
		return errors.New("invalid partition key provider: the only supported provider is `xid`")
	}

	return nil
}
