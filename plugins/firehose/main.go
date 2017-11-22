package firehose

import (
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/plugin"
	"github.com/s12v/awsbeats/firehose"
)

var Bundle = plugin.Bundle(
	outputs.Plugin("firehose", firehose.New),
)
