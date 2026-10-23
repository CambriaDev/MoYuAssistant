//go:build module_qr_transfer

package qrtransfer

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"github.com/skip2/go-qrcode"
)

func TestQRCodeModuleUIRendersBeforeFileSelection(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	module := &QRTransferModule{state: &appState{}}
	window := app.NewWindow("QR transfer test")
	defer window.Close()
	content := module.CreateUI(window)
	window.SetContent(content)
	window.Resize(fyne.NewSize(900, 700))
	window.Show()

	fileName := filepath.Join(t.TempDir(), "sample.bin")
	if err := os.WriteFile(fileName, []byte("test payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	module.loadFile(fileName)
	window.Canvas().Refresh(content)
}

func TestManifestAndDataFrameCanBeEncodedAsQR(t *testing.T) {
	transfer, err := NewTransfer(strings.Repeat("x", 255), bytes.Repeat([]byte{0xa5}, chunkSize))
	if err != nil {
		t.Fatal(err)
	}
	payloads := make([]string, 0, 2)
	manifestPayload, err := transfer.ManifestPayload()
	if err != nil {
		t.Fatal(err)
	}
	payloads = append(payloads, manifestPayload)
	dataPayload, err := transfer.Payload(0)
	if err != nil {
		t.Fatal(err)
	}
	payloads = append(payloads, dataPayload)
	for _, payload := range payloads {
		png, err := qrcode.Encode(payload, qrcode.Medium, 480)
		if err != nil {
			t.Fatalf("encode QR frame: %v", err)
		}
		if len(png) == 0 {
			t.Fatal("QR encoder returned an empty image")
		}
	}
}
