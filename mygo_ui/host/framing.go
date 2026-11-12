// Package host speaks desktop/HOST-PROTOCOL.md over stdio with spektralab-host.
//
// Frame layout (both directions):
//
//	u32 LE  header_len
//	u32 LE  payload_len
//	u8[header_len]  UTF-8 JSON
//	u8[payload_len] binary payload
package host

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// writeFrame writes one framed message.
func writeFrame(w io.Writer, header any, payload []byte) error {
	hb, err := json.Marshal(header)
	if err != nil {
		return err
	}
	var lens [8]byte
	binary.LittleEndian.PutUint32(lens[0:4], uint32(len(hb)))
	binary.LittleEndian.PutUint32(lens[4:8], uint32(len(payload)))
	if _, err := w.Write(lens[:]); err != nil {
		return err
	}
	if _, err := w.Write(hb); err != nil {
		return err
	}
	if len(payload) > 0 {
		_, err = w.Write(payload)
	}
	return err
}

// readFrame reads one framed message and returns the raw JSON header and payload.
func readFrame(r io.Reader) (header []byte, payload []byte, err error) {
	var lens [8]byte
	if _, err = io.ReadFull(r, lens[:]); err != nil {
		return nil, nil, err
	}
	hlen := binary.LittleEndian.Uint32(lens[0:4])
	plen := binary.LittleEndian.Uint32(lens[4:8])
	if hlen > 16<<20 || plen > 512<<20 {
		return nil, nil, fmt.Errorf("host frame too large: header=%d payload=%d", hlen, plen)
	}
	header = make([]byte, hlen)
	if _, err = io.ReadFull(r, header); err != nil {
		return nil, nil, err
	}
	if plen > 0 {
		payload = make([]byte, plen)
		if _, err = io.ReadFull(r, payload); err != nil {
			return nil, nil, err
		}
	}
	return header, payload, nil
}
