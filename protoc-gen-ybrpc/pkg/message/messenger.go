package message

import "google.golang.org/protobuf/proto"

type Messenger interface {
	SendMessage(service, method string, request, response proto.Message) error
}
