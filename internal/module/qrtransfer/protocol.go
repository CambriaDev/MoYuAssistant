package qrtransfer

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"unicode/utf8"
)

const (
	payloadVersion = 1
	chunkSize      = 700
)

// Frame describes one independently scannable part of a file transfer.
type Frame struct {
	Version   int    `json:"v"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	Total     int    `json:"total"`
	Index     int    `json:"index"`
	SHA256    string `json:"sha256"`
	Data      string `json:"data"`
}

// Transfer holds the metadata and file bytes needed to generate QR payloads on demand.
type Transfer struct {
	ID        string
	Name      string
	Size      int64
	Total     int
	SHA256    string
	data      []byte
}

// NewTransfer creates metadata for a file without imposing an application-level size limit.
func NewTransfer(fileName string, data []byte) (*Transfer, error) {
	name := path.Base(strings.ReplaceAll(fileName, `\`, "/"))
	if name == "" || name == "." || !utf8.ValidString(name) {
		return nil, fmt.Errorf("invalid file name")
	}

	var idBytes [16]byte
	if _, err := rand.Read(idBytes[:]); err != nil {
		return nil, fmt.Errorf("create transfer ID: %w", err)
	}

	total := 1
	if len(data) > 0 {
		total = (len(data)-1)/chunkSize + 1
	}
	digest := sha256.Sum256(data)

	return &Transfer{
		ID:     hex.EncodeToString(idBytes[:]),
		Name:   name,
		Size:   int64(len(data)),
		Total:  total,
		SHA256: hex.EncodeToString(digest[:]),
		data:   data,
	}, nil
}

// Payload returns the JSON QR payload for a zero-based frame index.
func (t *Transfer) Payload(index int) (string, error) {
	if index < 0 || index >= t.Total {
		return "", fmt.Errorf("frame index %d is out of range", index)
	}
	start := index * chunkSize
	end := start + chunkSize
	if end > len(t.data) {
		end = len(t.data)
	}
	if start > len(t.data) {
		start = len(t.data)
	}

	frame := Frame{
		Version: payloadVersion,
		ID:      t.ID,
		Name:    t.Name,
		Size:    t.Size,
		Total:   t.Total,
		Index:   index,
		SHA256:  t.SHA256,
		Data:    base64.RawURLEncoding.EncodeToString(t.data[start:end]),
	}
	encoded, err := json.Marshal(frame)
	if err != nil {
		return "", fmt.Errorf("encode QR frame: %w", err)
	}
	return string(encoded), nil
}

// DecodePayload validates a QR frame and returns its decoded chunk bytes.
func DecodePayload(payload string) (Frame, []byte, error) {
	var frame Frame
	if err := json.Unmarshal([]byte(payload), &frame); err != nil {
		return Frame{}, nil, fmt.Errorf("decode QR frame: %w", err)
	}
	if frame.Version != payloadVersion {
		return Frame{}, nil, fmt.Errorf("unsupported QR transfer version %d", frame.Version)
	}
	if len(frame.ID) != 32 {
		return Frame{}, nil, fmt.Errorf("invalid transfer ID")
	}
	if _, err := hex.DecodeString(frame.ID); err != nil {
		return Frame{}, nil, fmt.Errorf("invalid transfer ID")
	}
	if frame.Name == "" || !utf8.ValidString(frame.Name) {
		return Frame{}, nil, fmt.Errorf("invalid file name")
	}
	if frame.Size < 0 || frame.Total < 1 || frame.Index < 0 || frame.Index >= frame.Total {
		return Frame{}, nil, fmt.Errorf("invalid frame metadata")
	}
	expectedTotal := 1
	if frame.Size > 0 {
		expectedTotal = int((frame.Size-1)/chunkSize + 1)
	}
	if int64(expectedTotal) != int64(frame.Total) {
		return Frame{}, nil, fmt.Errorf("frame count does not match file size")
	}
	if len(frame.SHA256) != sha256.Size*2 {
		return Frame{}, nil, fmt.Errorf("invalid file checksum")
	}
	if _, err := hex.DecodeString(frame.SHA256); err != nil {
		return Frame{}, nil, fmt.Errorf("invalid file checksum")
	}

	chunk, err := base64.RawURLEncoding.DecodeString(frame.Data)
	if err != nil {
		return Frame{}, nil, fmt.Errorf("decode QR frame data: %w", err)
	}
	expectedLength := chunkSize
	if frame.Index == frame.Total-1 {
		expectedLength = int(frame.Size - int64(frame.Index*chunkSize))
	}
	if len(chunk) != expectedLength {
		return Frame{}, nil, fmt.Errorf("frame data length does not match frame metadata")
	}
	return frame, chunk, nil
}
