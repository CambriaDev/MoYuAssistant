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
	payloadVersion = 2
	chunkSize      = 700
	manifestType   = "manifest"
	dataType       = "data"
)

// Manifest carries the metadata needed to receive one complete file transfer.
type Manifest struct {
	Version   int    `json:"v"`
	Type      string `json:"type"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	ChunkSize int    `json:"chunkSize"`
	Total     int    `json:"total"`
	SHA256    string `json:"sha256"`
}

// DataFrame carries one independently indexed file chunk.
type DataFrame struct {
	Version int    `json:"v"`
	Type    string `json:"type"`
	ID      string `json:"id"`
	Index   int    `json:"index"`
	Data    string `json:"data"`
}

// DecodedFrame is either a transfer manifest or a decoded data chunk.
type DecodedFrame struct {
	Manifest *Manifest
	Data     *DataFrame
	Chunk    []byte
}

// Transfer holds metadata and file bytes needed to generate QR payloads on demand.
type Transfer struct {
	ID        string
	Name      string
	Size      int64
	Total     int
	ChunkSize int
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
		ID:        hex.EncodeToString(idBytes[:]),
		Name:      name,
		Size:      int64(len(data)),
		Total:     total,
		ChunkSize: chunkSize,
		SHA256:    hex.EncodeToString(digest[:]),
		data:      data,
	}, nil
}

// ManifestPayload returns the metadata QR payload shown before data frames.
func (t *Transfer) ManifestPayload() (string, error) {
	manifest := Manifest{
		Version:   payloadVersion,
		Type:      manifestType,
		ID:        t.ID,
		Name:      t.Name,
		Size:      t.Size,
		ChunkSize: t.ChunkSize,
		Total:     t.Total,
		SHA256:    t.SHA256,
	}
	encoded, err := json.Marshal(manifest)
	if err != nil {
		return "", fmt.Errorf("encode transfer manifest: %w", err)
	}
	return string(encoded), nil
}

// Payload returns the JSON QR payload for a zero-based data-frame index.
func (t *Transfer) Payload(index int) (string, error) {
	if index < 0 || index >= t.Total {
		return "", fmt.Errorf("frame index %d is out of range", index)
	}
	start := index * t.ChunkSize
	end := start + t.ChunkSize
	if end > len(t.data) {
		end = len(t.data)
	}
	if start > len(t.data) {
		start = len(t.data)
	}

	frame := DataFrame{
		Version: payloadVersion,
		Type:    dataType,
		ID:      t.ID,
		Index:   index,
		Data:    base64.RawURLEncoding.EncodeToString(t.data[start:end]),
	}
	encoded, err := json.Marshal(frame)
	if err != nil {
		return "", fmt.Errorf("encode QR data frame: %w", err)
	}
	return string(encoded), nil
}

// DecodePayload validates and decodes either a manifest or a data frame.
func DecodePayload(payload string) (DecodedFrame, error) {
	var envelope struct {
		Version int    `json:"v"`
		Type    string `json:"type"`
	}
	if err := json.Unmarshal([]byte(payload), &envelope); err != nil {
		return DecodedFrame{}, fmt.Errorf("decode QR frame: %w", err)
	}
	if envelope.Version != payloadVersion {
		return DecodedFrame{}, fmt.Errorf("unsupported QR transfer version %d", envelope.Version)
	}

	switch envelope.Type {
	case manifestType:
		var manifest Manifest
		if err := json.Unmarshal([]byte(payload), &manifest); err != nil {
			return DecodedFrame{}, fmt.Errorf("decode transfer manifest: %w", err)
		}
		if err := validateManifest(manifest); err != nil {
			return DecodedFrame{}, err
		}
		return DecodedFrame{Manifest: &manifest}, nil
	case dataType:
		var frame DataFrame
		if err := json.Unmarshal([]byte(payload), &frame); err != nil {
			return DecodedFrame{}, fmt.Errorf("decode QR data frame: %w", err)
		}
		if !validTransferID(frame.ID) || frame.Index < 0 {
			return DecodedFrame{}, fmt.Errorf("invalid data-frame metadata")
		}
		chunk, err := base64.RawURLEncoding.DecodeString(frame.Data)
		if err != nil {
			return DecodedFrame{}, fmt.Errorf("decode QR frame data: %w", err)
		}
		return DecodedFrame{Data: &frame, Chunk: chunk}, nil
	default:
		return DecodedFrame{}, fmt.Errorf("unsupported QR frame type %q", envelope.Type)
	}
}

func validateManifest(manifest Manifest) error {
	if !validTransferID(manifest.ID) {
		return fmt.Errorf("invalid transfer ID")
	}
	if manifest.Name == "" || !utf8.ValidString(manifest.Name) {
		return fmt.Errorf("invalid file name")
	}
	if manifest.Size < 0 || manifest.ChunkSize < 1 || manifest.Total < 1 {
		return fmt.Errorf("invalid manifest metadata")
	}
	expectedTotal := int64(1)
	if manifest.Size > 0 {
		expectedTotal = (manifest.Size-1)/int64(manifest.ChunkSize) + 1
	}
	if expectedTotal != int64(manifest.Total) {
		return fmt.Errorf("frame count does not match file size")
	}
	if !validChecksum(manifest.SHA256) {
		return fmt.Errorf("invalid file checksum")
	}
	return nil
}

func validTransferID(id string) bool {
	if len(id) != 32 {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

func validChecksum(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
