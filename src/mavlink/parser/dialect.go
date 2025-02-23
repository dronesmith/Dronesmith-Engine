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

// Dialect represents a set of message definitions.
// Some dialects have conflicting definitions for given message IDs,
// so a list of dialects must be provided to an Encoder/Decoder in
// order to specify which packets to use for the conflicting IDs.
//
// The 'DialectCommon' dialect is added to all Encoders/Decoders by default.
type Dialect struct {
	Name      string
	crcExtras map[uint8]uint8
}

// Alias for a slice of Dialect pointers
// Only really intended to be accessed as a field on Encoder/Decoder
type DialectSlice []*Dialect

// look up the crcextra for msgid
func (ds *DialectSlice) findCrcX(msgid uint8) (uint8, error) {

	// http://www.mavlink.org/mavlink/crc_extra_calculation
	for _, d := range *ds {
		if crcx, ok := d.crcExtras[msgid]; ok {
			return crcx, nil
		}
	}

	return 0, ErrUnknownMsgID
}

// IndexOf returns the index of d or -1 if not found
func (ds *DialectSlice) IndexOf(d *Dialect) int {
	for i, dlct := range *ds {
		if d.Name == dlct.Name {
			return i
		}
	}
	return -1
}

// Add appends d if not already present in ds
func (ds *DialectSlice) Add(d *Dialect) {
	if ds.IndexOf(d) < 0 {
		*ds = append(*ds, d)
	}
}

// Remove removes d if present in ds
func (ds *DialectSlice) Remove(d *Dialect) {
	if i := ds.IndexOf(d); i >= 0 {
		(*ds)[len(*ds)-1], *ds = nil, append((*ds)[:i], (*ds)[i+1:]...)
	}
}
