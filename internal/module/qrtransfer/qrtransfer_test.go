//go:build module_qr_transfer

package qrtransfer

import (
	"bytes"
	"strings"
	"testing"

	"github.com/skip2/go-qrcode"
)

func TestMaximumFileNameFrameCanBeEncodedAsQR(t *testing.T) {
	transfer, err := NewTransfer(strings.Repeat("x", 255), bytes.Repeat([]byte{0xa5}, chunkSize))
	if err != nil {
		t.Fatal(err)
	}
	payload, err := transfer.Payload(0)
	if err != nil {
		t.Fatal(err)
	}
	png, err := qrcode.Encode(payload, qrcode.Medium, 480)
	if err != nil {
		t.Fatalf("encode QR frame: %v", err)
	}
	if len(png) == 0 {
		t.Fatal("QR encoder returned an empty image")
	}
}
