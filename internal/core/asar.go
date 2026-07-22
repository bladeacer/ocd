package core

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type asarHeader struct {
	Files map[string]asarEntry `json:"files"`
}

type asarEntry struct {
	Offset string               `json:"offset"`
	Size   int                  `json:"size"`
	Files  map[string]asarEntry `json:"files,omitempty"`
}

// Obsidian ASAR format (Electron 28+):
//
//	[0-3]   unknown (ignored)
//	[4-11]  unknown (ignored)
//	[12-15] json header size (uint32 LE)
//	[16+]   JSON header string
//	[16+jsonSize]  file data
const asarJSONOffset = 16
const asarMaxHeaderSize = 10 * 1024 * 1024

func extractAppCSSFromASAR(asarPath, destPath string) error {
	f, err := os.Open(asarPath)
	if err != nil {
		return fmt.Errorf("open asar: %w", err)
	}
	defer f.Close()

	raw := make([]byte, 12)
	if _, err := io.ReadFull(f, raw); err != nil {
		return fmt.Errorf("read asar header prefix: %w", err)
	}

	var jsonSize uint32
	if err := binary.Read(f, binary.LittleEndian, &jsonSize); err != nil {
		return fmt.Errorf("read asar json header size: %w", err)
	}
	if jsonSize == 0 || jsonSize > asarMaxHeaderSize {
		return fmt.Errorf("invalid asar header size: %d", jsonSize)
	}

	jsonBuf := make([]byte, jsonSize)
	if _, err := io.ReadFull(f, jsonBuf); err != nil {
		return fmt.Errorf("read asar json header: %w", err)
	}

	var header asarHeader
	if err := json.Unmarshal(jsonBuf, &header); err != nil {
		return fmt.Errorf("parse asar header: %w", err)
	}

	entry, ok := header.Files["app.css"]
	if !ok {
		return fmt.Errorf("app.css not found in asar archive")
	}

	var offset int64
	if _, err := fmt.Sscanf(entry.Offset, "%d", &offset); err != nil {
		return fmt.Errorf("parse asar offset: %w", err)
	}

	dataOffset := int64(asarJSONOffset + int(jsonSize))

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer out.Close()

	section := io.NewSectionReader(f, dataOffset+offset, int64(entry.Size))
	if _, err := io.Copy(out, section); err != nil {
		return fmt.Errorf("extract app.css: %w", err)
	}

	return nil
}
