package message

import (
	"bytes"
	"encoding/binary"
	"fmt"

	. "github.com/icza/gox/gox"
	"github.com/pkg/errors"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/rpc"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client/session"
	"google.golang.org/protobuf/proto"
)

type MessengerImpl struct {
	Session *session.Session
}

// TODO: Needs to copy the buffer to the connection
// TODO: Needs extensive tests
// TODO: This should actually be implemented as a message queue
func (m *MessengerImpl) SendMessage(service string, method string, request proto.Message, response proto.Message) error {
	m.Session.Lock()
	defer m.Session.Unlock()

	callID := m.Session.GenerateCallID()
	pb := &rpc.RequestHeader{
		CallId: &callID,
		RemoteMethod: &rpc.RemoteMethodPB{
			ServiceName: &service,
			MethodName:  &method,
		},
		// TODO: Timout should be a config value as part of client configuration
		TimeoutMillis: NewUint32(3000),
	}
	// TODO: what to do about sidecars?

	messageHeader, err := proto.Marshal(pb)
	if err != nil {
		return err
	}

	var buf [binary.MaxVarintLen32]byte
	encodedLen := binary.PutUvarint(buf[:], uint64(len(messageHeader)))

	var b bytes.Buffer
	b.Write(buf[0:encodedLen])
	b.Write(messageHeader)

	messageBody, err := proto.Marshal(request)
	if err != nil {
		return err
	}

	encodedLen = binary.PutUvarint(buf[:], uint64(len(messageBody)))

	b.Write(buf[0:encodedLen])

	b.Write(messageBody)
	messageLen := uint32(b.Len())

	var intBytes [4]byte
	binary.BigEndian.PutUint32(intBytes[:], messageLen)

	n, err := m.Session.Write(intBytes[:])
	if err != nil {
		return err
	}
	if n != 4 {
		return errors.New("packet len not 4 bytes")
	}

	n, err = m.Session.Write(b.Bytes())
	if err != nil {
		return err
	}
	if n != int(messageLen) {
		return errors.New("request over the wire not equal to messageLen")
	}
	return m.GetResponse(callID, response)
}

func (m *MessengerImpl) GetResponse(callID int32, response proto.Message) error {
	responseLen, err := m.getMessageLen()
	if err != nil {
		return fmt.Errorf("invalid message len callID %d: %w", callID, err)
	}

	var offset, n int
	responseBuf := make([]byte, responseLen)
	for offset = 0; offset < int(responseLen); offset = offset + n {
		n, err = m.Session.Read(responseBuf[offset:])
		if err != nil {
			return err
		}
	}

	offset = 0
	value, nbytes := binary.Uvarint(responseBuf)
	if nbytes <= 0 {
		return errors.New("varint corruption")
	}
	offset = offset + nbytes

	responseHeader := &rpc.ResponseHeader{}
	err = proto.Unmarshal(responseBuf[offset:offset+int(value)], responseHeader)
	if err != nil {
		return err
	}
	offset = offset + int(value)
	if responseHeader.GetCallId() != callID {
		return errors.New("unknown call ID in response")
	}

	if offset != int(responseLen-1) {
		_, nbytes = binary.Uvarint(responseBuf[offset:])
		if nbytes <= 0 {
			return errors.New("varint corruption")
		}
		offset = offset + nbytes

		err = proto.Unmarshal(responseBuf[offset:], response)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MessengerImpl) getMessageLen() (uint32, error) {
	responseLenBuf := make([]byte, 4)

	n, err := m.Session.Read(responseLenBuf)
	if err != nil {
		return 0, err
	}
	if n != 4 {
		return 0, errors.New("could not read response length")
	}

	// It is valid to receive an empty message, so if we get all zeros move on to read the next message
	emptyMessage := true
	for _, responseByte := range responseLenBuf {
		if responseByte != 0 {
			emptyMessage = false
		}
	}

	if emptyMessage {
		return m.getMessageLen()
	}

	return binary.BigEndian.Uint32(responseLenBuf), nil
}
