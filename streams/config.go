package streams

import (
	"errors"
	"fmt"
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
		BatchSizeBytes: maxBatchSizeBytes, // 5MB
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
		return errors.New(fmt.Sprintf("invalid batch size got:%d", c.BatchSize))
	}

	if c.BatchSizeBytes > maxBatchSizeBytes || c.BatchSizeBytes < 1 {
		return errors.New(fmt.Sprintf("invalid batch size bytes got:%d", c.BatchSizeBytes))
	}

	if c.PartitionKeyProvider != "" && c.PartitionKeyProvider != "xid" {
		return errors.New("invalid partition key provider: the only supported provider is `xid`")
	}

	return nil
}
