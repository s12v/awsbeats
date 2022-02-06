package streams

import "testing"

func TestValidate(t *testing.T) {
	config := &StreamsConfig{}
	err := config.Validate()
	if err == nil {
		t.Errorf("Expected an error")
	}
}

func TestValidateWithRegion(t *testing.T) {
	config := &StreamsConfig{Region: "eu-central-1"}
	err := config.Validate()
	if err == nil {
		t.Errorf("Expected an error")
	}
}

func TestValidateBatchSizeBytes(t *testing.T) {
	config := &StreamsConfig{Region: "eu-central-1", DeliveryStreamName: "foo", BatchSizeBytes: 5 * 1024 * 1024, BatchSize: 50}
	err := config.Validate()
	if err == nil {
		t.Errorf("Expected an error invalid batch size bytes")
	}
	config = &StreamsConfig{Region: "eu-central-1", DeliveryStreamName: "foo", BatchSizeBytes: 5*1024*1024 - 2, BatchSize: 50}
	err = config.Validate()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}
func TestValidateWithRegionAndStreamNameAndBatchSize(t *testing.T) {
	config := &StreamsConfig{Region: "eu-central-1", DeliveryStreamName: "foo", BatchSize: 50, BatchSizeBytes: 5}
	err := config.Validate()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestValidateWithRegionAndStreamNameAndInvalidBatchSize501(t *testing.T) {
	config := &StreamsConfig{Region: "eu-central-1", DeliveryStreamName: "foo", BatchSize: 501, BatchSizeBytes: 5}
	err := config.Validate()
	if err == nil {
		t.Errorf("Expected an error")
	}
}

func TestValidateWithRegionAndStreamNameAndInvalidBatchSize0(t *testing.T) {
	config := &StreamsConfig{Region: "eu-central-1", DeliveryStreamName: "foo", BatchSize: 0, BatchSizeBytes: 5}
	err := config.Validate()
	if err == nil {
		t.Errorf("Expected an error")
	}
}

func TestValidateWithRegionAndStreamNameAndInvalidPartitionKeyProvider(t *testing.T) {
	config := &StreamsConfig{Region: "eu-central-1", DeliveryStreamName: "foo", PartitionKeyProvider: "uuid", BatchSizeBytes: 5}
	err := config.Validate()
	if err == nil {
		t.Errorf("Expected an error")
	}
}
