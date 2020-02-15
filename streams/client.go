package streams

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/codec"
	"github.com/elastic/beats/libbeat/outputs/codec/json"
	"github.com/elastic/beats/libbeat/publisher"
	"time"
)

type client struct {
	streams              kinesisStreamsClient
	streamName           string
	partitionKeyProvider PartitionKeyProvider
	beatName             string
	encoder              codec.Codec
	timeout              time.Duration
	observer             outputs.Observer
}

type kinesisStreamsClient interface {
	PutRecords(input *kinesis.PutRecordsInput) (*kinesis.PutRecordsOutput, error)
}

func newClient(sess *session.Session, config *StreamsConfig, observer outputs.Observer, beat beat.Info) (*client, error) {
	partitionKeyProvider := createPartitionKeyProvider(config)
	client := &client{
		streams:              kinesis.New(sess),
		streamName:           config.DeliveryStreamName,
		partitionKeyProvider: partitionKeyProvider,
		beatName:             beat.Beat,
		encoder: json.New(beat.Version, json.Config{
			Pretty:     false,
			EscapeHTML: false,
		}),
		timeout:  config.Timeout,
		observer: observer,
	}

	return client, nil
}

func createPartitionKeyProvider(config *StreamsConfig) PartitionKeyProvider {
	if config.PartitionKeyProvider == "xid" {
		return newXidPartitionKeyProvider()
	} else {
		return newFieldPartitionKeyProvider(config.PartitionKey)
	}
}

func (client client) String() string {
	return "streams"
}

func (client *client) Close() error {
	return nil
}

func (client *client) Connect() error {
	return nil
}

func (client *client) Publish(batch publisher.Batch) error {
	events := batch.Events()
	rest, _ := client.publishEvents(events)
	if len(rest) == 0 {
		// We have to ACK only when all the submission succeeded
		// Ref: https://github.com/elastic/beats/blob/c4af03c51373c1de7daaca660f5d21b3f602771c/libbeat/outputs/elasticsearch/client.go#L232
		batch.ACK()
	} else {
		// Mark the failed events to retry
		// Ref: https://github.com/elastic/beats/blob/c4af03c51373c1de7daaca660f5d21b3f602771c/libbeat/outputs/elasticsearch/client.go#L234
		batch.RetryEvents(rest)
	}
	// This shouldn't be an error object according to other official beats' implementations
	// Ref: https://github.com/elastic/beats/blob/c4af03c51373c1de7daaca660f5d21b3f602771c/libbeat/outputs/kafka/client.go#L119
	return nil
}

func (client *client) publishEvents(events []publisher.Event) ([]publisher.Event, error) {
	observer := client.observer
	observer.NewBatch(len(events))

	logp.Debug("kinesis", "received events: %v", events)
	okEvents, records, dropped := client.mapEvents(events)
	if dropped > 0 {
		logp.Debug("kinesis", "sent %d records: %v", len(records), records)
		observer.Dropped(dropped)
		observer.Acked(len(okEvents))
	}
	logp.Debug("kinesis", "mapped to records: %v", records)
	res, err := client.putKinesisRecords(records)
	failed := collectFailedEvents(res, events)
	if err != nil && len(failed) == 0 {
		failed = events
	}
	if len(failed) > 0 {
		logp.Info("retrying %d events on error: %v", len(failed), err)
	}
	return failed, err
}

func (client *client) mapEvents(events []publisher.Event) ([]publisher.Event, []*kinesis.PutRecordsRequestEntry, int) {
	dropped := 0
	records := make([]*kinesis.PutRecordsRequestEntry, 0, len(events))
	okEvents := make([]publisher.Event, 0, len(events))
	for i := range events {
		event := events[i]
		record, err := client.mapEvent(&event)
		if err != nil {
			logp.Debug("kinesis", "failed to map event(%v): %v", event, err)
			dropped++
		} else {
			okEvents = append(okEvents, event)
			records = append(records, record)
		}
	}
	return okEvents, records, dropped
}

func (client *client) mapEvent(event *publisher.Event) (*kinesis.PutRecordsRequestEntry, error) {
	var buf []byte
	{
		serializedEvent, err := client.encoder.Encode(client.beatName, &event.Content)
		if err != nil {
			logp.Critical("Unable to encode event: %v", err)
			return nil, err
		}
		// See https://github.com/elastic/beats/blob/5a6630a8bc9b9caf312978f57d1d9193bdab1ac7/libbeat/outputs/kafka/client.go#L163-L164
		// You need to copy the byte data like this. Otherwise you see strange issues like all the records sent in a same batch has the same Data.
		buf = make([]byte, len(serializedEvent)+1)
		copy(buf, serializedEvent)
		// Firehose doesn't automatically add trailing new-line on after each record.
		// This ends up a stream->firehose->s3 pipeline to produce useless s3 objects.
		// No ndjson, but a sequence of json objects without separators...
		// Fix it just adding a new-line.
		//
		// See https://stackoverflow.com/questions/43010117/writing-properly-formatted-json-to-s3-to-load-in-athena-redshift
		buf[len(buf)-1] = byte('\n')
	}

	partitionKey, err := client.partitionKeyProvider.PartitionKeyFor(event)
	if err != nil {
		return nil, fmt.Errorf("failed to get parititon key: %v", err)
	}

	return &kinesis.PutRecordsRequestEntry{Data: buf, PartitionKey: aws.String(partitionKey)}, nil
}
func (client *client) putKinesisRecords(records []*kinesis.PutRecordsRequestEntry) (*kinesis.PutRecordsOutput, error) {
	request := kinesis.PutRecordsInput{
		StreamName: &client.streamName,
		Records:    records,
	}
	res, err := client.streams.PutRecords(&request)
	if err != nil {
		return res, fmt.Errorf("failed to put records: %v", err)
	}
	return res, nil
}

func collectFailedEvents(res *kinesis.PutRecordsOutput, events []publisher.Event) []publisher.Event {
	if res.FailedRecordCount != nil && *res.FailedRecordCount > 0 {
		failedEvents := make([]publisher.Event, 0)
		records := res.Records
		for i, r := range records {
			if r == nil {
				// See https://github.com/s12v/awsbeats/issues/27 for more info
				logp.NewLogger("streams").Warn("no record returned from kinesis for event: ", events[i])
				continue
			}
			if r.ErrorCode == nil {
				logp.NewLogger("streams").Warn("skipping failed event with unexpected state: corresponding kinesis record misses error code: ", r)
				continue
			}
			if *r.ErrorCode != "" {
				failedEvents = append(failedEvents, events[i])
			}
		}
		logp.Warn("Retrying %d events", len(failedEvents))
		return failedEvents
	}
	return []publisher.Event{}
}
