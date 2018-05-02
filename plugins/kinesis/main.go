package main

import (
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/plugin"
	"github.com/s12v/awsbeats/firehose"
	"github.com/s12v/awsbeats/streams"
)

var Bundle = plugin.Bundle(
	outputs.Plugin("firehose", firehose.New),
	outputs.Plugin("streams", streams.New),
)
