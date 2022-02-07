package streams

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/codec"
	"github.com/elastic/beats/libbeat/outputs/codec/json"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/jpillora/backoff"
)

type client struct {
	streams              kinesisStreamsClient
	streamName           string
	partitionKeyProvider PartitionKeyProvider
	beatName             string
	encoder              codec.Codec
	timeout              time.Duration
	batchSizeBytes       int
	observer             outputs.Observer
	backoff              backoff.Backoff
}

type batch struct {
	allEvents []publisher.Event
	okEvents  []publisher.Event
	records   []*kinesis.PutRecordsRequestEntry
	dropped   int
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
		encoder:              json.New(beat.Version, json.Config{Pretty: false, EscapeHTML: true}),
		timeout:              config.Timeout,
		batchSizeBytes:       config.BatchSizeBytes,
		observer:             observer,
		backoff:              config.Backoff,
	}

	return client, nil
}

func createPartitionKeyProvider(config *StreamsConfig) PartitionKeyProvider {
	if config.PartitionKeyProvider == "xid" {
		return newXidPartitionKeyProvider()
	}
	return newFieldPartitionKeyProvider(config.PartitionKey)
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
	batches := client.mapEvents(events)
	totalFailed := []publisher.Event{}
	for _, batch := range batches {
		failed, err := client.publishBatch(batch)
		if err != nil || len(failed) > 0 {
			totalFailed = append(totalFailed, failed...)
		}
	}
	logp.Debug("kinesis", "received batches: %d for events %d", len(batches), len(events))
	return totalFailed, nil
}

func (client *client) publishBatch(b batch) ([]publisher.Event, error) {
	observer := client.observer

	okEvents, records, dropped, events := b.okEvents, b.records, b.dropped, b.allEvents
	observer.NewBatch(len(b.allEvents))
	if dropped > 0 {
		logp.Debug("kinesis", "sent %d records: %v", len(records), records)
		observer.Dropped(dropped)
		observer.Acked(len(okEvents))
		if len(records) == 0 {
			logp.Debug("kinesis", "No records were mapped")
			return nil, nil
		}
	}
	logp.Debug("kinesis", "mapped to records: %v", records)
	res, err := client.putKinesisRecords(records)
	// TODO: verify if I need to pass the events. Shouldn't I be able to get the events from the batch?
	// maybe just use okEvents?
	failed := collectFailedEvents(res, events)
	if len(failed) == 0 {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case kinesis.ErrCodeLimitExceededException:
			case kinesis.ErrCodeProvisionedThroughputExceededException:
			case kinesis.ErrCodeInternalFailureException:
				logp.Info("putKinesisRecords failed (api level, not per-record failure). Will retry all records. error: %v", err)
				failed = events
			default:
				logp.Warn("putKinesisRecords persistent failure. Will not retry. error: %v", err)
			}
		}
	}
	if err != nil || len(failed) > 0 {
		dur := client.backoff.Duration()
		logp.Info("retrying %d events client.backoff.Duration %s on error: %v", len(failed), dur, err)
		time.Sleep(dur)
	} else {
		client.backoff.Reset()
	}
	return failed, err
}

func (client *client) mapEvents(events []publisher.Event) []batch {
	batches := []batch{}
	dropped := 0
	records := []*kinesis.PutRecordsRequestEntry{}
	okEvents := []publisher.Event{}
	allEvents := []publisher.Event{}

	batchSize := 0

	for i := range events {
		event := events[i]
		size, record, err := client.mapEvent(&event)
		if size >= client.batchSizeBytes {
			logp.Critical("kinesis single record of size %d is bigger than batchSizeBytes %d, sending batch without it! no backoff!", size, client.batchSizeBytes)
			continue
		}
		allEvents = append(allEvents, event)
		if err != nil {
			logp.Debug("kinesis", "failed to map event(%v): %v", event, err)
			dropped++
		} else if batchSize+size >= client.batchSizeBytes {
			batches = append(batches, batch{
				okEvents:  okEvents,
				records:   records,
				dropped:   dropped,
				allEvents: allEvents[:len(allEvents)-1]})
			dropped = 0
			allEvents = []publisher.Event{event}
			batchSize = size
			records = []*kinesis.PutRecordsRequestEntry{record}
			okEvents = []publisher.Event{event}
		} else {
			batchSize += size
			okEvents = append(okEvents, event)
			records = append(records, record)
		}
	}
	batches = append(batches, batch{okEvents: okEvents, records: records, dropped: dropped, allEvents: allEvents})
	return batches
}

func (client *client) mapEvent(event *publisher.Event) (int, *kinesis.PutRecordsRequestEntry, error) {
	var buf []byte
	{
		serializedEvent, err := client.encoder.Encode(client.beatName, &event.Content)
		if err != nil {
			logp.Critical("Unable to encode event: %v", err)
			return 0, nil, err
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
		return 0, nil, fmt.Errorf("failed to get parititon key: %v", err)
	}

	return len(buf), &kinesis.PutRecordsRequestEntry{Data: buf, PartitionKey: aws.String(partitionKey)}, nil
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
				// logp.NewLogger("streams").Warn("skipping failed event with unexpected state: corresponding kinesis record misses error code: ", r)
				continue
			}
			if *r.ErrorCode == "ProvisionedThroughputExceededException" {
				logp.NewLogger("streams").Debug("throughput exceeded. will retry", r)
				failedEvents = append(failedEvents, events[i])
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
