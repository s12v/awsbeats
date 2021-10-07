package firehose

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs"
)

var (
	newClientFunc = newClient
	awsNewSession = session.NewSession
)

func New(
	_ outputs.IndexManager,
	beat beat.Info,
	stats outputs.Observer,
	cfg *common.Config,
) (outputs.Group, error) {
	if !cfg.HasField("batch_size") {
		cfg.SetInt("batch_size", -1, defaultBatchSize)
	}

	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return outputs.Fail(err)
	}

	var client outputs.NetworkClient
	sess := session.Must(awsNewSession(&aws.Config{Region: aws.String(config.Region)}))
	client, err := newClientFunc(sess, &config, stats, beat)
	if err != nil {
		return outputs.Fail(err)
	}

	client = outputs.WithBackoff(client, config.Backoff.Init, config.Backoff.Max)
	return outputs.Success(config.BatchSize, config.MaxRetries, client)
}
