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
// source: yb/util/opid.proto

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

// An id for a generic state machine operation. Composed of the leaders' term
// plus the index of the operation in that term, e.g., the <index>th operation
// of the <term>th leader.
type OpIdPB struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The term of an operation or the leader's sequence id.
	Term  *int64 `protobuf:"varint,1,req,name=term" json:"term,omitempty"`
	Index *int64 `protobuf:"varint,2,req,name=index" json:"index,omitempty"`
}

func (x *OpIdPB) Reset() {
	*x = OpIdPB{}
	if protoimpl.UnsafeEnabled {
		mi := &file_yb_util_opid_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *OpIdPB) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OpIdPB) ProtoMessage() {}

func (x *OpIdPB) ProtoReflect() protoreflect.Message {
	mi := &file_yb_util_opid_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OpIdPB.ProtoReflect.Descriptor instead.
func (*OpIdPB) Descriptor() ([]byte, []int) {
	return file_yb_util_opid_proto_rawDescGZIP(), []int{0}
}

func (x *OpIdPB) GetTerm() int64 {
	if x != nil && x.Term != nil {
		return *x.Term
	}
	return 0
}

func (x *OpIdPB) GetIndex() int64 {
	if x != nil && x.Index != nil {
		return *x.Index
	}
	return 0
}

var File_yb_util_opid_proto protoreflect.FileDescriptor

var file_yb_util_opid_proto_rawDesc = []byte{
	0x0a, 0x12, 0x79, 0x62, 0x2f, 0x75, 0x74, 0x69, 0x6c, 0x2f, 0x6f, 0x70, 0x69, 0x64, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x02, 0x79, 0x62, 0x22, 0x32, 0x0a, 0x06, 0x4f, 0x70, 0x49, 0x64,
	0x50, 0x42, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x65, 0x72, 0x6d, 0x18, 0x01, 0x20, 0x02, 0x28, 0x03,
	0x52, 0x04, 0x74, 0x65, 0x72, 0x6d, 0x12, 0x14, 0x0a, 0x05, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x18,
	0x02, 0x20, 0x02, 0x28, 0x03, 0x52, 0x05, 0x69, 0x6e, 0x64, 0x65, 0x78, 0x42, 0x08, 0x0a, 0x06,
	0x6f, 0x72, 0x67, 0x2e, 0x79, 0x62,
}

var (
	file_yb_util_opid_proto_rawDescOnce sync.Once
	file_yb_util_opid_proto_rawDescData = file_yb_util_opid_proto_rawDesc
)

func file_yb_util_opid_proto_rawDescGZIP() []byte {
	file_yb_util_opid_proto_rawDescOnce.Do(func() {
		file_yb_util_opid_proto_rawDescData = protoimpl.X.CompressGZIP(file_yb_util_opid_proto_rawDescData)
	})
	return file_yb_util_opid_proto_rawDescData
}

var file_yb_util_opid_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_yb_util_opid_proto_goTypes = []interface{}{
	(*OpIdPB)(nil), // 0: yb.OpIdPB
}
var file_yb_util_opid_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_yb_util_opid_proto_init() }
func file_yb_util_opid_proto_init() {
	if File_yb_util_opid_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_yb_util_opid_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*OpIdPB); i {
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
			RawDescriptor: file_yb_util_opid_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_yb_util_opid_proto_goTypes,
		DependencyIndexes: file_yb_util_opid_proto_depIdxs,
		MessageInfos:      file_yb_util_opid_proto_msgTypes,
	}.Build()
	File_yb_util_opid_proto = out.File
	file_yb_util_opid_proto_rawDesc = nil
	file_yb_util_opid_proto_goTypes = nil
	file_yb_util_opid_proto_depIdxs = nil
}
