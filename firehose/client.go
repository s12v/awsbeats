package firehose

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/codec"
	"github.com/elastic/beats/libbeat/outputs/codec/json"
	"github.com/elastic/beats/libbeat/publisher"
	"time"
)

type client struct {
	firehose           *firehose.Firehose
	deliveryStreamName string
	beatName           string
	encoder            codec.Codec
	timeout            time.Duration
	observer           outputs.Observer
}

func newClient(sess *session.Session, config *FirehoseConfig, observer outputs.Observer, beat beat.Info) (*client, error) {
	client := &client{
		firehose:           firehose.New(sess),
		deliveryStreamName: config.DeliveryStreamName,
		beatName:           beat.Beat,
		encoder:            json.New(false, beat.Version),
		timeout:            config.Timeout,
		observer:           observer,
	}

	return client, nil
}

func (client *client) Close() error {
	return nil
}

func (client *client) Connect() error {
	return nil
}

func (client *client) Publish(batch publisher.Batch) error {
	events := batch.Events()
	observer := client.observer
	observer.NewBatch(len(events))

	records, dropped := client.mapEvents(events)
	res, err := client.sendRecords(records)
	if err != nil {
		logp.Critical("Unable to send batch: %v", err)
		observer.Dropped(len(events))
		return err
	}

	processFailedDeliveries(res, batch)
	batch.ACK()
	logp.Debug("firehose", "Sent %d records", len(events))
	observer.Dropped(dropped)
	observer.Acked(len(events) - dropped)
	return nil
}

func (client *client) mapEvents(events []publisher.Event) ([]*firehose.Record, int) {
	dropped := 0
	records := make([]*firehose.Record, 0, len(events))
	for _, event := range events {
		record, err := client.mapEvent(&event)
		if err != nil {
			dropped++
		} else {
			records = append(records, record)
		}
	}

	return records, dropped
}

func (client *client) mapEvent(event *publisher.Event) (*firehose.Record, error) {
	var buf []byte
	{
		serializedEvent, err := client.encoder.Encode(client.beatName, &event.Content)
		if err != nil {
			if !event.Guaranteed() {
				return nil, err
			}

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

	return &firehose.Record{Data: buf}, nil
}
func (client *client) sendRecords(records []*firehose.Record) (*firehose.PutRecordBatchOutput, error) {
	request := firehose.PutRecordBatchInput{
		DeliveryStreamName: &client.deliveryStreamName,
		Records:            records,
	}
	return client.firehose.PutRecordBatch(&request)
}

func processFailedDeliveries(res *firehose.PutRecordBatchOutput, batch publisher.Batch) {
	if *res.FailedPutCount > 0 {
		events := batch.Events()
		failedEvents := make([]publisher.Event, 0)
		responses := res.RequestResponses
		for i, response := range responses {
			if *response.ErrorCode != "" {
				failedEvents = append(failedEvents, events[i])
			}
		}

		if len(failedEvents) > 0 {
			logp.Warn("Retrying %d events", len(failedEvents))
			batch.RetryEvents(failedEvents)
			return
		}
	}
}
