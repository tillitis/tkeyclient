// SPDX-FileCopyrightText: 2026 Tillitis AB <tillitis.se>
// SPDX-License-Identifier: BSD-2-Clause
package tkeyclient

type ResetType uint8

const (
	RstTypeStartDefault   ResetType = 0
	RstTypeStartFlash0    ResetType = 1
	RstTypeStartFlash1    ResetType = 2
	RstTypeStartFlash0Ver ResetType = 3
	RstTypeStartFlash1Ver ResetType = 4
	RstTypeStartClient    ResetType = 5
	RstTypeStartClientVer ResetType = 6
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
