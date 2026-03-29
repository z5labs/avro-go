// Copyright (c) 2026 Z5Labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package avro

// fpTable is the precomputed CRC-64-AVRO lookup table.
var fpTable = func() [256]uint64 {
	const empty uint64 = 0xc15d213aa4d7a795
	var table [256]uint64
	for i := range 256 {
		fp := uint64(i)
		for range 8 {
			fp = (fp >> 1) ^ (empty & -(fp & 1))
		}
		table[i] = fp
	}
	return table
}()

// Fingerprint64 computes the CRC-64-AVRO fingerprint of the given data.
// This implements the 64-bit Rabin fingerprint algorithm defined in the Avro specification.
func Fingerprint64(data []byte) uint64 {
	const empty uint64 = 0xc15d213aa4d7a795
	fp := empty
	for _, b := range data {
		fp = (fp >> 8) ^ fpTable[byte(fp)^b]
	}
	return fp
}
