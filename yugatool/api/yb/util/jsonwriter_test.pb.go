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
// source: yb/util/jsonwriter_test.proto

package util

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
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

type TestAllTypes_NestedEnum int32

const (
	TestAllTypes_FOO TestAllTypes_NestedEnum = 1
	TestAllTypes_BAR TestAllTypes_NestedEnum = 2
	TestAllTypes_BAZ TestAllTypes_NestedEnum = 3
)

// Enum value maps for TestAllTypes_NestedEnum.
var (
	TestAllTypes_NestedEnum_name = map[int32]string{
		1: "FOO",
		2: "BAR",
		3: "BAZ",
	}
	TestAllTypes_NestedEnum_value = map[string]int32{
		"FOO": 1,
		"BAR": 2,
		"BAZ": 3,
	}
)

func (x TestAllTypes_NestedEnum) Enum() *TestAllTypes_NestedEnum {
	p := new(TestAllTypes_NestedEnum)
	*p = x
	return p
}

func (x TestAllTypes_NestedEnum) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (TestAllTypes_NestedEnum) Descriptor() protoreflect.EnumDescriptor {
	return file_yb_util_jsonwriter_test_proto_enumTypes[0].Descriptor()
}

func (TestAllTypes_NestedEnum) Type() protoreflect.EnumType {
	return &file_yb_util_jsonwriter_test_proto_enumTypes[0]
}

func (x TestAllTypes_NestedEnum) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Do not use.
func (x *TestAllTypes_NestedEnum) UnmarshalJSON(b []byte) error {
	num, err := protoimpl.X.UnmarshalJSONEnum(x.Descriptor(), b)
	if err != nil {
		return err
	}
	*x = TestAllTypes_NestedEnum(num)
	return nil
}

// Deprecated: Use TestAllTypes_NestedEnum.Descriptor instead.
func (TestAllTypes_NestedEnum) EnumDescriptor() ([]byte, []int) {
	return file_yb_util_jsonwriter_test_proto_rawDescGZIP(), []int{0, 0}
}

// This proto includes every type of field in both singular and repeated
// forms. This is mostly copied from 'unittest.proto' in the protobuf source
// (hence the odd field numbers which skip some).
type TestAllTypes struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Singular
	OptionalInt32         *int32                      `protobuf:"varint,1,opt,name=optional_int32,json=optionalInt32" json:"optional_int32,omitempty"`
	OptionalInt64         *int64                      `protobuf:"varint,2,opt,name=optional_int64,json=optionalInt64" json:"optional_int64,omitempty"`
	OptionalUint32        *uint32                     `protobuf:"varint,3,opt,name=optional_uint32,json=optionalUint32" json:"optional_uint32,omitempty"`
	OptionalUint64        *uint64                     `protobuf:"varint,4,opt,name=optional_uint64,json=optionalUint64" json:"optional_uint64,omitempty"`
	OptionalSint32        *int32                      `protobuf:"zigzag32,5,opt,name=optional_sint32,json=optionalSint32" json:"optional_sint32,omitempty"`
	OptionalSint64        *int64                      `protobuf:"zigzag64,6,opt,name=optional_sint64,json=optionalSint64" json:"optional_sint64,omitempty"`
	OptionalFixed32       *uint32                     `protobuf:"fixed32,7,opt,name=optional_fixed32,json=optionalFixed32" json:"optional_fixed32,omitempty"`
	OptionalFixed64       *uint64                     `protobuf:"fixed64,8,opt,name=optional_fixed64,json=optionalFixed64" json:"optional_fixed64,omitempty"`
	OptionalSfixed32      *int32                      `protobuf:"fixed32,9,opt,name=optional_sfixed32,json=optionalSfixed32" json:"optional_sfixed32,omitempty"`
	OptionalSfixed64      *int64                      `protobuf:"fixed64,10,opt,name=optional_sfixed64,json=optionalSfixed64" json:"optional_sfixed64,omitempty"`
	OptionalFloat         *float32                    `protobuf:"fixed32,11,opt,name=optional_float,json=optionalFloat" json:"optional_float,omitempty"`
	OptionalDouble        *float64                    `protobuf:"fixed64,12,opt,name=optional_double,json=optionalDouble" json:"optional_double,omitempty"`
	OptionalBool          *bool                       `protobuf:"varint,13,opt,name=optional_bool,json=optionalBool" json:"optional_bool,omitempty"`
	OptionalString        *string                     `protobuf:"bytes,14,opt,name=optional_string,json=optionalString" json:"optional_string,omitempty"`
	OptionalBytes         []byte                      `protobuf:"bytes,15,opt,name=optional_bytes,json=optionalBytes" json:"optional_bytes,omitempty"`
	OptionalNestedMessage *TestAllTypes_NestedMessage `protobuf:"bytes,18,opt,name=optional_nested_message,json=optionalNestedMessage" json:"optional_nested_message,omitempty"`
	OptionalNestedEnum    *TestAllTypes_NestedEnum    `protobuf:"varint,21,opt,name=optional_nested_enum,json=optionalNestedEnum,enum=jsonwriter_test.TestAllTypes_NestedEnum" json:"optional_nested_enum,omitempty"`
	// Repeated
	RepeatedInt32         []int32                       `protobuf:"varint,31,rep,name=repeated_int32,json=repeatedInt32" json:"repeated_int32,omitempty"`
	RepeatedInt64         []int64                       `protobuf:"varint,32,rep,name=repeated_int64,json=repeatedInt64" json:"repeated_int64,omitempty"`
	RepeatedUint32        []uint32                      `protobuf:"varint,33,rep,name=repeated_uint32,json=repeatedUint32" json:"repeated_uint32,omitempty"`
	RepeatedUint64        []uint64                      `protobuf:"varint,34,rep,name=repeated_uint64,json=repeatedUint64" json:"repeated_uint64,omitempty"`
	RepeatedSint32        []int32                       `protobuf:"zigzag32,35,rep,name=repeated_sint32,json=repeatedSint32" json:"repeated_sint32,omitempty"`
	RepeatedSint64        []int64                       `protobuf:"zigzag64,36,rep,name=repeated_sint64,json=repeatedSint64" json:"repeated_sint64,omitempty"`
	RepeatedFixed32       []uint32                      `protobuf:"fixed32,37,rep,name=repeated_fixed32,json=repeatedFixed32" json:"repeated_fixed32,omitempty"`
	RepeatedFixed64       []uint64                      `protobuf:"fixed64,38,rep,name=repeated_fixed64,json=repeatedFixed64" json:"repeated_fixed64,omitempty"`
	RepeatedSfixed32      []int32                       `protobuf:"fixed32,39,rep,name=repeated_sfixed32,json=repeatedSfixed32" json:"repeated_sfixed32,omitempty"`
	RepeatedSfixed64      []int64                       `protobuf:"fixed64,40,rep,name=repeated_sfixed64,json=repeatedSfixed64" json:"repeated_sfixed64,omitempty"`
	RepeatedFloat         []float32                     `protobuf:"fixed32,41,rep,name=repeated_float,json=repeatedFloat" json:"repeated_float,omitempty"`
	RepeatedDouble        []float64                     `protobuf:"fixed64,42,rep,name=repeated_double,json=repeatedDouble" json:"repeated_double,omitempty"`
	RepeatedBool          []bool                        `protobuf:"varint,43,rep,name=repeated_bool,json=repeatedBool" json:"repeated_bool,omitempty"`
	RepeatedString        []string                      `protobuf:"bytes,44,rep,name=repeated_string,json=repeatedString" json:"repeated_string,omitempty"`
	RepeatedBytes         [][]byte                      `protobuf:"bytes,45,rep,name=repeated_bytes,json=repeatedBytes" json:"repeated_bytes,omitempty"`
	RepeatedNestedMessage []*TestAllTypes_NestedMessage `protobuf:"bytes,48,rep,name=repeated_nested_message,json=repeatedNestedMessage" json:"repeated_nested_message,omitempty"`
	RepeatedNestedEnum    []TestAllTypes_NestedEnum     `protobuf:"varint,51,rep,name=repeated_nested_enum,json=repeatedNestedEnum,enum=jsonwriter_test.TestAllTypes_NestedEnum" json:"repeated_nested_enum,omitempty"`
}

func (x *TestAllTypes) Reset() {
	*x = TestAllTypes{}
	if protoimpl.UnsafeEnabled {
		mi := &file_yb_util_jsonwriter_test_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TestAllTypes) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestAllTypes) ProtoMessage() {}

func (x *TestAllTypes) ProtoReflect() protoreflect.Message {
	mi := &file_yb_util_jsonwriter_test_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestAllTypes.ProtoReflect.Descriptor instead.
func (*TestAllTypes) Descriptor() ([]byte, []int) {
	return file_yb_util_jsonwriter_test_proto_rawDescGZIP(), []int{0}
}

func (x *TestAllTypes) GetOptionalInt32() int32 {
	if x != nil && x.OptionalInt32 != nil {
		return *x.OptionalInt32
	}
	return 0
}

func (x *TestAllTypes) GetOptionalInt64() int64 {
	if x != nil && x.OptionalInt64 != nil {
		return *x.OptionalInt64
	}
	return 0
}

func (x *TestAllTypes) GetOptionalUint32() uint32 {
	if x != nil && x.OptionalUint32 != nil {
		return *x.OptionalUint32
	}
	return 0
}

func (x *TestAllTypes) GetOptionalUint64() uint64 {
	if x != nil && x.OptionalUint64 != nil {
		return *x.OptionalUint64
	}
	return 0
}

func (x *TestAllTypes) GetOptionalSint32() int32 {
	if x != nil && x.OptionalSint32 != nil {
		return *x.OptionalSint32
	}
	return 0
}

func (x *TestAllTypes) GetOptionalSint64() int64 {
	if x != nil && x.OptionalSint64 != nil {
		return *x.OptionalSint64
	}
	return 0
}

func (x *TestAllTypes) GetOptionalFixed32() uint32 {
	if x != nil && x.OptionalFixed32 != nil {
		return *x.OptionalFixed32
	}
	return 0
}

func (x *TestAllTypes) GetOptionalFixed64() uint64 {
	if x != nil && x.OptionalFixed64 != nil {
		return *x.OptionalFixed64
	}
	return 0
}

func (x *TestAllTypes) GetOptionalSfixed32() int32 {
	if x != nil && x.OptionalSfixed32 != nil {
		return *x.OptionalSfixed32
	}
	return 0
}

func (x *TestAllTypes) GetOptionalSfixed64() int64 {
	if x != nil && x.OptionalSfixed64 != nil {
		return *x.OptionalSfixed64
	}
	return 0
}

func (x *TestAllTypes) GetOptionalFloat() float32 {
	if x != nil && x.OptionalFloat != nil {
		return *x.OptionalFloat
	}
	return 0
}

func (x *TestAllTypes) GetOptionalDouble() float64 {
	if x != nil && x.OptionalDouble != nil {
		return *x.OptionalDouble
	}
	return 0
}

func (x *TestAllTypes) GetOptionalBool() bool {
	if x != nil && x.OptionalBool != nil {
		return *x.OptionalBool
	}
	return false
}

func (x *TestAllTypes) GetOptionalString() string {
	if x != nil && x.OptionalString != nil {
		return *x.OptionalString
	}
	return ""
}

func (x *TestAllTypes) GetOptionalBytes() []byte {
	if x != nil {
		return x.OptionalBytes
	}
	return nil
}

func (x *TestAllTypes) GetOptionalNestedMessage() *TestAllTypes_NestedMessage {
	if x != nil {
		return x.OptionalNestedMessage
	}
	return nil
}

func (x *TestAllTypes) GetOptionalNestedEnum() TestAllTypes_NestedEnum {
	if x != nil && x.OptionalNestedEnum != nil {
		return *x.OptionalNestedEnum
	}
	return TestAllTypes_FOO
}

func (x *TestAllTypes) GetRepeatedInt32() []int32 {
	if x != nil {
		return x.RepeatedInt32
	}
	return nil
}

func (x *TestAllTypes) GetRepeatedInt64() []int64 {
	if x != nil {
		return x.RepeatedInt64
	}
	return nil
}

func (x *TestAllTypes) GetRepeatedUint32() []uint32 {
	if x != nil {
		return x.RepeatedUint32
	}
	return nil
}

func (x *TestAllTypes) GetRepeatedUint64() []uint64 {
	if x != nil {
		return x.RepeatedUint64
	}
	return nil
}

func (x *TestAllTypes) GetRepeatedSint32() []int32 {
	if x != nil {
		return x.RepeatedSint32
	}
	return nil
}

func (x *TestAllTypes) GetRepeatedSint64() []int64 {
	if x != nil {
		return x.RepeatedSint64
	}
	return nil
}

func (x *TestAllTypes) GetRepeatedFixed32() []uint32 {
	if x != nil {
		return x.RepeatedFixed32
	}
	return nil
}

func (x *TestAllTypes) GetRepeatedFixed64() []uint64 {
	if x != nil {
		return x.RepeatedFixed64
	}
	return nil
}

func (x *TestAllTypes) GetRepeatedSfixed32() []int32 {
	if x != nil {
		return x.RepeatedSfixed32
	}
	return nil
}

func (x *TestAllTypes) GetRepeatedSfixed64() []int64 {
	if x != nil {
		return x.RepeatedSfixed64
	}
	return nil
}

func (x *TestAllTypes) GetRepeatedFloat() []float32 {
	if x != nil {
		return x.RepeatedFloat
	}
	return nil
}

func (x *TestAllTypes) GetRepeatedDouble() []float64 {
	if x != nil {
		return x.RepeatedDouble
	}
	return nil
}

func (x *TestAllTypes) GetRepeatedBool() []bool {
	if x != nil {
		return x.RepeatedBool
	}
	return nil
}

func (x *TestAllTypes) GetRepeatedString() []string {
	if x != nil {
		return x.RepeatedString
	}
	return nil
}

func (x *TestAllTypes) GetRepeatedBytes() [][]byte {
	if x != nil {
		return x.RepeatedBytes
	}
	return nil
}

func (x *TestAllTypes) GetRepeatedNestedMessage() []*TestAllTypes_NestedMessage {
	if x != nil {
		return x.RepeatedNestedMessage
	}
	return nil
}

func (x *TestAllTypes) GetRepeatedNestedEnum() []TestAllTypes_NestedEnum {
	if x != nil {
		return x.RepeatedNestedEnum
	}
	return nil
}

type TestAllTypes_NestedMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	IntField *int32 `protobuf:"varint,1,opt,name=int_field,json=intField" json:"int_field,omitempty"`
}

func (x *TestAllTypes_NestedMessage) Reset() {
	*x = TestAllTypes_NestedMessage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_yb_util_jsonwriter_test_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TestAllTypes_NestedMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestAllTypes_NestedMessage) ProtoMessage() {}

func (x *TestAllTypes_NestedMessage) ProtoReflect() protoreflect.Message {
	mi := &file_yb_util_jsonwriter_test_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestAllTypes_NestedMessage.ProtoReflect.Descriptor instead.
func (*TestAllTypes_NestedMessage) Descriptor() ([]byte, []int) {
	return file_yb_util_jsonwriter_test_proto_rawDescGZIP(), []int{0, 0}
}

func (x *TestAllTypes_NestedMessage) GetIntField() int32 {
	if x != nil && x.IntField != nil {
		return *x.IntField
	}
	return 0
}

var File_yb_util_jsonwriter_test_proto protoreflect.FileDescriptor

var file_yb_util_jsonwriter_test_proto_rawDesc = []byte{
	0x0a, 0x1d, 0x79, 0x62, 0x2f, 0x75, 0x74, 0x69, 0x6c, 0x2f, 0x6a, 0x73, 0x6f, 0x6e, 0x77, 0x72,
	0x69, 0x74, 0x65, 0x72, 0x5f, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x0f, 0x6a, 0x73, 0x6f, 0x6e, 0x77, 0x72, 0x69, 0x74, 0x65, 0x72, 0x5f, 0x74, 0x65, 0x73, 0x74,
	0x22, 0xb5, 0x0d, 0x0a, 0x0c, 0x54, 0x65, 0x73, 0x74, 0x41, 0x6c, 0x6c, 0x54, 0x79, 0x70, 0x65,
	0x73, 0x12, 0x25, 0x0a, 0x0e, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x5f, 0x69, 0x6e,
	0x74, 0x33, 0x32, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0d, 0x6f, 0x70, 0x74, 0x69, 0x6f,
	0x6e, 0x61, 0x6c, 0x49, 0x6e, 0x74, 0x33, 0x32, 0x12, 0x25, 0x0a, 0x0e, 0x6f, 0x70, 0x74, 0x69,
	0x6f, 0x6e, 0x61, 0x6c, 0x5f, 0x69, 0x6e, 0x74, 0x36, 0x34, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03,
	0x52, 0x0d, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x49, 0x6e, 0x74, 0x36, 0x34, 0x12,
	0x27, 0x0a, 0x0f, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x5f, 0x75, 0x69, 0x6e, 0x74,
	0x33, 0x32, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x0e, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e,
	0x61, 0x6c, 0x55, 0x69, 0x6e, 0x74, 0x33, 0x32, 0x12, 0x27, 0x0a, 0x0f, 0x6f, 0x70, 0x74, 0x69,
	0x6f, 0x6e, 0x61, 0x6c, 0x5f, 0x75, 0x69, 0x6e, 0x74, 0x36, 0x34, 0x18, 0x04, 0x20, 0x01, 0x28,
	0x04, 0x52, 0x0e, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x55, 0x69, 0x6e, 0x74, 0x36,
	0x34, 0x12, 0x27, 0x0a, 0x0f, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x5f, 0x73, 0x69,
	0x6e, 0x74, 0x33, 0x32, 0x18, 0x05, 0x20, 0x01, 0x28, 0x11, 0x52, 0x0e, 0x6f, 0x70, 0x74, 0x69,
	0x6f, 0x6e, 0x61, 0x6c, 0x53, 0x69, 0x6e, 0x74, 0x33, 0x32, 0x12, 0x27, 0x0a, 0x0f, 0x6f, 0x70,
	0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x5f, 0x73, 0x69, 0x6e, 0x74, 0x36, 0x34, 0x18, 0x06, 0x20,
	0x01, 0x28, 0x12, 0x52, 0x0e, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x53, 0x69, 0x6e,
	0x74, 0x36, 0x34, 0x12, 0x29, 0x0a, 0x10, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x5f,
	0x66, 0x69, 0x78, 0x65, 0x64, 0x33, 0x32, 0x18, 0x07, 0x20, 0x01, 0x28, 0x07, 0x52, 0x0f, 0x6f,
	0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x46, 0x69, 0x78, 0x65, 0x64, 0x33, 0x32, 0x12, 0x29,
	0x0a, 0x10, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x5f, 0x66, 0x69, 0x78, 0x65, 0x64,
	0x36, 0x34, 0x18, 0x08, 0x20, 0x01, 0x28, 0x06, 0x52, 0x0f, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e,
	0x61, 0x6c, 0x46, 0x69, 0x78, 0x65, 0x64, 0x36, 0x34, 0x12, 0x2b, 0x0a, 0x11, 0x6f, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x5f, 0x73, 0x66, 0x69, 0x78, 0x65, 0x64, 0x33, 0x32, 0x18, 0x09,
	0x20, 0x01, 0x28, 0x0f, 0x52, 0x10, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x53, 0x66,
	0x69, 0x78, 0x65, 0x64, 0x33, 0x32, 0x12, 0x2b, 0x0a, 0x11, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e,
	0x61, 0x6c, 0x5f, 0x73, 0x66, 0x69, 0x78, 0x65, 0x64, 0x36, 0x34, 0x18, 0x0a, 0x20, 0x01, 0x28,
	0x10, 0x52, 0x10, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x53, 0x66, 0x69, 0x78, 0x65,
	0x64, 0x36, 0x34, 0x12, 0x25, 0x0a, 0x0e, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x5f,
	0x66, 0x6c, 0x6f, 0x61, 0x74, 0x18, 0x0b, 0x20, 0x01, 0x28, 0x02, 0x52, 0x0d, 0x6f, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x46, 0x6c, 0x6f, 0x61, 0x74, 0x12, 0x27, 0x0a, 0x0f, 0x6f, 0x70,
	0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x5f, 0x64, 0x6f, 0x75, 0x62, 0x6c, 0x65, 0x18, 0x0c, 0x20,
	0x01, 0x28, 0x01, 0x52, 0x0e, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x44, 0x6f, 0x75,
	0x62, 0x6c, 0x65, 0x12, 0x23, 0x0a, 0x0d, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x5f,
	0x62, 0x6f, 0x6f, 0x6c, 0x18, 0x0d, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0c, 0x6f, 0x70, 0x74, 0x69,
	0x6f, 0x6e, 0x61, 0x6c, 0x42, 0x6f, 0x6f, 0x6c, 0x12, 0x27, 0x0a, 0x0f, 0x6f, 0x70, 0x74, 0x69,
	0x6f, 0x6e, 0x61, 0x6c, 0x5f, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x18, 0x0e, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x0e, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x53, 0x74, 0x72, 0x69, 0x6e,
	0x67, 0x12, 0x25, 0x0a, 0x0e, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x5f, 0x62, 0x79,
	0x74, 0x65, 0x73, 0x18, 0x0f, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0d, 0x6f, 0x70, 0x74, 0x69, 0x6f,
	0x6e, 0x61, 0x6c, 0x42, 0x79, 0x74, 0x65, 0x73, 0x12, 0x63, 0x0a, 0x17, 0x6f, 0x70, 0x74, 0x69,
	0x6f, 0x6e, 0x61, 0x6c, 0x5f, 0x6e, 0x65, 0x73, 0x74, 0x65, 0x64, 0x5f, 0x6d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x18, 0x12, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2b, 0x2e, 0x6a, 0x73, 0x6f, 0x6e,
	0x77, 0x72, 0x69, 0x74, 0x65, 0x72, 0x5f, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x54, 0x65, 0x73, 0x74,
	0x41, 0x6c, 0x6c, 0x54, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x4e, 0x65, 0x73, 0x74, 0x65, 0x64, 0x4d,
	0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x52, 0x15, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c,
	0x4e, 0x65, 0x73, 0x74, 0x65, 0x64, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x5a, 0x0a,
	0x14, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x5f, 0x6e, 0x65, 0x73, 0x74, 0x65, 0x64,
	0x5f, 0x65, 0x6e, 0x75, 0x6d, 0x18, 0x15, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x28, 0x2e, 0x6a, 0x73,
	0x6f, 0x6e, 0x77, 0x72, 0x69, 0x74, 0x65, 0x72, 0x5f, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x54, 0x65,
	0x73, 0x74, 0x41, 0x6c, 0x6c, 0x54, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x4e, 0x65, 0x73, 0x74, 0x65,
	0x64, 0x45, 0x6e, 0x75, 0x6d, 0x52, 0x12, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x61, 0x6c, 0x4e,
	0x65, 0x73, 0x74, 0x65, 0x64, 0x45, 0x6e, 0x75, 0x6d, 0x12, 0x25, 0x0a, 0x0e, 0x72, 0x65, 0x70,
	0x65, 0x61, 0x74, 0x65, 0x64, 0x5f, 0x69, 0x6e, 0x74, 0x33, 0x32, 0x18, 0x1f, 0x20, 0x03, 0x28,
	0x05, 0x52, 0x0d, 0x72, 0x65, 0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x49, 0x6e, 0x74, 0x33, 0x32,
	0x12, 0x25, 0x0a, 0x0e, 0x72, 0x65, 0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x5f, 0x69, 0x6e, 0x74,
	0x36, 0x34, 0x18, 0x20, 0x20, 0x03, 0x28, 0x03, 0x52, 0x0d, 0x72, 0x65, 0x70, 0x65, 0x61, 0x74,
	0x65, 0x64, 0x49, 0x6e, 0x74, 0x36, 0x34, 0x12, 0x27, 0x0a, 0x0f, 0x72, 0x65, 0x70, 0x65, 0x61,
	0x74, 0x65, 0x64, 0x5f, 0x75, 0x69, 0x6e, 0x74, 0x33, 0x32, 0x18, 0x21, 0x20, 0x03, 0x28, 0x0d,
	0x52, 0x0e, 0x72, 0x65, 0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x55, 0x69, 0x6e, 0x74, 0x33, 0x32,
	0x12, 0x27, 0x0a, 0x0f, 0x72, 0x65, 0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x5f, 0x75, 0x69, 0x6e,
	0x74, 0x36, 0x34, 0x18, 0x22, 0x20, 0x03, 0x28, 0x04, 0x52, 0x0e, 0x72, 0x65, 0x70, 0x65, 0x61,
	0x74, 0x65, 0x64, 0x55, 0x69, 0x6e, 0x74, 0x36, 0x34, 0x12, 0x27, 0x0a, 0x0f, 0x72, 0x65, 0x70,
	0x65, 0x61, 0x74, 0x65, 0x64, 0x5f, 0x73, 0x69, 0x6e, 0x74, 0x33, 0x32, 0x18, 0x23, 0x20, 0x03,
	0x28, 0x11, 0x52, 0x0e, 0x72, 0x65, 0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x53, 0x69, 0x6e, 0x74,
	0x33, 0x32, 0x12, 0x27, 0x0a, 0x0f, 0x72, 0x65, 0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x5f, 0x73,
	0x69, 0x6e, 0x74, 0x36, 0x34, 0x18, 0x24, 0x20, 0x03, 0x28, 0x12, 0x52, 0x0e, 0x72, 0x65, 0x70,
	0x65, 0x61, 0x74, 0x65, 0x64, 0x53, 0x69, 0x6e, 0x74, 0x36, 0x34, 0x12, 0x29, 0x0a, 0x10, 0x72,
	0x65, 0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x5f, 0x66, 0x69, 0x78, 0x65, 0x64, 0x33, 0x32, 0x18,
	0x25, 0x20, 0x03, 0x28, 0x07, 0x52, 0x0f, 0x72, 0x65, 0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x46,
	0x69, 0x78, 0x65, 0x64, 0x33, 0x32, 0x12, 0x29, 0x0a, 0x10, 0x72, 0x65, 0x70, 0x65, 0x61, 0x74,
	0x65, 0x64, 0x5f, 0x66, 0x69, 0x78, 0x65, 0x64, 0x36, 0x34, 0x18, 0x26, 0x20, 0x03, 0x28, 0x06,
	0x52, 0x0f, 0x72, 0x65, 0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x46, 0x69, 0x78, 0x65, 0x64, 0x36,
	0x34, 0x12, 0x2b, 0x0a, 0x11, 0x72, 0x65, 0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x5f, 0x73, 0x66,
	0x69, 0x78, 0x65, 0x64, 0x33, 0x32, 0x18, 0x27, 0x20, 0x03, 0x28, 0x0f, 0x52, 0x10, 0x72, 0x65,
	0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x53, 0x66, 0x69, 0x78, 0x65, 0x64, 0x33, 0x32, 0x12, 0x2b,
	0x0a, 0x11, 0x72, 0x65, 0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x5f, 0x73, 0x66, 0x69, 0x78, 0x65,
	0x64, 0x36, 0x34, 0x18, 0x28, 0x20, 0x03, 0x28, 0x10, 0x52, 0x10, 0x72, 0x65, 0x70, 0x65, 0x61,
	0x74, 0x65, 0x64, 0x53, 0x66, 0x69, 0x78, 0x65, 0x64, 0x36, 0x34, 0x12, 0x25, 0x0a, 0x0e, 0x72,
	0x65, 0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x5f, 0x66, 0x6c, 0x6f, 0x61, 0x74, 0x18, 0x29, 0x20,
	0x03, 0x28, 0x02, 0x52, 0x0d, 0x72, 0x65, 0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x46, 0x6c, 0x6f,
	0x61, 0x74, 0x12, 0x27, 0x0a, 0x0f, 0x72, 0x65, 0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x5f, 0x64,
	0x6f, 0x75, 0x62, 0x6c, 0x65, 0x18, 0x2a, 0x20, 0x03, 0x28, 0x01, 0x52, 0x0e, 0x72, 0x65, 0x70,
	0x65, 0x61, 0x74, 0x65, 0x64, 0x44, 0x6f, 0x75, 0x62, 0x6c, 0x65, 0x12, 0x23, 0x0a, 0x0d, 0x72,
	0x65, 0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x5f, 0x62, 0x6f, 0x6f, 0x6c, 0x18, 0x2b, 0x20, 0x03,
	0x28, 0x08, 0x52, 0x0c, 0x72, 0x65, 0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x42, 0x6f, 0x6f, 0x6c,
	0x12, 0x27, 0x0a, 0x0f, 0x72, 0x65, 0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x5f, 0x73, 0x74, 0x72,
	0x69, 0x6e, 0x67, 0x18, 0x2c, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0e, 0x72, 0x65, 0x70, 0x65, 0x61,
	0x74, 0x65, 0x64, 0x53, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x12, 0x25, 0x0a, 0x0e, 0x72, 0x65, 0x70,
	0x65, 0x61, 0x74, 0x65, 0x64, 0x5f, 0x62, 0x79, 0x74, 0x65, 0x73, 0x18, 0x2d, 0x20, 0x03, 0x28,
	0x0c, 0x52, 0x0d, 0x72, 0x65, 0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x42, 0x79, 0x74, 0x65, 0x73,
	0x12, 0x63, 0x0a, 0x17, 0x72, 0x65, 0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x5f, 0x6e, 0x65, 0x73,
	0x74, 0x65, 0x64, 0x5f, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x30, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x2b, 0x2e, 0x6a, 0x73, 0x6f, 0x6e, 0x77, 0x72, 0x69, 0x74, 0x65, 0x72, 0x5f, 0x74,
	0x65, 0x73, 0x74, 0x2e, 0x54, 0x65, 0x73, 0x74, 0x41, 0x6c, 0x6c, 0x54, 0x79, 0x70, 0x65, 0x73,
	0x2e, 0x4e, 0x65, 0x73, 0x74, 0x65, 0x64, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x52, 0x15,
	0x72, 0x65, 0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x4e, 0x65, 0x73, 0x74, 0x65, 0x64, 0x4d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x5a, 0x0a, 0x14, 0x72, 0x65, 0x70, 0x65, 0x61, 0x74, 0x65,
	0x64, 0x5f, 0x6e, 0x65, 0x73, 0x74, 0x65, 0x64, 0x5f, 0x65, 0x6e, 0x75, 0x6d, 0x18, 0x33, 0x20,
	0x03, 0x28, 0x0e, 0x32, 0x28, 0x2e, 0x6a, 0x73, 0x6f, 0x6e, 0x77, 0x72, 0x69, 0x74, 0x65, 0x72,
	0x5f, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x54, 0x65, 0x73, 0x74, 0x41, 0x6c, 0x6c, 0x54, 0x79, 0x70,
	0x65, 0x73, 0x2e, 0x4e, 0x65, 0x73, 0x74, 0x65, 0x64, 0x45, 0x6e, 0x75, 0x6d, 0x52, 0x12, 0x72,
	0x65, 0x70, 0x65, 0x61, 0x74, 0x65, 0x64, 0x4e, 0x65, 0x73, 0x74, 0x65, 0x64, 0x45, 0x6e, 0x75,
	0x6d, 0x1a, 0x2c, 0x0a, 0x0d, 0x4e, 0x65, 0x73, 0x74, 0x65, 0x64, 0x4d, 0x65, 0x73, 0x73, 0x61,
	0x67, 0x65, 0x12, 0x1b, 0x0a, 0x09, 0x69, 0x6e, 0x74, 0x5f, 0x66, 0x69, 0x65, 0x6c, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x08, 0x69, 0x6e, 0x74, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x22,
	0x27, 0x0a, 0x0a, 0x4e, 0x65, 0x73, 0x74, 0x65, 0x64, 0x45, 0x6e, 0x75, 0x6d, 0x12, 0x07, 0x0a,
	0x03, 0x46, 0x4f, 0x4f, 0x10, 0x01, 0x12, 0x07, 0x0a, 0x03, 0x42, 0x41, 0x52, 0x10, 0x02, 0x12,
	0x07, 0x0a, 0x03, 0x42, 0x41, 0x5a, 0x10, 0x03,
}

var (
	file_yb_util_jsonwriter_test_proto_rawDescOnce sync.Once
	file_yb_util_jsonwriter_test_proto_rawDescData = file_yb_util_jsonwriter_test_proto_rawDesc
)

func file_yb_util_jsonwriter_test_proto_rawDescGZIP() []byte {
	file_yb_util_jsonwriter_test_proto_rawDescOnce.Do(func() {
		file_yb_util_jsonwriter_test_proto_rawDescData = protoimpl.X.CompressGZIP(file_yb_util_jsonwriter_test_proto_rawDescData)
	})
	return file_yb_util_jsonwriter_test_proto_rawDescData
}

var file_yb_util_jsonwriter_test_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_yb_util_jsonwriter_test_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_yb_util_jsonwriter_test_proto_goTypes = []interface{}{
	(TestAllTypes_NestedEnum)(0),       // 0: jsonwriter_test.TestAllTypes.NestedEnum
	(*TestAllTypes)(nil),               // 1: jsonwriter_test.TestAllTypes
	(*TestAllTypes_NestedMessage)(nil), // 2: jsonwriter_test.TestAllTypes.NestedMessage
}
var file_yb_util_jsonwriter_test_proto_depIdxs = []int32{
	2, // 0: jsonwriter_test.TestAllTypes.optional_nested_message:type_name -> jsonwriter_test.TestAllTypes.NestedMessage
	0, // 1: jsonwriter_test.TestAllTypes.optional_nested_enum:type_name -> jsonwriter_test.TestAllTypes.NestedEnum
	2, // 2: jsonwriter_test.TestAllTypes.repeated_nested_message:type_name -> jsonwriter_test.TestAllTypes.NestedMessage
	0, // 3: jsonwriter_test.TestAllTypes.repeated_nested_enum:type_name -> jsonwriter_test.TestAllTypes.NestedEnum
	4, // [4:4] is the sub-list for method output_type
	4, // [4:4] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_yb_util_jsonwriter_test_proto_init() }
func file_yb_util_jsonwriter_test_proto_init() {
	if File_yb_util_jsonwriter_test_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_yb_util_jsonwriter_test_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TestAllTypes); i {
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
		file_yb_util_jsonwriter_test_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TestAllTypes_NestedMessage); i {
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
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_yb_util_jsonwriter_test_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_yb_util_jsonwriter_test_proto_goTypes,
		DependencyIndexes: file_yb_util_jsonwriter_test_proto_depIdxs,
		EnumInfos:         file_yb_util_jsonwriter_test_proto_enumTypes,
		MessageInfos:      file_yb_util_jsonwriter_test_proto_msgTypes,
	}.Build()
	File_yb_util_jsonwriter_test_proto = out.File
	file_yb_util_jsonwriter_test_proto_rawDesc = nil
	file_yb_util_jsonwriter_test_proto_goTypes = nil
	file_yb_util_jsonwriter_test_proto_depIdxs = nil
}
