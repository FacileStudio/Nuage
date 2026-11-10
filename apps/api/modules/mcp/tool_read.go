package mcp

import (
	"bytes"
	"context"
	stderrors "errors"
	"fmt"
	"io"

	mcpkit "github.com/FacileStudio/tronc/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Limits of nuage_read_file.
const (
	maxReadBytes = 60_000
	sniffBytes   = 8 << 10
)

// errOffset is the refusal for an offset past the end of the file.
var errOffset = stderrors.New("offset is past the end of the file. Start again with offset 0")

// readFile is nuage_read_file.
func (t *toolset) readFile(ctx context.Context, req *sdk.CallToolRequest, in readInput) (*sdk.CallToolResult, readOutput, error) {
	userID, err := t.userOf(ctx, req)
	if err != nil {
		return nil, readOutput{}, err
	}
	body, f, err := t.Files.Open(ctx, userID, in.FileID)
	if err != nil {
		return nil, readOutput{}, failure(err, hintItem)
	}
	defer func() { _ = body.Close() }()
	out := readOutput{FileID: f.ID, Name: f.Name, MimeType: f.MimeType, Size: f.Size, Offset: in.Offset}
	if in.Offset < 0 || (in.Offset > 0 && in.Offset >= f.Size) {
		return nil, readOutput{}, errOffset
	}
	text, err := isText(body, f.MimeType, f.Name)
	if err != nil {
		return nil, readOutput{}, mcpkit.ErrUnavailable
	}
	if !text {
		out.Note = fmt.Sprintf("This is a binary file (%s, %d bytes), so it cannot be read as text. The user can open it in Nuage.", f.MimeType, f.Size)
		return nil, out, nil
	}
	out.Text = true
	chunk, err := readChunk(body, in.Offset, clamp(in.MaxBytes, maxReadBytes, maxReadBytes))
	if err != nil {
		return nil, readOutput{}, mcpkit.ErrUnavailable
	}
	return nil, fill(out, chunk), nil
}

// isText samples the first bytes, then rewinds body. A NUL byte means
// binary whatever the declared type; otherwise a declared text type or
// name is trusted, and anything else must be valid UTF-8.
func isText(body io.ReadSeeker, mimeType, name string) (bool, error) {
	sample := make([]byte, sniffBytes)
	n, err := io.ReadFull(body, sample)
	if err != nil && !stderrors.Is(err, io.ErrUnexpectedEOF) && !stderrors.Is(err, io.EOF) {
		return false, err
	}
	if _, err := body.Seek(0, io.SeekStart); err != nil {
		return false, err
	}
	sample = sample[:n]
	if bytes.IndexByte(sample, 0) >= 0 {
		return false, nil
	}
	return n == 0 || declaresText(mimeType, name) || looksText(sample), nil
}

// chunk is a slice of a text file aligned on runes.
type chunk struct {
	start int64
	data  []byte
}

// readChunk reads up to limit bytes from offset, moved forward past a rune
// begun before it and cut back to the last whole rune.
func readChunk(body io.ReadSeeker, offset int64, limit int) (chunk, error) {
	if _, err := body.Seek(offset, io.SeekStart); err != nil {
		return chunk{}, err
	}
	buf := make([]byte, limit+4)
	n, err := io.ReadFull(body, buf)
	if err != nil && !stderrors.Is(err, io.ErrUnexpectedEOF) && !stderrors.Is(err, io.EOF) {
		return chunk{}, err
	}
	data := buf[:n]
	skip := 0
	if offset > 0 {
		skip = leadingContinuation(data)
	}
	data = runeAligned(data[skip:min(len(data), skip+limit)])
	return chunk{start: offset + int64(skip), data: data}, nil
}

// fill puts the chunk in out, shortened until the result fits the budget,
// with the offset to read on from.
func fill(out readOutput, c chunk) readOutput {
	out.Offset = c.start
	keep := mcpkit.FitCount(len(c.data), func(n int) int {
		probe := out
		probe.Content, probe.NextOffset, probe.Truncated, probe.Note = string(c.data[:n]), out.Size, true, "placeholder note of a typical length for the budget"
		return mcpkit.EncodedSize(probe)
	})
	data := runeAligned(c.data[:keep])
	out.Content = string(data)
	if end := c.start + int64(len(data)); end < out.Size {
		out.NextOffset, out.Truncated = end, true
		out.Note = fmt.Sprintf("Shortened: the file is %d bytes. Call again with offset %d for the rest.", out.Size, end)
	}
	return out
}
