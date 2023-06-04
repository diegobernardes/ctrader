package ctrader

import (
	"bufio"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type transportTCP struct {
	deadline       time.Duration
	conn           *tls.Conn
	reader         io.Reader
	sendMutex      sync.Mutex
	wg             sync.WaitGroup
	stopSignal     atomic.Bool
	handlerMessage func([]byte)
	handlerError   func(error)
}

// start should only be used after setHandlerMessage and setHandlerError functions are called.
func (t *transportTCP) start(address string) error {
	tlsDialer := net.Dialer{
		Deadline: time.Now().Add(t.deadline),
	}
	conn, err := tls.DialWithDialer(&tlsDialer, "tcp", address, nil)
	if err != nil {
		return fmt.Errorf("tls dial failed: %w", err)
	}
	t.conn = conn
	t.reader = bufio.NewReader(t.conn)
	t.receive()
	return nil
}

func (t *transportTCP) stop() error {
	t.stopSignal.Store(true)
	t.wg.Wait()
	if err := t.conn.Close(); err != nil {
		return fmt.Errorf("connection close failed: %w", err)
	}
	return nil
}

func (t *transportTCP) send(payload []byte) error {
	t.sendMutex.Lock()
	defer t.sendMutex.Unlock()

	// https://help.ctrader.com/open-api/sending-receiving-protobuf/#using-tcp
	arr := make([]byte, 4)
	binary.BigEndian.PutUint32(arr, uint32(len(payload)))

	if err := t.conn.SetWriteDeadline(time.Now().Add(t.deadline)); err != nil {
		return fmt.Errorf("failed to set the write deadline into the connection: %w", err)
	}
	written, err := t.conn.Write(append(arr, payload...))
	if err != nil {
		return fmt.Errorf("connection write failed: %w", err)
	}
	if written != len(payload)+4 {
		return fmt.Errorf("incomplete message %d bytes written but was expected %d", written, len(payload))
	}

	return nil
}

func (t *transportTCP) receive() {
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		bufferLength := make([]byte, 4)
		bufferPayload := make([]byte, 0)
		for {
			if t.stopSignal.Load() {
				break
			}

			if err := t.conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
				t.handlerError(fmt.Errorf("failed to set the read deadline: %w", err))
				return
			}

			if _, err := io.ReadFull(t.reader, bufferLength); err != nil {
				if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
					continue
				}
				t.handlerError(fmt.Errorf("failed to read: %w", err))
				return
			}

			payloadLength := int(binary.BigEndian.Uint32(bufferLength))
			if payloadLength > cap(bufferPayload) {
				bufferPayload = make([]byte, payloadLength)
			} else {
				bufferPayload = bufferPayload[:payloadLength]
			}
			if err := t.conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
				t.handlerError(fmt.Errorf("failed to set the read deadline: %w", err))
				return
			}
			if _, err := io.ReadFull(t.reader, bufferPayload); err != nil {
				if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
					continue
				}
				t.handlerError(fmt.Errorf("failed to read: %w", err))
				return
			}
			t.handlerMessage(bufferPayload)
		}
	}()
}

func (t *transportTCP) setHandler(handlerMessage func([]byte), handlerError func(error)) {
	t.handlerMessage = handlerMessage
	t.handlerError = handlerError
}
