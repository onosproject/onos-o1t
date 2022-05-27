// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// File adapted from content of repository
// https://github.com/andaru/netconf/

package ssh

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
)

const (
	// msgSeparator is used to separate sent rawMessages via NETCONF v1.1
	msgSeparator = "\n##\n"
)

// ErrBadChunk indicates a chunked framing protocol error occurred
var ErrBadChunk = errors.New("bad chunk")

type conn struct {
	io.Reader
	io.WriteCloser
	sync.Mutex
}

type serverConn struct {
	conn
}

func (c *conn) send(data []byte) error {
	c.Lock()
	defer c.Unlock()

	var separator []byte
	var rawMessage []byte
	separator = append(separator, []byte(msgSeparator)...)
	header := fmt.Sprintf("\n#%d\n", len(string(data)))

	rawMessage = append(rawMessage, header...)
	rawMessage = append(rawMessage, data...)
	rawMessage = append(rawMessage, separator...)

	_, err := c.Write(rawMessage)

	return err
}

func (c *conn) receive() ([]byte, error) {
	var separator []byte
	separator = append(separator, []byte(msgSeparator)...)

	b, err := c.receiveUntil(separator)
	if err != nil {
		return nil, err
	}

	return framing(b)
}

func (c *conn) receiveUntil(separator []byte) ([]byte, error) {
	var errWait error
	var out bytes.Buffer
	buf := make([]byte, 8192)

	pos := 0
	for {
		n, err := c.Read(buf[pos : pos+(len(buf)/2)])
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			errWait = err
			break
		}

		if n > 0 {
			end := bytes.Index(buf[0:pos+n], separator)
			if err != nil {
				return nil, err
			}

			if end > -1 {
				out.Write(buf[0:end])
				return out.Bytes(), nil
			}

			if pos > 0 {
				out.Write(buf[0:pos])
				copy(buf, buf[pos:pos+n])
			}

			pos = n
		}
	}

	return nil, errWait
}

func framing(b []byte) ([]byte, error) {
	rdr := bytes.NewReader(b)
	scanner := bufio.NewScanner(rdr)
	bsize := 16
	scanner.Buffer(make([]byte, bsize), bsize*2)

	scanner.Split(splitChunked(nil))
	var got []byte
	for scanner.Scan() {
		got = append(got, scanner.Bytes()...)
	}
	return got, nil
}

func splitChunked(endOfrawMessage func()) bufio.SplitFunc {
	type stateT int
	const (
		headerStart stateT = iota
		headerSize
		data
		endOfChunks
	)
	var state stateT
	var cs, dataleft int

	return func(b []byte, atEOF bool) (advance int, token []byte, err error) {
		for cur := b[advance:]; err == nil && advance < len(b); cur = b[advance:] {
			if len(cur) < 4 && !atEOF {
				return
			}
			switch state {
			case headerStart:
				switch {
				case bytes.HasPrefix(cur, []byte("\n#")):
					if len(cur) < 4 {
						err = ErrBadChunk
						return
					}
					switch r := cur[2]; {
					case r == '#':
						advance += 3
						state = endOfChunks
					case r >= '1' && r <= '9':
						advance += 2
						state = headerSize
					default:
						err = ErrBadChunk
					}
				default:
					err = ErrBadChunk
				}
			case headerSize:
				switch idx := bytes.IndexByte(cur, '\n'); {
				case idx < 1, idx > 10:
					if len(cur) < 11 && !atEOF {
						return
					}
					err = ErrBadChunk
				default:
					csize := cur[:idx]
					if csizeVal, csizeErr := strconv.ParseUint(string(csize), 10, 31); csizeErr != nil {
						err = ErrBadChunk
					} else {
						advance += idx + 1
						dataleft = int(csizeVal)
						state = data
					}
				}
			case data:
				var rsize int
				if rsize = len(cur); dataleft < rsize {
					rsize = dataleft
				}
				token = append(token, cur[:rsize]...)
				advance += rsize
				if dataleft -= rsize; dataleft < 1 {
					state = headerStart
					cs++
				}
				if rsize > 0 {
					return
				}
			case endOfChunks:
				switch r := cur[0]; {
				case r == '\n' && cs > 0:
					advance++
					state = headerStart
					if endOfrawMessage != nil {
						endOfrawMessage()
					}
				default:
					err = ErrBadChunk
				}
			}
		}
		if atEOF && dataleft > 0 || state != headerStart {
			err = io.ErrUnexpectedEOF
		}
		return
	}
}
