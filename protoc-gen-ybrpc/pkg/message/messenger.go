package message

import "google.golang.org/protobuf/proto"

type Messenger interface {
	GetHost() string
	SendMessage(service, method string, request, response proto.Message) error
}
