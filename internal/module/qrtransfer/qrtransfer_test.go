//go:build module_qr_transfer

package qrtransfer

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestChangingChunkSizeRegeneratesTransferAndMatrix(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	module := &QRTransferModule{state: &appState{}}
	window := app.NewWindow("QR transfer chunk size test")
	defer window.Close()
	content := module.CreateUI(window)
	window.SetContent(content)
	window.Resize(fyne.NewSize(900, 700))
	window.Show()

	fileName := filepath.Join(t.TempDir(), "sample.bin")
	if err := os.WriteFile(fileName, bytes.Repeat([]byte{0xa5}, 1600), 0o600); err != nil {
		t.Fatal(err)
	}
	module.loadFile(fileName)

	module.state.mu.Lock()
	oldID := module.state.transfer.ID
	module.state.mu.Unlock()
	module.setChunkSize(300)

	module.state.mu.Lock()
	transfer := module.state.transfer
	matrix := module.state.chunkMatrix
	module.state.mu.Unlock()
	if transfer.ID == oldID {
		t.Fatal("changing chunk size did not invalidate the old transfer")
	}
	if transfer.ChunkSize != 300 || transfer.Total != 6 {
		t.Fatalf("transfer has chunk size %d and %d frames, want 300 and 6", transfer.ChunkSize, transfer.Total)
	}
	if matrix.total != transfer.Total || matrix.current != -1 {
		t.Fatalf("matrix has %d chunks and current index %d, want %d and -1", matrix.total, matrix.current, transfer.Total)
	}
	matrix.Tapped(&fyne.PointEvent{
		Position: fyne.NewPos(
			matrixPadding+matrixLabelWidth+4*matrixCellPitch+matrixCellSize/2,
			matrixPadding+matrixCellSize/2,
		),
	})
	module.state.mu.Lock()
	current := module.state.current
	module.state.mu.Unlock()
	if current != 4 || matrix.current != 4 {
		t.Fatalf("matrix click selected frame %d and highlighted %d, want 4 for both", current, matrix.current)
	}
	window.Canvas().Refresh(content)
}

func TestManifestAndDataFrameCanBeEncodedAsQR(t *testing.T) {
	for _, size := range []int{300, 500, 700, 900} {
		t.Run(fmt.Sprintf("%d-byte chunks", size), func(t *testing.T) {
			transfer, err := NewTransferWithChunkSize(
				strings.Repeat("x", 255),
				bytes.Repeat([]byte{0xa5}, size),
				size,
			)
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
		})
	}
}

func TestMatrixTapSelectsTheCorrespondingChunk(t *testing.T) {
	selected := -1
	matrix := newChunkMatrix(func(index int) {
		selected = index
	})
	matrix.SetTotal(41)
	matrix.Tapped(&fyne.PointEvent{
		Position: fyne.NewPos(
			matrixPadding+matrixLabelWidth+matrixCellSize/2,
			matrixPadding+2*matrixCellPitch+matrixCellSize/2,
		),
	})
	if selected != 40 {
		t.Fatalf("selected chunk = %d, want 40", selected)
	}
}

func TestMatrixIndexAt(t *testing.T) {
	tests := []struct {
		name     string
		position fyne.Position
		total    int
		want     int
		wantOK   bool
	}{
		{
			name:     "first chunk",
			position: fyne.NewPos(matrixPadding+matrixLabelWidth+matrixCellSize/2, matrixPadding+matrixCellSize/2),
			total:    21,
			want:     0,
			wantOK:   true,
		},
		{
			name:     "twentieth chunk",
			position: fyne.NewPos(matrixPadding+matrixLabelWidth+19*matrixCellPitch+matrixCellSize/2, matrixPadding+matrixCellSize/2),
			total:    21,
			want:     19,
			wantOK:   true,
		},
		{
			name:     "twenty-first chunk",
			position: fyne.NewPos(matrixPadding+matrixLabelWidth+matrixCellSize/2, matrixPadding+matrixCellPitch+matrixCellSize/2),
			total:    21,
			want:     20,
			wantOK:   true,
		},
		{
			name:     "row label is not clickable",
			position: fyne.NewPos(matrixPadding+matrixLabelWidth/2, matrixPadding+matrixCellSize/2),
			total:    21,
			wantOK:   false,
		},
		{
			name:     "empty cell is not clickable",
			position: fyne.NewPos(matrixPadding+matrixLabelWidth+matrixCellSize+1, matrixPadding+matrixCellSize/2),
			total:    21,
			wantOK:   false,
		},
		{
			name:     "unused cell is not clickable",
			position: fyne.NewPos(matrixPadding+matrixLabelWidth+matrixCellSize/2+matrixCellPitch, matrixPadding+matrixCellPitch+matrixCellSize/2),
			total:    21,
			wantOK:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := matrixIndexAt(tc.position, tc.total)
			if got != tc.want || ok != tc.wantOK {
				t.Fatalf("matrixIndexAt(%v, %d) = (%d, %v), want (%d, %v)", tc.position, tc.total, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

func TestFrameIntervalForFPS(t *testing.T) {
	tests := []struct {
		name string
		fps  float64
		want time.Duration
	}{
		{name: "default", fps: 10, want: 100 * time.Millisecond},
		{name: "fractional milliseconds are truncated", fps: 3, want: 333 * time.Millisecond},
		{name: "slow playback", fps: 0.2, want: 5 * time.Second},
		{name: "sub-millisecond interval uses scheduler minimum", fps: 2000, want: time.Millisecond},
		{name: "tiny positive FPS saturates safely", fps: 1e-300, want: time.Duration((1<<63-1)/int64(time.Millisecond)) * time.Millisecond},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := frameIntervalForFPS(tc.fps); got != tc.want {
				t.Fatalf("frameIntervalForFPS(%g) = %s, want %s", tc.fps, got, tc.want)
			}
		})
	}
}
