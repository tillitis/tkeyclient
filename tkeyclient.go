// SPDX-FileCopyrightText: 2022 Tillitis AB <tillitis.se>
// SPDX-License-Identifier: BSD-2-Clause

// Package tkeyclient provides a connection to a Tillitis TKey
// security stick. To create a new connection:
//
//	tk := tkeyclient.New()
//	err := tk.Connect(port)
//
// Then you can start using it by asking it to identify itself:
//
//	nameVer, err := tk.GetNameVersion()
//
// Or loading and starting an app on the stick:
//
//	err = tk.LoadAppFromFile(*fileName)
//
// After this, you will have to switch to a new protocol specific to
// the app, see for instance the Go package
// https://github.com/tillitis/tkeysign for one such app specific
// protocol to speak to the signer app:
//
// https://github.com/tillitis/tkey-device-signer
//
// When writing your app specific protocol you might still want to use
// the framing protocol provided here. See NewFrameBuf() and
// ReadFrame().
package tkeyclient

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/ccoveille/go-safecast/v2"
	"go.bug.st/serial"
	"golang.org/x/crypto/blake2s"
)

var le = log.New(os.Stderr, "", 0)

func SilenceLogging() {
	le.SetOutput(io.Discard)
}

const (
	// Speed in bps for talking to the TKey
	SerialSpeed = 62500
	// Codes used in app proto responses
	StatusOK  = 0x00
	StatusBad = 0x01

	// Size of RAM in the TKey. See TK1_APP_MAX_SIZE in tk1_mem.h
	AppMaxSize = 0x20000

	// UDI Product IDs
	UDIPIDEngSample         = 0 // XXX Also used for development purposes
	UDIPIDAcrab             = 1
	UDIPIDBellatrix         = 2
	UDIPIDCastor            = 3
	UDIPIDBellatrixUnlocked = 8

	ErrUnsupported = constError("unsupported")
)

// TillitisKey is a serial connection to a TKey and the commands that
// the firmware supports.
type TillitisKey struct {
	path           string
	speed          int
	conn           serial.Port
	forceFullUss   bool
	usbSerial      string
	isUSBDevice    bool
	CanRemoteClose bool
}

// New allocates a new TillitisKey. Use the Connect() method to
// actually open a connection.
func New() *TillitisKey {
	tk := &TillitisKey{}
	return tk
}

// Option to set serial speed. For use with Connect.
func WithSpeed(speed int) func(*TillitisKey) {
	return func(tk *TillitisKey) {
		tk.speed = speed
	}
}

// Option to force use of 32 byte USS digest on Bellatrix and earlier. For use with Connect.
func WithFullUss() func(*TillitisKey) {
	return func(tk *TillitisKey) {
		tk.forceFullUss = true
	}
}

// Option to disable features requiring USB functionality
func NotUSBDevice() func(*TillitisKey) {
	return func(tk *TillitisKey) {
		tk.isUSBDevice = false
	}
}

// Option to disable features that require serial port to be closed by the
// TKey.
func NoRemoteClose() func(*TillitisKey) {
	return func(tk *TillitisKey) {
		tk.CanRemoteClose = false
	}
}

// Connect connects to a TKey serial port using the provided port
// device and options.
func (tk *TillitisKey) Connect(port string, options ...func(*TillitisKey)) error {
	var err error

	tk.speed = SerialSpeed
	tk.forceFullUss = false
	tk.isUSBDevice = true
	tk.CanRemoteClose = true
	for _, opt := range options {
		opt(tk)
	}

	var dev SerialPort

	if port == "" {
		ports, err := GetSerialPorts()
		if err != nil || len(ports) == 0 {
			// No devices.
		}

		if len(ports) > 1 {
			// Too many
		}
		dev = ports[0]
		port = dev.DevPath
	} else {
		var err error

		dev, err = serialPortFromPath(port)
		if err != nil {
			panic(err)
		}
	}

	tk.CanRemoteClose = dev.CanRemoteClose
	tk.usbSerial = dev.SerialNumber

	err = tk.open(port)
	if err != nil {
		return fmt.Errorf("Connect: %w", err)
	}

	return nil
}

func (tk *TillitisKey) open(port string) error {
	var err error

	tk.conn, err = serial.Open(port, &serial.Mode{BaudRate: tk.speed})
	if err != nil {
		// Ensure this value is nil, because Open returns an interface
		tk.conn = nil
		return fmt.Errorf("Open %s: %w", port, err)
	}

	return nil
}

// Close the connection to the TKey
func (tk TillitisKey) Close() error {
	if tk.conn == nil {
		return nil
	}
	if err := tk.conn.Close(); err != nil {
		return fmt.Errorf("conn.Close: %w", err)
	}
	return nil
}

// Reconnect reconnects to a previously connected TKey Castor. Will timeout if
// taking too long.
func (tk *TillitisKey) Reconnect() error {
	var devPath string
	var err error
	retryAttempts := 10
	retryDelay := 100 * time.Millisecond
	timeout := 10 * time.Second

	if !tk.CanRemoteClose {
		le.Printf("Skipping reconnect")
		time.Sleep(time.Second / 2)
		return fmt.Errorf("Reconnect: %w", ErrUnsupported)
	}

	if tk.usbSerial == "" {
		return errors.New("invalid serial nr")
	}

	_ = tk.Close()

	for retryAttempts > 0 {
		// Find TKey by USB serial number
		startTime := time.Now()
		for time.Since(startTime) < timeout {
			devPath, err = pathBySerialNr(tk.usbSerial)
			le.Printf("pathBySerial: '%s', '%v'...\n", devPath, err)
			if err == nil {
				break
			}
			time.Sleep(retryDelay)
		}
		if err != nil {
			fmt.Printf("couldn't find TKey\n")
			return fmt.Errorf("Reconnect: %w", err)
		}

		// On Linux we might get an "Open /dev/ttyACM0: Permission denied"
		// error if we try to reconnect to soon. So we try until we succeed or
		// timeout.
		startTime = time.Now()
		for time.Since(startTime) < timeout {
			err = tk.open(devPath)
			if err == nil {
				break
			}
			time.Sleep(retryDelay)
		}
		if err != nil {
			tk.Close()
			return fmt.Errorf("Reconnect: %w", err)
		}

		// On Linux and Windows it seems like we can connect to the
		// port that we previously closed. Here we check if we can read
		// from the open port and if not then we close the port and try
		// to find the TKey by serial number again.
		var n int
		_ = tk.SetReadTimeout(1)
		n, err = tk.conn.Read([]byte{0})
		if n == 0 {
			le.Printf("Reconnect done tk.conn=%p\n", tk.conn)
			err = tk.SetReadTimeout(0)
			if err != nil {
				return fmt.Errorf("Reconnect: %w", err)
			}
			return nil
		}

		tk.Close()
		retryAttempts--
	}

	return errors.New("Reconnect: timeout")
}

// WaitClosed waits until the underlying serial port is closed.
func (tk TillitisKey) WaitClosed() error {
	var err error
	timeout := 10 * time.Second

	if !tk.CanRemoteClose {
		le.Printf("Skipping port closed detection")
		time.Sleep(time.Second / 2)
		return nil
	}

	defer func() { _ = tk.SetReadTimeout(0) }()
	startTime := time.Now()
	for time.Since(startTime) < timeout {
		err = tk.SetReadTimeout(1)
		if err != nil {
			return fmt.Errorf("WaitClosed: %w", err)
		}

		// Try to read a byte. Treat any error as port closed.
		_, err = tk.conn.Read([]byte{0})
		if err != nil {
			return nil
		}
	}

	return errors.New("WaitClosed: timeout")
}

// SetReadTimeout sets the timeout of the underlying serial connection to the
// TKey. Pass 0 seconds to not have any timeout. Note that the timeout
// implemented in the serial lib only works for simple Read(). E.g.
// io.ReadFull() will Read() until the buffer is full.
//
// Deprecated: use SetReadTimeoutNoErr, which can more easily be used with
// defer.
func (tk TillitisKey) SetReadTimeout(seconds int) error {
	var t time.Duration = -1
	if seconds > 0 {
		t = time.Duration(seconds) * time.Second
	}
	if err := tk.conn.SetReadTimeout(t); err != nil {
		return fmt.Errorf("SetReadTimeout: %w", err)
	}
	return nil
}

// SetReadTimeoutNoErr sets the timeout, in seconds, of the underlying
// serial connection to the TKey. Pass 0 seconds to not have any
// timeout.
//
// Note that the timeout only works for simple Read(). E.g.
// io.ReadFull() will still read until the buffer is full.
func (tk TillitisKey) SetReadTimeoutNoErr(seconds int) {
	var t time.Duration = -1 // disables timeout
	if seconds > 0 {
		t = time.Duration(seconds) * time.Second
	}
	if err := tk.conn.SetReadTimeout(t); err != nil {
		// err != nil exclusively on invalid values of t,
		// which is handled before the call. Panic only
		// possible for API change in go.bug.st/serial
		panic(fmt.Sprintf("SetReadTimeout: %v", err))
	}
	return
}

type NameVersion struct {
	Name0   string
	Name1   string
	Version uint32
}

func (n *NameVersion) Unpack(raw []byte) {
	n.Name0 = fmt.Sprintf("%c%c%c%c", raw[0], raw[1], raw[2], raw[3])
	n.Name1 = fmt.Sprintf("%c%c%c%c", raw[4], raw[5], raw[6], raw[7])
	n.Version = binary.LittleEndian.Uint32(raw[8:12])
}

// GetNameVersion gets the name and version from the TKey firmware
func (tk TillitisKey) GetNameVersion() (*NameVersion, error) {
	id := 2
	tx, err := NewFrameBuf(cmdGetNameVersion, id)
	if err != nil {
		return nil, err
	}

	Dump("GetNameVersion tx", tx)
	if err = tk.Write(tx); err != nil {
		return nil, err
	}

	tk.SetReadTimeoutNoErr(2)
	defer tk.SetReadTimeoutNoErr(0)

	rx, _, err := tk.ReadFrame(rspGetNameVersion, id)
	if err != nil {
		return nil, fmt.Errorf("ReadFrame: %w", err)
	}

	nameVer := &NameVersion{}
	nameVer.Unpack(rx[2:])

	return nameVer, nil
}

func (tk TillitisKey) Reset(t ResetType, d NextAppData) error {
	id := 2

	tx, err := NewFrameBuf(cmdReset, id)
	if err != nil {
		return err
	}

	tx[2] = uint8(t)
	tx[3] = uint8(d)

	if err = tk.Write(tx); err != nil {
		return err
	}

	return nil
}

// Modelled after how tpt.py (in tillitis-key1 repo) generates the UDI
type UDI struct {
	Unnamed         uint8 // 4 bits, hardcoded to 0 by tpt.py
	VendorID        uint16
	ProductID       uint8 // 6 bits
	ProductRevision uint8 // 6 bits
	Serial          uint32
	raw             []byte
}

func (u *UDI) RawBytes() []byte {
	return u.raw
}

func (u *UDI) String() string {
	return fmt.Sprintf("%01x%04x:%x:%x:%08x",
		u.Unnamed, u.VendorID, u.ProductID, u.ProductRevision, u.Serial)
}

// Unpack unpacks the UDI parts from the raw 8 bytes (2 * 32-bit
// words) sent on the wire.
//
// Returns any error
func (u *UDI) Unpack(raw []byte) error {
	var err error

	vpr := binary.LittleEndian.Uint32(raw[0:4])
	u.Unnamed, err = safecast.Convert[uint8]((vpr >> 28) & 0xf)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	u.VendorID, err = safecast.Convert[uint16]((vpr >> 12) & 0xffff)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	u.ProductID, err = safecast.Convert[uint8]((vpr >> 6) & 0x3f)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	u.ProductRevision, err = safecast.Convert[uint8](vpr & 0x3f)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	u.Serial = binary.LittleEndian.Uint32(raw[4:8])
	u.raw = make([]byte, len(raw))
	copy(u.raw, raw)

	return nil
}

// GetUDI gets the UDI (Unique Device ID) from the TKey firmware
func (tk TillitisKey) GetUDI() (*UDI, error) {
	id := 2
	tx, err := NewFrameBuf(cmdGetUDI, id)
	if err != nil {
		return nil, err
	}

	Dump("GetUDI tx", tx)
	if err = tk.Write(tx); err != nil {
		return nil, err
	}

	rx, _, err := tk.ReadFrame(rspGetUDI, id)
	if err != nil {
		return nil, fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[2] != StatusOK {
		return nil, fmt.Errorf("GetUDI NOK")
	}

	udi := &UDI{}
	err = udi.Unpack(rx[3 : 3+8])
	if err != nil {
		return nil, fmt.Errorf("couldn't unpack UDI: %w", err)
	}

	return udi, nil
}

// LoadAppFromFile loads and runs a raw binary file from fileName into
// the TKey.
func (tk TillitisKey) LoadAppFromFile(fileName string, secretPhrase []byte) error {
	content, err := os.ReadFile(fileName)
	if err != nil {
		return fmt.Errorf("ReadFile: %w", err)
	}

	return tk.LoadApp(content, secretPhrase)
}

// LoadApp sends a device app in bin and optionally a User Supplied
// Secret digest to the TKey.
//
// After successfully sending the device app it computes a digest over
// bin and compares it to what was returned from the TKey, returning
// an error if it isn't equal.
//
// The USS is a BLAKE2s digest of the secretPhrase argument. How much
// of the digest is used differs on different hardware:
//
//   - Bellatrix and earlier: last 31 bytes of the digest for backward
//     compatibility reasons.
//
//   - Castor and later: all 32 bytes used.
//
// Returns ErrResponseStatusNotOK if firmware is not detected.
func (tk TillitisKey) LoadApp(bin []byte, secretPhrase []byte) error {
	binLen := len(bin)
	if binLen > AppMaxSize {
		return fmt.Errorf("File too big")
	}

	le.Printf("app size: %v, 0x%x, 0b%b\n", binLen, binLen, binLen)

	udi, err := tk.GetUDI()
	if err != nil {
		// Probably not running firmware? Or communication error
		return err
	}

	if err = tk.loadApp(binLen, secretPhrase, udi.ProductID); err != nil {
		return err
	}

	// Load the file
	var offset int
	var deviceDigest [32]byte

	for nsent := 0; offset < binLen; offset += nsent {
		if binLen-offset <= cmdLoadAppData.CmdLen().Bytelen()-1 {
			deviceDigest, nsent, err = tk.loadAppData(bin[offset:], true)
		} else {
			_, nsent, err = tk.loadAppData(bin[offset:], false)
		}
		if err != nil {
			return fmt.Errorf("loadAppData: %w", err)
		}
	}
	if offset > binLen {
		return fmt.Errorf("transmitted more than expected")
	}

	digest := blake2s.Sum256(bin)

	le.Printf("Digest from host:\n")
	printDigest(digest)
	le.Printf("Digest from device:\n")
	printDigest(deviceDigest)

	if deviceDigest != digest {
		return fmt.Errorf("Different digests")
	}
	le.Printf("Same digests!\n")

	// The app has now started automatically.
	return nil
}

// loadApp sets the size and USS of the app to be loaded into the TKey.
func (tk TillitisKey) loadApp(size int, secretPhrase []byte, pid uint8) error {
	id := 2
	tx, err := NewFrameBuf(cmdLoadApp, id)
	if err != nil {
		return err
	}

	// Set size
	tx[2] = byte(size)
	tx[3] = byte(size >> 8)
	tx[4] = byte(size >> 16)
	tx[5] = byte(size >> 24)

	if len(secretPhrase) == 0 {
		tx[6] = 0
	} else {
		tx[6] = 1
		// Hash user's phrase as USS
		uss := blake2s.Sum256(secretPhrase)

		// Skip first byte of computed digest for backwards
		// compatibility unless forced to send 32 bytes with
		// the WithFullUss() option function or we have
		// identified a Castor.
		switch pid {
		case UDIPIDCastor:
			copy(tx[7:], uss[:])

		case UDIPIDEngSample:
			fallthrough
		case UDIPIDAcrab:
			fallthrough
		case UDIPIDBellatrix:
			fallthrough
		case UDIPIDBellatrixUnlocked:
			fallthrough
		default:
			if tk.forceFullUss {
				copy(tx[7:], uss[:])
			} else {
				copy(tx[7:], uss[1:])
			}
		}
	}

	Dump("LoadApp tx", tx)
	if err = tk.Write(tx); err != nil {
		return err
	}

	rx, _, err := tk.ReadFrame(rspLoadApp, id)
	if err != nil {
		return fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[2] != StatusOK {
		return fmt.Errorf("LoadApp NOK")
	}

	return nil
}

// loadAppData loads a chunk of the raw app binary into the TKey.
func (tk TillitisKey) loadAppData(content []byte, last bool) ([32]byte, int, error) {
	id := 2
	tx, err := NewFrameBuf(cmdLoadAppData, id)
	if err != nil {
		return [32]byte{}, 0, err
	}

	payload := make([]byte, cmdLoadAppData.CmdLen().Bytelen()-1)
	copied := copy(payload, content)

	// Add padding if not filling the payload buffer.
	if copied < len(payload) {
		padding := make([]byte, len(payload)-copied)
		copy(payload[copied:], padding)
	}

	copy(tx[2:], payload)

	Dump("LoadAppData tx", tx)

	if err = tk.Write(tx); err != nil {
		return [32]byte{}, 0, err
	}

	var rx []byte
	var expectedResp Cmd

	if last {
		expectedResp = rspLoadAppDataReady
	} else {
		expectedResp = rspLoadAppData
	}

	// Wait for reply
	rx, _, err = tk.ReadFrame(expectedResp, id)
	if err != nil {
		return [32]byte{}, 0, fmt.Errorf("ReadFrame: %w", err)
	}

	if rx[2] != StatusOK {
		return [32]byte{}, 0, fmt.Errorf("LoadAppData NOK")
	}

	if last {
		var digest [32]byte
		copy(digest[:], rx[3:])
		return digest, copied, nil
	}

	return [32]byte{}, copied, nil
}

func printDigest(md [32]byte) {
	digest := ""
	for j := 0; j < 4; j++ {
		for i := 0; i < 8; i++ {
			digest += fmt.Sprintf("%02x", md[i+8*j])
		}
		digest += " "
	}
	le.Print(digest + "\n")
}
