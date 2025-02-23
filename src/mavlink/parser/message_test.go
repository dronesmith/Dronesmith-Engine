/**
 * Dronesmith API
 *
 * Authors
 *  Geoff Gardner <geoff@dronesmith.io>
 *
 * Copyright (C) 2016 Dronesmith Technologies Inc, all rights reserved.
 * Unauthorized copying of any source code or assets within this project, via
 * any medium is strictly prohibited.
 *
 * Proprietary and confidential.
 */
 
package mavlink

import (
	"bytes"
	"testing"
)

func TestRoundTrip(t *testing.T) {

	cases := []struct{ seq uint32 }{
		{12345},
	}

	for _, c := range cases {
		p := Ping{
			Seq: c.seq,
		}

		var pkt Packet
		if err := p.Pack(&pkt); err != nil {
			t.Errorf("Pack fail %q (%q)", pkt, err)
		}

		var buf bytes.Buffer

		if err := NewEncoder(&buf).EncodePacket(&pkt); err != nil {
			t.Errorf("Encode fail %q", err)
		}

		pktOut, err := NewDecoder(&buf).Decode()
		if err != nil {
			t.Errorf("Decode fail %q", err)
		}

		if pktOut.MsgID != MSG_ID_PING {
			t.Errorf("MsgID fail, want %q, got %q", MSG_ID_PING, pktOut.MsgID)
		}

		var pingOut Ping
		if err := pingOut.Unpack(pktOut); err != nil {
			t.Errorf("Unpack fail %q", err)
		}

		if pingOut.Seq != c.seq {
			t.Errorf("Mismatch msg field, got %q, want %q", pingOut.Seq, c.seq)
		}
	}
}

func TestDecode(t *testing.T) {
	// decode a known good byte stream
	pktbytes := []byte{0xfe, 0x09, 0x0, 0x01, 0xC8, 0x00, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x5A, 0x3E}
	_, err := NewDecoder(bytes.NewBuffer(pktbytes)).Decode()
	if err != nil {
		t.Errorf("Decode fail:", err)
	}
}

func TestDialects(t *testing.T) {

	var buf bytes.Buffer

	enc := NewEncoder(&buf)
	dec := NewDecoder(&buf)

	// try to encode an ardupilot msg before we've added that dialect,
	// ensure it fails as expected
	mi := &Meminfo{
		Brkval:  1000,
		Freemem: 10,
	}

	err := enc.Encode(0x1, 0x1, mi)
	if err != ErrUnknownMsgID {
		t.Errorf("encode expected ErrUnknownMsgID, got %q", err)
	}

	buf.Reset()

	// add the dialect, and ensure it succeeds
	enc.Dialects.Add(DialectArdupilotmega)
	if err = enc.Encode(0x1, 0x1, mi); err != nil {
		t.Errorf("Encode fail %q", err)
	}

	_, err = NewDecoder(&buf).Decode()
	if err != ErrUnknownMsgID {
		t.Errorf("decode expected ErrUnknownMsgID, got %q", err)
	}

	dec.Dialects.Add(DialectArdupilotmega)

	// re-encode the msg, and decode it again after adding the required dialect
	if err = enc.Encode(0x1, 0x1, mi); err != nil {
		t.Errorf("Encode fail %q", err)
	}

	pktOut, err := dec.Decode()
	if err != nil {
		t.Errorf("Decode fail %q", err)
	}

	// make sure the output matches our original input for good measure
	var miOut Meminfo
	if err := miOut.Unpack(pktOut); err != nil {
		t.Errorf("Unpack fail %q", err)
	}

	if miOut.Brkval != mi.Brkval || miOut.Freemem != mi.Freemem {
		t.Errorf("Round trip fail")
	}
}
