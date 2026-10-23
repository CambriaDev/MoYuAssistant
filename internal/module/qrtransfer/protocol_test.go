package qrtransfer

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestTransferPayloadRoundTrip(t *testing.T) {
	data := bytes.Repeat([]byte("qr-transfer"), 200)
	transfer, err := NewTransfer(`folder\sample.bin`, data)
	if err != nil {
		t.Fatal(err)
	}
	if transfer.Name != "sample.bin" {
		t.Fatalf("transfer name = %q, want sample.bin", transfer.Name)
	}

	var reconstructed []byte
	for index := 0; index < transfer.Total; index++ {
		payload, err := transfer.Payload(index)
		if err != nil {
			t.Fatal(err)
		}
		frame, chunk, err := DecodePayload(payload)
		if err != nil {
			t.Fatal(err)
		}
		if frame.ID != transfer.ID || frame.Index != index || frame.Total != transfer.Total {
			t.Fatalf("unexpected frame metadata: %+v", frame)
		}
		reconstructed = append(reconstructed, chunk...)
	}
	if !bytes.Equal(reconstructed, data) {
		t.Fatal("decoded frames did not reconstruct the source bytes")
	}
}

func TestEmptyFileHasOneEmptyFrame(t *testing.T) {
	transfer, err := NewTransfer("empty.txt", nil)
	if err != nil {
		t.Fatal(err)
	}
	if transfer.Total != 1 {
		t.Fatalf("frame count = %d, want 1", transfer.Total)
	}
	payload, err := transfer.Payload(0)
	if err != nil {
		t.Fatal(err)
	}
	_, chunk, err := DecodePayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunk) != 0 {
		t.Fatalf("empty file frame contains %d bytes", len(chunk))
	}
}

func TestDecodePayloadRejectsInvalidChunkLength(t *testing.T) {
	transfer, err := NewTransfer("sample.bin", []byte("data"))
	if err != nil {
		t.Fatal(err)
	}
	payload, err := transfer.Payload(0)
	if err != nil {
		t.Fatal(err)
	}
	var frame Frame
	if err := json.Unmarshal([]byte(payload), &frame); err != nil {
		t.Fatal(err)
	}
	frame.Data = ""
	payloadBytes, err := json.Marshal(frame)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := DecodePayload(string(payloadBytes)); err == nil {
		t.Fatal("expected invalid chunk length error")
	}
}

func TestSharedHelloWorldFixture(t *testing.T) {
	const payload = `{"v":1,"id":"0123456789abcdef0123456789abcdef","name":"hello.txt","size":11,"total":1,"index":0,"sha256":"b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9","data":"aGVsbG8gd29ybGQ"}`
	frame, chunk, err := DecodePayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	if frame.Name != "hello.txt" || string(chunk) != "hello world" {
		t.Fatalf("unexpected shared fixture: name=%q data=%q", frame.Name, chunk)
	}
}
