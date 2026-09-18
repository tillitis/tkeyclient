// SPDX-FileCopyrightText: 2026 Tillitis AB <tillitis.se>
// SPDX-License-Identifier: BSD-2-Clause
package tkeyclient

type ResetType uint8

const (
	RstTypeStartFlash0    ResetType = 0
	RstTypeStartFlash1Ver ResetType = 1
	RstTypeStartClient    ResetType = 2
	RstTypeStartClientVer ResetType = 3
)

type NextAppData [126]byte

// NewNextAppDataFromSlice creates a NextAppData from a byte slice.
// Shorter slices are zero-padded; longer slices are truncated to 126 bytes.
func NewNextAppDataFromSlice(b []byte) NextAppData {
	var d NextAppData
	copy(d[:], b)
	return d
}

var (
	VerifierBootSlot1 = NewNextAppDataFromSlice([]byte{0})
	VerifierCmdMode   = NewNextAppDataFromSlice([]byte{1})
)
