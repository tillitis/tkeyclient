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

type NextAppData uint8

const (
	VerifierBootSlot1 NextAppData = 0
	VerifierCmdMode   NextAppData = 1
)
