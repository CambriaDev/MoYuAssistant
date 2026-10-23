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

	manifestPayload, err := transfer.ManifestPayload()
	if err != nil {
		t.Fatal(err)
	}
	manifestFrame, err := DecodePayload(manifestPayload)
	if err != nil {
		t.Fatal(err)
	}
	if manifestFrame.Manifest == nil ||
		manifestFrame.Manifest.ID != transfer.ID ||
		manifestFrame.Manifest.Name != transfer.Name ||
		manifestFrame.Manifest.Size != transfer.Size ||
		manifestFrame.Manifest.ChunkSize != chunkSize ||
		manifestFrame.Manifest.Total != transfer.Total {
		t.Fatalf("unexpected manifest: %+v", manifestFrame.Manifest)
	}

	var reconstructed []byte
	for index := 0; index < transfer.Total; index++ {
		payload, err := transfer.Payload(index)
		if err != nil {
			t.Fatal(err)
		}
		frame, err := DecodePayload(payload)
		if err != nil {
			t.Fatal(err)
		}
		if frame.Data == nil || frame.Data.ID != transfer.ID || frame.Data.Index != index {
			t.Fatalf("unexpected data frame: %+v", frame.Data)
		}
		reconstructed = append(reconstructed, frame.Chunk...)
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
	frame, err := DecodePayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(frame.Chunk) != 0 {
		t.Fatalf("empty file frame contains %d bytes", len(frame.Chunk))
	}
}

func TestDecodePayloadRejectsInvalidDataFrame(t *testing.T) {
	transfer, err := NewTransfer("sample.bin", []byte("data"))
	if err != nil {
		t.Fatal(err)
	}
	payload, err := transfer.Payload(0)
	if err != nil {
		t.Fatal(err)
	}
	var frame DataFrame
	if err := json.Unmarshal([]byte(payload), &frame); err != nil {
		t.Fatal(err)
	}
	frame.Index = -1
	payloadBytes, err := json.Marshal(frame)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodePayload(string(payloadBytes)); err == nil {
		t.Fatal("expected invalid frame index error")
	}
}

func TestRejectsManifestWithInconsistentFrameCount(t *testing.T) {
	const payload = `{"v":2,"type":"manifest","id":"0123456789abcdef0123456789abcdef","name":"hello.txt","size":11,"chunkSize":700,"total":2,"sha256":"b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"}`
	if _, err := DecodePayload(payload); err == nil {
		t.Fatal("expected inconsistent manifest to be rejected")
	}
}

func TestSharedHelloWorldFixture(t *testing.T) {
	const manifestPayload = `{"v":2,"type":"manifest","id":"0123456789abcdef0123456789abcdef","name":"hello.txt","size":11,"chunkSize":700,"total":1,"sha256":"b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"}`
	const dataPayload = `{"v":2,"type":"data","id":"0123456789abcdef0123456789abcdef","index":0,"data":"aGVsbG8gd29ybGQ"}`

	manifest, err := DecodePayload(manifestPayload)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Manifest == nil || manifest.Manifest.Name != "hello.txt" {
		t.Fatalf("unexpected shared manifest: %+v", manifest.Manifest)
	}
	frame, err := DecodePayload(dataPayload)
	if err != nil {
		t.Fatal(err)
	}
	if frame.Data == nil || frame.Data.ID != manifest.Manifest.ID || string(frame.Chunk) != "hello world" {
		t.Fatalf("unexpected shared data frame: %+v data=%q", frame.Data, frame.Chunk)
	}
}

func TestTenMiBTransferFrameCount(t *testing.T) {
	data := make([]byte, 10*1024*1024)
	transfer, err := NewTransfer("ten-mebibytes.bin", data)
	if err != nil {
		t.Fatal(err)
	}
	expected := (len(data)-1)/chunkSize + 1
	if transfer.Total != expected {
		t.Fatalf("frame count = %d, want %d", transfer.Total, expected)
	}
}
