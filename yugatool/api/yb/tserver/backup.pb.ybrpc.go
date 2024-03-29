// Copyright (c) YugaByte, Inc.
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

// service: yb.tserver.TabletServerBackupService
// service: TabletServerBackupService
type TabletServerBackupService interface {
	TabletSnapshotOp(request *TabletSnapshotOpRequestPB) (*TabletSnapshotOpResponsePB, error)
}

type TabletServerBackupServiceImpl struct {
	Log       logr.Logger
	Messenger message.Messenger
}

func (s *TabletServerBackupServiceImpl) TabletSnapshotOp(request *TabletSnapshotOpRequestPB) (*TabletSnapshotOpResponsePB, error) {
	s.Log.V(1).Info("sending RPC request", "service", "yb.tserver.TabletServerBackupService", "method", "TabletSnapshotOp", "request", request)
	response := &TabletSnapshotOpResponsePB{}

	err := s.Messenger.SendMessage("yb.tserver.TabletServerBackupService", "TabletSnapshotOp", request.ProtoReflect().Interface(), response.ProtoReflect().Interface())
	if err != nil {
		return nil, err
	}

	s.Log.V(1).Info("received RPC response", "service", "yb.tserver.TabletServerBackupService", "method", "TabletSnapshotOp", "response", response)

	return response, nil
}
