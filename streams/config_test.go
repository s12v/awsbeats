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

func TestValidateWithRegionAndStreamNameAndBatchSize(t *testing.T) {
	config := &StreamsConfig{Region: "eu-central-1", DeliveryStreamName: "foo", BatchSize: 50}
	err := config.Validate()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestValidateWithRegionAndStreamNameAndInvalidBatchSize501(t *testing.T) {
	config := &StreamsConfig{Region: "eu-central-1", DeliveryStreamName: "foo", BatchSize: 501}
	err := config.Validate()
	if err == nil {
		t.Errorf("Expected an error")
	}
}

func TestValidateWithRegionAndStreamNameAndInvalidBatchSize0(t *testing.T) {
	config := &StreamsConfig{Region: "eu-central-1", DeliveryStreamName: "foo", BatchSize: 0}
	err := config.Validate()
	if err == nil {
		t.Errorf("Expected an error")
	}
}
