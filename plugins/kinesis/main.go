package main

import (
	"github.com/lumigo-io/awsbeats/firehose"
	"github.com/lumigo-io/awsbeats/streams"

	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/plugin"
)

var Bundle = plugin.Bundle(
	outputs.Plugin("firehose", firehose.New),
	outputs.Plugin("streams", streams.New),
)
