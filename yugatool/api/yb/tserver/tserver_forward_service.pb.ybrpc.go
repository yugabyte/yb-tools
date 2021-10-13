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

// Code generated by protoc-gen-ybrpc. DO NOT EDIT.

package tserver

import (
	"github.com/go-logr/logr"
	"github.com/yugabyte/yb-tools/protoc-gen-ybrpc/pkg/message"
)

// service: yb.tserver.TabletServerForwardService
// service: TabletServerForwardService
type TabletServerForwardService interface {
	Write(request *WriteRequestPB) (*WriteResponsePB, error)
	Read(request *ReadRequestPB) (*ReadResponsePB, error)
}

type TabletServerForwardServiceImpl struct {
	Log       logr.Logger
	Messenger message.Messenger
}

func (s *TabletServerForwardServiceImpl) Write(request *WriteRequestPB) (*WriteResponsePB, error) {
	s.Log.V(1).Info("sending RPC request", "service", "yb.tserver.TabletServerForwardService", "method", "Write", "request", request)
	response := &WriteResponsePB{}

	err := s.Messenger.SendMessage("yb.tserver.TabletServerForwardService", "Write", request.ProtoReflect().Interface(), response.ProtoReflect().Interface())
	if err != nil {
		return nil, err
	}

	s.Log.V(1).Info("received RPC response", "service", "yb.tserver.TabletServerForwardService", "method", "Write", "response", response)

	return response, nil
}

func (s *TabletServerForwardServiceImpl) Read(request *ReadRequestPB) (*ReadResponsePB, error) {
	s.Log.V(1).Info("sending RPC request", "service", "yb.tserver.TabletServerForwardService", "method", "Read", "request", request)
	response := &ReadResponsePB{}

	err := s.Messenger.SendMessage("yb.tserver.TabletServerForwardService", "Read", request.ProtoReflect().Interface(), response.ProtoReflect().Interface())
	if err != nil {
		return nil, err
	}

	s.Log.V(1).Info("received RPC response", "service", "yb.tserver.TabletServerForwardService", "method", "Read", "response", response)

	return response, nil
}