package storage

import (
	"kula-szpiegula/internal/collector"
	"testing"
	"time"
)

func TestEncodeDecode(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	original := &AggregatedSample{
		Timestamp: now,
		Duration:  time.Second,
		Data: &collector.Sample{
			Timestamp: now,
			CPU: collector.CPUStats{
				Total: collector.CPUCoreStats{
					ID:     "cpu",
					User:   25.5,
					System: 10.2,
					Idle:   64.3,
					Usage:  35.7,
				},
			},
			LoadAvg: collector.LoadAvg{
				Load1:  1.5,
				Load5:  1.2,
				Load15: 0.8,
			},
			Memory: collector.MemoryStats{
				Total: 16 * 1024 * 1024 * 1024,
				Used:  8 * 1024 * 1024 * 1024,
				Free:  4 * 1024 * 1024 * 1024,
			},
			System: collector.SystemStats{
				Hostname: "test-host",
				Entropy:  256,
			},
		},
	}

	encoded, err := encodeSample(original)
	if err != nil {
		t.Fatalf("encodeSample() error: %v", err)
	}
	if len(encoded) == 0 {
		t.Fatal("encodeSample() returned empty data")
	}

	decoded, err := decodeSample(encoded)
	if err != nil {
		t.Fatalf("decodeSample() error: %v", err)
	}

	if !decoded.Timestamp.Equal(original.Timestamp) {
		t.Errorf("Timestamp = %v, want %v", decoded.Timestamp, original.Timestamp)
	}
	if decoded.Duration != original.Duration {
		t.Errorf("Duration = %v, want %v", decoded.Duration, original.Duration)
	}
	if decoded.Data == nil {
		t.Fatal("Decoded Data is nil")
	}
	if decoded.Data.CPU.Total.Usage != original.Data.CPU.Total.Usage {
		t.Errorf("CPU Usage = %f, want %f", decoded.Data.CPU.Total.Usage, original.Data.CPU.Total.Usage)
	}
	if decoded.Data.System.Hostname != "test-host" {
		t.Errorf("Hostname = %q, want \"test-host\"", decoded.Data.System.Hostname)
	}
	if decoded.Data.Memory.Total != original.Data.Memory.Total {
		t.Errorf("Memory Total = %d, want %d", decoded.Data.Memory.Total, original.Data.Memory.Total)
	}
}

func TestDecodeInvalid(t *testing.T) {
	_, err := decodeSample([]byte("not json"))
	if err == nil {
		t.Error("decodeSample() with invalid data should return error")
	}
}

func TestEncodeNilData(t *testing.T) {
	s := &AggregatedSample{
		Timestamp: time.Now(),
		Duration:  time.Second,
		Data:      nil,
	}
	encoded, err := encodeSample(s)
	if err != nil {
		t.Fatalf("encodeSample() with nil Data: %v", err)
	}

	decoded, err := decodeSample(encoded)
	if err != nil {
		t.Fatalf("decodeSample() error: %v", err)
	}
	if decoded.Data != nil {
		t.Error("Decoded Data should be nil")
	}
}
