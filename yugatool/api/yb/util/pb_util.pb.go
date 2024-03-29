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
// source: yb/util/pb_util.proto

package util

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	descriptorpb "google.golang.org/protobuf/types/descriptorpb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Supplemental protobuf container header, after the main header (see
// pb_util.h for details).
type ContainerSupHeaderPB struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The protobuf schema for the messages expected in this container.
	//
	// This schema is complete, that is, it includes all of its dependencies
	// (i.e. other schemas defined in .proto files imported by this schema's
	// .proto file).
	Protos *descriptorpb.FileDescriptorSet `protobuf:"bytes,1,req,name=protos" json:"protos,omitempty"`
	// The PB message type expected in each data entry in this container. Must
	// be fully qualified (i.e. yb.tablet.RaftGroupReplicaSuperBlockPB).
	PbType *string `protobuf:"bytes,2,req,name=pb_type,json=pbType" json:"pb_type,omitempty"`
}

func (x *ContainerSupHeaderPB) Reset() {
	*x = ContainerSupHeaderPB{}
	if protoimpl.UnsafeEnabled {
		mi := &file_yb_util_pb_util_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ContainerSupHeaderPB) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ContainerSupHeaderPB) ProtoMessage() {}

func (x *ContainerSupHeaderPB) ProtoReflect() protoreflect.Message {
	mi := &file_yb_util_pb_util_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ContainerSupHeaderPB.ProtoReflect.Descriptor instead.
func (*ContainerSupHeaderPB) Descriptor() ([]byte, []int) {
	return file_yb_util_pb_util_proto_rawDescGZIP(), []int{0}
}

func (x *ContainerSupHeaderPB) GetProtos() *descriptorpb.FileDescriptorSet {
	if x != nil {
		return x.Protos
	}
	return nil
}

func (x *ContainerSupHeaderPB) GetPbType() string {
	if x != nil && x.PbType != nil {
		return *x.PbType
	}
	return ""
}

var File_yb_util_pb_util_proto protoreflect.FileDescriptor

var file_yb_util_pb_util_proto_rawDesc = []byte{
	0x0a, 0x15, 0x79, 0x62, 0x2f, 0x75, 0x74, 0x69, 0x6c, 0x2f, 0x70, 0x62, 0x5f, 0x75, 0x74, 0x69,
	0x6c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x02, 0x79, 0x62, 0x1a, 0x20, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64, 0x65, 0x73,
	0x63, 0x72, 0x69, 0x70, 0x74, 0x6f, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x6b, 0x0a,
	0x14, 0x43, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x53, 0x75, 0x70, 0x48, 0x65, 0x61,
	0x64, 0x65, 0x72, 0x50, 0x42, 0x12, 0x3a, 0x0a, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x18,
	0x01, 0x20, 0x02, 0x28, 0x0b, 0x32, 0x22, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x46, 0x69, 0x6c, 0x65, 0x44, 0x65, 0x73, 0x63,
	0x72, 0x69, 0x70, 0x74, 0x6f, 0x72, 0x53, 0x65, 0x74, 0x52, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x73, 0x12, 0x17, 0x0a, 0x07, 0x70, 0x62, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x02,
	0x28, 0x09, 0x52, 0x06, 0x70, 0x62, 0x54, 0x79, 0x70, 0x65, 0x42, 0x08, 0x0a, 0x06, 0x6f, 0x72,
	0x67, 0x2e, 0x79, 0x62,
}

var (
	file_yb_util_pb_util_proto_rawDescOnce sync.Once
	file_yb_util_pb_util_proto_rawDescData = file_yb_util_pb_util_proto_rawDesc
)

func file_yb_util_pb_util_proto_rawDescGZIP() []byte {
	file_yb_util_pb_util_proto_rawDescOnce.Do(func() {
		file_yb_util_pb_util_proto_rawDescData = protoimpl.X.CompressGZIP(file_yb_util_pb_util_proto_rawDescData)
	})
	return file_yb_util_pb_util_proto_rawDescData
}

var file_yb_util_pb_util_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_yb_util_pb_util_proto_goTypes = []interface{}{
	(*ContainerSupHeaderPB)(nil),           // 0: yb.ContainerSupHeaderPB
	(*descriptorpb.FileDescriptorSet)(nil), // 1: google.protobuf.FileDescriptorSet
}
var file_yb_util_pb_util_proto_depIdxs = []int32{
	1, // 0: yb.ContainerSupHeaderPB.protos:type_name -> google.protobuf.FileDescriptorSet
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_yb_util_pb_util_proto_init() }
func file_yb_util_pb_util_proto_init() {
	if File_yb_util_pb_util_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_yb_util_pb_util_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ContainerSupHeaderPB); i {
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
			RawDescriptor: file_yb_util_pb_util_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_yb_util_pb_util_proto_goTypes,
		DependencyIndexes: file_yb_util_pb_util_proto_depIdxs,
		MessageInfos:      file_yb_util_pb_util_proto_msgTypes,
	}.Build()
	File_yb_util_pb_util_proto = out.File
	file_yb_util_pb_util_proto_rawDesc = nil
	file_yb_util_pb_util_proto_goTypes = nil
	file_yb_util_pb_util_proto_depIdxs = nil
}
