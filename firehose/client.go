package firehose

import (
	"time"
	"github.com/aws/aws-sdk-go/service/firehose"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/codec"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/outputs/codec/json"
)

type client struct {
	firehose           *firehose.Firehose
	deliveryStreamName string
	beatName           string
	encoder            codec.Codec
	timeout            time.Duration
	stats              *outputs.Stats
}

func newClient(sess *session.Session, config *FirehoseConfig, stats *outputs.Stats, beat beat.Info) (*client, error) {
	client := &client{
		firehose:           firehose.New(sess),
		deliveryStreamName: config.DeliveryStreamName,
		beatName:           beat.Beat,
		encoder:            json.New(false, beat.Version),
		timeout:            config.Timeout,
		stats:              stats,
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
	st := client.stats
	events := batch.Events()
	st.NewBatch(len(events))

	records, dropped := client.mapEvents(events)
	res, err := client.sendRecords(records)
	if err != nil {
		logp.Critical("Unable to send batch: %v", err)
		st.Dropped(len(events))
		return err
	}

	processFailedDeliveries(res, batch)
	batch.ACK()
	debugf("Sent %d records", len(events))
	st.Dropped(dropped)
	st.Acked(len(events) - dropped)
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
	serializedEvent, err := client.encoder.Encode(client.beatName, &event.Content)
	if err != nil {
		if !event.Guaranteed() {
			return nil, err
		}

		logp.Critical("Unable to encode event: %v", err)
		return nil, err
	}

	return &firehose.Record{Data: serializedEvent}, nil
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
