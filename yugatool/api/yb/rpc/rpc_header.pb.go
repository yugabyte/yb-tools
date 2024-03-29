// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//
// The following only applies to changes made to this file as part of YugaByte development.
//
// Portions Copyright (c) YugaByte, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.  You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed under the License
// is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
// or implied.  See the License for the specific language governing permissions and limitations
// under the License.
//

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.25.0-devel
// 	protoc        v3.14.0
// source: yb/rpc/rpc_header.proto

package rpc

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoiface "google.golang.org/protobuf/runtime/protoiface"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// These codes have all been inherited from Hadoop's RPC mechanism.
type ErrorStatusPB_RpcErrorCodePB int32

const (
	ErrorStatusPB_FATAL_UNKNOWN ErrorStatusPB_RpcErrorCodePB = 10
	// Non-fatal RPC errors. Connection should be left open for future RPC calls.
	//------------------------------------------------------------
	// The application generated an error status. See the message field for
	// more details.
	ErrorStatusPB_ERROR_APPLICATION ErrorStatusPB_RpcErrorCodePB = 1
	// The specified method was not valid.
	ErrorStatusPB_ERROR_NO_SUCH_METHOD ErrorStatusPB_RpcErrorCodePB = 2
	// The specified service was not valid.
	ErrorStatusPB_ERROR_NO_SUCH_SERVICE ErrorStatusPB_RpcErrorCodePB = 3
	// The server is overloaded - the client should try again shortly.
	ErrorStatusPB_ERROR_SERVER_TOO_BUSY ErrorStatusPB_RpcErrorCodePB = 4
	// The request parameter was not parseable or was missing required fields.
	ErrorStatusPB_ERROR_INVALID_REQUEST ErrorStatusPB_RpcErrorCodePB = 5
	// FATAL_* errors indicate that the client should shut down the connection.
	//------------------------------------------------------------
	// The RPC server is already shutting down.
	ErrorStatusPB_FATAL_SERVER_SHUTTING_DOWN ErrorStatusPB_RpcErrorCodePB = 11
	// Could not deserialize RPC request.
	ErrorStatusPB_FATAL_DESERIALIZING_REQUEST ErrorStatusPB_RpcErrorCodePB = 13
	// IPC Layer version mismatch.
	ErrorStatusPB_FATAL_VERSION_MISMATCH ErrorStatusPB_RpcErrorCodePB = 14
	// Auth failed.
	ErrorStatusPB_FATAL_UNAUTHORIZED ErrorStatusPB_RpcErrorCodePB = 15
)

// Enum value maps for ErrorStatusPB_RpcErrorCodePB.
var (
	ErrorStatusPB_RpcErrorCodePB_name = map[int32]string{
		10: "FATAL_UNKNOWN",
		1:  "ERROR_APPLICATION",
		2:  "ERROR_NO_SUCH_METHOD",
		3:  "ERROR_NO_SUCH_SERVICE",
		4:  "ERROR_SERVER_TOO_BUSY",
		5:  "ERROR_INVALID_REQUEST",
		11: "FATAL_SERVER_SHUTTING_DOWN",
		13: "FATAL_DESERIALIZING_REQUEST",
		14: "FATAL_VERSION_MISMATCH",
		15: "FATAL_UNAUTHORIZED",
	}
	ErrorStatusPB_RpcErrorCodePB_value = map[string]int32{
		"FATAL_UNKNOWN":               10,
		"ERROR_APPLICATION":           1,
		"ERROR_NO_SUCH_METHOD":        2,
		"ERROR_NO_SUCH_SERVICE":       3,
		"ERROR_SERVER_TOO_BUSY":       4,
		"ERROR_INVALID_REQUEST":       5,
		"FATAL_SERVER_SHUTTING_DOWN":  11,
		"FATAL_DESERIALIZING_REQUEST": 13,
		"FATAL_VERSION_MISMATCH":      14,
		"FATAL_UNAUTHORIZED":          15,
	}
)

func (x ErrorStatusPB_RpcErrorCodePB) Enum() *ErrorStatusPB_RpcErrorCodePB {
	p := new(ErrorStatusPB_RpcErrorCodePB)
	*p = x
	return p
}

func (x ErrorStatusPB_RpcErrorCodePB) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ErrorStatusPB_RpcErrorCodePB) Descriptor() protoreflect.EnumDescriptor {
	return file_yb_rpc_rpc_header_proto_enumTypes[0].Descriptor()
}

func (ErrorStatusPB_RpcErrorCodePB) Type() protoreflect.EnumType {
	return &file_yb_rpc_rpc_header_proto_enumTypes[0]
}

func (x ErrorStatusPB_RpcErrorCodePB) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Do not use.
func (x *ErrorStatusPB_RpcErrorCodePB) UnmarshalJSON(b []byte) error {
	num, err := protoimpl.X.UnmarshalJSONEnum(x.Descriptor(), b)
	if err != nil {
		return err
	}
	*x = ErrorStatusPB_RpcErrorCodePB(num)
	return nil
}

// Deprecated: Use ErrorStatusPB_RpcErrorCodePB.Descriptor instead.
func (ErrorStatusPB_RpcErrorCodePB) EnumDescriptor() ([]byte, []int) {
	return file_yb_rpc_rpc_header_proto_rawDescGZIP(), []int{4, 0}
}

type RemoteMethodPB struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Service name for the RPC layer.
	// The client created a proxy with this service name.
	// Example: yb.rpc_test.CalculatorService
	ServiceName *string `protobuf:"bytes,1,req,name=service_name,json=serviceName" json:"service_name,omitempty"`
	// Name of the RPC method.
	MethodName *string `protobuf:"bytes,2,req,name=method_name,json=methodName" json:"method_name,omitempty"`
}

func (x *RemoteMethodPB) Reset() {
	*x = RemoteMethodPB{}
	if protoimpl.UnsafeEnabled {
		mi := &file_yb_rpc_rpc_header_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RemoteMethodPB) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RemoteMethodPB) ProtoMessage() {}

func (x *RemoteMethodPB) ProtoReflect() protoreflect.Message {
	mi := &file_yb_rpc_rpc_header_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RemoteMethodPB.ProtoReflect.Descriptor instead.
func (*RemoteMethodPB) Descriptor() ([]byte, []int) {
	return file_yb_rpc_rpc_header_proto_rawDescGZIP(), []int{0}
}

func (x *RemoteMethodPB) GetServiceName() string {
	if x != nil && x.ServiceName != nil {
		return *x.ServiceName
	}
	return ""
}

func (x *RemoteMethodPB) GetMethodName() string {
	if x != nil && x.MethodName != nil {
		return *x.MethodName
	}
	return ""
}

// The header for the RPC request frame.
type RequestHeader struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// A sequence number that is sent back in the Response. Hadoop specifies a uint32 and
	// casts it to a signed int. That is counterintuitive, so we use an int32 instead.
	// Allowed values (inherited from Hadoop):
	//   0 through INT32_MAX: Regular RPC call IDs.
	//   -2: Invalid call ID.
	//   -3: Connection context call ID.
	//   -33: SASL negotiation call ID.
	CallId *int32 `protobuf:"varint,1,req,name=call_id,json=callId" json:"call_id,omitempty"`
	// RPC method being invoked.
	// Not used for "connection setup" calls.
	RemoteMethod *RemoteMethodPB `protobuf:"bytes,2,opt,name=remote_method,json=remoteMethod" json:"remote_method,omitempty"`
	// Propagate the timeout as specified by the user. Note that, since there is some
	// transit time between the client and server, if you wait exactly this amount of
	// time and then respond, you are likely to cause a timeout on the client.
	TimeoutMillis *uint32 `protobuf:"varint,3,opt,name=timeout_millis,json=timeoutMillis" json:"timeout_millis,omitempty"`
}

func (x *RequestHeader) Reset() {
	*x = RequestHeader{}
	if protoimpl.UnsafeEnabled {
		mi := &file_yb_rpc_rpc_header_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RequestHeader) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RequestHeader) ProtoMessage() {}

func (x *RequestHeader) ProtoReflect() protoreflect.Message {
	mi := &file_yb_rpc_rpc_header_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RequestHeader.ProtoReflect.Descriptor instead.
func (*RequestHeader) Descriptor() ([]byte, []int) {
	return file_yb_rpc_rpc_header_proto_rawDescGZIP(), []int{1}
}

func (x *RequestHeader) GetCallId() int32 {
	if x != nil && x.CallId != nil {
		return *x.CallId
	}
	return 0
}

func (x *RequestHeader) GetRemoteMethod() *RemoteMethodPB {
	if x != nil {
		return x.RemoteMethod
	}
	return nil
}

func (x *RequestHeader) GetTimeoutMillis() uint32 {
	if x != nil && x.TimeoutMillis != nil {
		return *x.TimeoutMillis
	}
	return 0
}

type ResponseHeader struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	CallId *int32 `protobuf:"varint,1,req,name=call_id,json=callId" json:"call_id,omitempty"`
	// If this is set, then this is an error response and the
	// response message will be of type ErrorStatusPB instead of
	// the expected response type.
	IsError *bool `protobuf:"varint,2,opt,name=is_error,json=isError,def=0" json:"is_error,omitempty"`
	// Byte offsets for side cars in the main body of the response message.
	// These offsets are counted AFTER the message header, i.e., offset 0
	// is the first byte after the bytes for this protobuf.
	SidecarOffsets []uint32 `protobuf:"varint,3,rep,name=sidecar_offsets,json=sidecarOffsets" json:"sidecar_offsets,omitempty"`
}

// Default values for ResponseHeader fields.
const (
	Default_ResponseHeader_IsError = bool(false)
)

func (x *ResponseHeader) Reset() {
	*x = ResponseHeader{}
	if protoimpl.UnsafeEnabled {
		mi := &file_yb_rpc_rpc_header_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ResponseHeader) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ResponseHeader) ProtoMessage() {}

func (x *ResponseHeader) ProtoReflect() protoreflect.Message {
	mi := &file_yb_rpc_rpc_header_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ResponseHeader.ProtoReflect.Descriptor instead.
func (*ResponseHeader) Descriptor() ([]byte, []int) {
	return file_yb_rpc_rpc_header_proto_rawDescGZIP(), []int{2}
}

func (x *ResponseHeader) GetCallId() int32 {
	if x != nil && x.CallId != nil {
		return *x.CallId
	}
	return 0
}

func (x *ResponseHeader) GetIsError() bool {
	if x != nil && x.IsError != nil {
		return *x.IsError
	}
	return Default_ResponseHeader_IsError
}

func (x *ResponseHeader) GetSidecarOffsets() []uint32 {
	if x != nil {
		return x.SidecarOffsets
	}
	return nil
}

// An emtpy message. Since CQL RPC server bypasses protobuf to handle requests and responses but
// RPC context expects the request and response messages to be present, we set up an empty message
// inside RPC context to satisfy it.
type EmptyMessagePB struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *EmptyMessagePB) Reset() {
	*x = EmptyMessagePB{}
	if protoimpl.UnsafeEnabled {
		mi := &file_yb_rpc_rpc_header_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *EmptyMessagePB) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EmptyMessagePB) ProtoMessage() {}

func (x *EmptyMessagePB) ProtoReflect() protoreflect.Message {
	mi := &file_yb_rpc_rpc_header_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EmptyMessagePB.ProtoReflect.Descriptor instead.
func (*EmptyMessagePB) Descriptor() ([]byte, []int) {
	return file_yb_rpc_rpc_header_proto_rawDescGZIP(), []int{3}
}

// Sent as response when is_error == true.
type ErrorStatusPB struct {
	state           protoimpl.MessageState
	sizeCache       protoimpl.SizeCache
	unknownFields   protoimpl.UnknownFields
	extensionFields protoimpl.ExtensionFields

	Message *string `protobuf:"bytes,1,req,name=message" json:"message,omitempty"`
	// TODO: Make code required?
	Code *ErrorStatusPB_RpcErrorCodePB `protobuf:"varint,2,opt,name=code,enum=yb.rpc.ErrorStatusPB_RpcErrorCodePB" json:"code,omitempty"` // Specific error identifier.
}

func (x *ErrorStatusPB) Reset() {
	*x = ErrorStatusPB{}
	if protoimpl.UnsafeEnabled {
		mi := &file_yb_rpc_rpc_header_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ErrorStatusPB) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ErrorStatusPB) ProtoMessage() {}

func (x *ErrorStatusPB) ProtoReflect() protoreflect.Message {
	mi := &file_yb_rpc_rpc_header_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ErrorStatusPB.ProtoReflect.Descriptor instead.
func (*ErrorStatusPB) Descriptor() ([]byte, []int) {
	return file_yb_rpc_rpc_header_proto_rawDescGZIP(), []int{4}
}

var extRange_ErrorStatusPB = []protoiface.ExtensionRangeV1{
	{Start: 100, End: 536870911},
}

// Deprecated: Use ErrorStatusPB.ProtoReflect.Descriptor.ExtensionRanges instead.
func (*ErrorStatusPB) ExtensionRangeArray() []protoiface.ExtensionRangeV1 {
	return extRange_ErrorStatusPB
}

func (x *ErrorStatusPB) GetMessage() string {
	if x != nil && x.Message != nil {
		return *x.Message
	}
	return ""
}

func (x *ErrorStatusPB) GetCode() ErrorStatusPB_RpcErrorCodePB {
	if x != nil && x.Code != nil {
		return *x.Code
	}
	return ErrorStatusPB_FATAL_UNKNOWN
}

var File_yb_rpc_rpc_header_proto protoreflect.FileDescriptor

var file_yb_rpc_rpc_header_proto_rawDesc = []byte{
	0x0a, 0x17, 0x79, 0x62, 0x2f, 0x72, 0x70, 0x63, 0x2f, 0x72, 0x70, 0x63, 0x5f, 0x68, 0x65, 0x61,
	0x64, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x06, 0x79, 0x62, 0x2e, 0x72, 0x70,
	0x63, 0x22, 0x54, 0x0a, 0x0e, 0x52, 0x65, 0x6d, 0x6f, 0x74, 0x65, 0x4d, 0x65, 0x74, 0x68, 0x6f,
	0x64, 0x50, 0x42, 0x12, 0x21, 0x0a, 0x0c, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x5f, 0x6e,
	0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x02, 0x28, 0x09, 0x52, 0x0b, 0x73, 0x65, 0x72, 0x76, 0x69,
	0x63, 0x65, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x1f, 0x0a, 0x0b, 0x6d, 0x65, 0x74, 0x68, 0x6f, 0x64,
	0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x02, 0x28, 0x09, 0x52, 0x0a, 0x6d, 0x65, 0x74,
	0x68, 0x6f, 0x64, 0x4e, 0x61, 0x6d, 0x65, 0x22, 0x8c, 0x01, 0x0a, 0x0d, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x12, 0x17, 0x0a, 0x07, 0x63, 0x61, 0x6c,
	0x6c, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x02, 0x28, 0x05, 0x52, 0x06, 0x63, 0x61, 0x6c, 0x6c,
	0x49, 0x64, 0x12, 0x3b, 0x0a, 0x0d, 0x72, 0x65, 0x6d, 0x6f, 0x74, 0x65, 0x5f, 0x6d, 0x65, 0x74,
	0x68, 0x6f, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x16, 0x2e, 0x79, 0x62, 0x2e, 0x72,
	0x70, 0x63, 0x2e, 0x52, 0x65, 0x6d, 0x6f, 0x74, 0x65, 0x4d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x50,
	0x42, 0x52, 0x0c, 0x72, 0x65, 0x6d, 0x6f, 0x74, 0x65, 0x4d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x12,
	0x25, 0x0a, 0x0e, 0x74, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74, 0x5f, 0x6d, 0x69, 0x6c, 0x6c, 0x69,
	0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x0d, 0x74, 0x69, 0x6d, 0x65, 0x6f, 0x75, 0x74,
	0x4d, 0x69, 0x6c, 0x6c, 0x69, 0x73, 0x22, 0x74, 0x0a, 0x0e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x12, 0x17, 0x0a, 0x07, 0x63, 0x61, 0x6c, 0x6c,
	0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x02, 0x28, 0x05, 0x52, 0x06, 0x63, 0x61, 0x6c, 0x6c, 0x49,
	0x64, 0x12, 0x20, 0x0a, 0x08, 0x69, 0x73, 0x5f, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x08, 0x3a, 0x05, 0x66, 0x61, 0x6c, 0x73, 0x65, 0x52, 0x07, 0x69, 0x73, 0x45, 0x72,
	0x72, 0x6f, 0x72, 0x12, 0x27, 0x0a, 0x0f, 0x73, 0x69, 0x64, 0x65, 0x63, 0x61, 0x72, 0x5f, 0x6f,
	0x66, 0x66, 0x73, 0x65, 0x74, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x0d, 0x52, 0x0e, 0x73, 0x69,
	0x64, 0x65, 0x63, 0x61, 0x72, 0x4f, 0x66, 0x66, 0x73, 0x65, 0x74, 0x73, 0x22, 0x10, 0x0a, 0x0e,
	0x45, 0x6d, 0x70, 0x74, 0x79, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x50, 0x42, 0x22, 0x8a,
	0x03, 0x0a, 0x0d, 0x45, 0x72, 0x72, 0x6f, 0x72, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x50, 0x42,
	0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x01, 0x20, 0x02, 0x28,
	0x09, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x38, 0x0a, 0x04, 0x63, 0x6f,
	0x64, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x24, 0x2e, 0x79, 0x62, 0x2e, 0x72, 0x70,
	0x63, 0x2e, 0x45, 0x72, 0x72, 0x6f, 0x72, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x50, 0x42, 0x2e,
	0x52, 0x70, 0x63, 0x45, 0x72, 0x72, 0x6f, 0x72, 0x43, 0x6f, 0x64, 0x65, 0x50, 0x42, 0x52, 0x04,
	0x63, 0x6f, 0x64, 0x65, 0x22, 0x9a, 0x02, 0x0a, 0x0e, 0x52, 0x70, 0x63, 0x45, 0x72, 0x72, 0x6f,
	0x72, 0x43, 0x6f, 0x64, 0x65, 0x50, 0x42, 0x12, 0x11, 0x0a, 0x0d, 0x46, 0x41, 0x54, 0x41, 0x4c,
	0x5f, 0x55, 0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e, 0x10, 0x0a, 0x12, 0x15, 0x0a, 0x11, 0x45, 0x52,
	0x52, 0x4f, 0x52, 0x5f, 0x41, 0x50, 0x50, 0x4c, 0x49, 0x43, 0x41, 0x54, 0x49, 0x4f, 0x4e, 0x10,
	0x01, 0x12, 0x18, 0x0a, 0x14, 0x45, 0x52, 0x52, 0x4f, 0x52, 0x5f, 0x4e, 0x4f, 0x5f, 0x53, 0x55,
	0x43, 0x48, 0x5f, 0x4d, 0x45, 0x54, 0x48, 0x4f, 0x44, 0x10, 0x02, 0x12, 0x19, 0x0a, 0x15, 0x45,
	0x52, 0x52, 0x4f, 0x52, 0x5f, 0x4e, 0x4f, 0x5f, 0x53, 0x55, 0x43, 0x48, 0x5f, 0x53, 0x45, 0x52,
	0x56, 0x49, 0x43, 0x45, 0x10, 0x03, 0x12, 0x19, 0x0a, 0x15, 0x45, 0x52, 0x52, 0x4f, 0x52, 0x5f,
	0x53, 0x45, 0x52, 0x56, 0x45, 0x52, 0x5f, 0x54, 0x4f, 0x4f, 0x5f, 0x42, 0x55, 0x53, 0x59, 0x10,
	0x04, 0x12, 0x19, 0x0a, 0x15, 0x45, 0x52, 0x52, 0x4f, 0x52, 0x5f, 0x49, 0x4e, 0x56, 0x41, 0x4c,
	0x49, 0x44, 0x5f, 0x52, 0x45, 0x51, 0x55, 0x45, 0x53, 0x54, 0x10, 0x05, 0x12, 0x1e, 0x0a, 0x1a,
	0x46, 0x41, 0x54, 0x41, 0x4c, 0x5f, 0x53, 0x45, 0x52, 0x56, 0x45, 0x52, 0x5f, 0x53, 0x48, 0x55,
	0x54, 0x54, 0x49, 0x4e, 0x47, 0x5f, 0x44, 0x4f, 0x57, 0x4e, 0x10, 0x0b, 0x12, 0x1f, 0x0a, 0x1b,
	0x46, 0x41, 0x54, 0x41, 0x4c, 0x5f, 0x44, 0x45, 0x53, 0x45, 0x52, 0x49, 0x41, 0x4c, 0x49, 0x5a,
	0x49, 0x4e, 0x47, 0x5f, 0x52, 0x45, 0x51, 0x55, 0x45, 0x53, 0x54, 0x10, 0x0d, 0x12, 0x1a, 0x0a,
	0x16, 0x46, 0x41, 0x54, 0x41, 0x4c, 0x5f, 0x56, 0x45, 0x52, 0x53, 0x49, 0x4f, 0x4e, 0x5f, 0x4d,
	0x49, 0x53, 0x4d, 0x41, 0x54, 0x43, 0x48, 0x10, 0x0e, 0x12, 0x16, 0x0a, 0x12, 0x46, 0x41, 0x54,
	0x41, 0x4c, 0x5f, 0x55, 0x4e, 0x41, 0x55, 0x54, 0x48, 0x4f, 0x52, 0x49, 0x5a, 0x45, 0x44, 0x10,
	0x0f, 0x2a, 0x08, 0x08, 0x64, 0x10, 0x80, 0x80, 0x80, 0x80, 0x02, 0x42, 0x0c, 0x0a, 0x0a, 0x6f,
	0x72, 0x67, 0x2e, 0x79, 0x62, 0x2e, 0x72, 0x70, 0x63,
}

var (
	file_yb_rpc_rpc_header_proto_rawDescOnce sync.Once
	file_yb_rpc_rpc_header_proto_rawDescData = file_yb_rpc_rpc_header_proto_rawDesc
)

func file_yb_rpc_rpc_header_proto_rawDescGZIP() []byte {
	file_yb_rpc_rpc_header_proto_rawDescOnce.Do(func() {
		file_yb_rpc_rpc_header_proto_rawDescData = protoimpl.X.CompressGZIP(file_yb_rpc_rpc_header_proto_rawDescData)
	})
	return file_yb_rpc_rpc_header_proto_rawDescData
}

var file_yb_rpc_rpc_header_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_yb_rpc_rpc_header_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_yb_rpc_rpc_header_proto_goTypes = []interface{}{
	(ErrorStatusPB_RpcErrorCodePB)(0), // 0: yb.rpc.ErrorStatusPB.RpcErrorCodePB
	(*RemoteMethodPB)(nil),            // 1: yb.rpc.RemoteMethodPB
	(*RequestHeader)(nil),             // 2: yb.rpc.RequestHeader
	(*ResponseHeader)(nil),            // 3: yb.rpc.ResponseHeader
	(*EmptyMessagePB)(nil),            // 4: yb.rpc.EmptyMessagePB
	(*ErrorStatusPB)(nil),             // 5: yb.rpc.ErrorStatusPB
}
var file_yb_rpc_rpc_header_proto_depIdxs = []int32{
	1, // 0: yb.rpc.RequestHeader.remote_method:type_name -> yb.rpc.RemoteMethodPB
	0, // 1: yb.rpc.ErrorStatusPB.code:type_name -> yb.rpc.ErrorStatusPB.RpcErrorCodePB
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_yb_rpc_rpc_header_proto_init() }
func file_yb_rpc_rpc_header_proto_init() {
	if File_yb_rpc_rpc_header_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_yb_rpc_rpc_header_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RemoteMethodPB); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_yb_rpc_rpc_header_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RequestHeader); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_yb_rpc_rpc_header_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ResponseHeader); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_yb_rpc_rpc_header_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*EmptyMessagePB); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_yb_rpc_rpc_header_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ErrorStatusPB); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			case 3:
				return &v.extensionFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_yb_rpc_rpc_header_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_yb_rpc_rpc_header_proto_goTypes,
		DependencyIndexes: file_yb_rpc_rpc_header_proto_depIdxs,
		EnumInfos:         file_yb_rpc_rpc_header_proto_enumTypes,
		MessageInfos:      file_yb_rpc_rpc_header_proto_msgTypes,
	}.Build()
	File_yb_rpc_rpc_header_proto = out.File
	file_yb_rpc_rpc_header_proto_rawDesc = nil
	file_yb_rpc_rpc_header_proto_goTypes = nil
	file_yb_rpc_rpc_header_proto_depIdxs = nil
}
