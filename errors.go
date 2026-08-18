// SPDX-FileCopyrightText: 2026 Tillitis AB <tillitis.se>
// SPDX-License-Identifier: BSD-2-Clause

package tkeyclient

type constError string

func (err constError) Error() string {
	return string(err)
}

func (err constError) Unwrap() error {
	return err
}
