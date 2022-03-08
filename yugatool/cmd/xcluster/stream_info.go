/*
Copyright © 2021 Yugabyte Support

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package xcluster

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/pkg/format"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/master"
	"github.com/yugabyte/yb-tools/yugatool/pkg/cmdutil"
)

func StreamInfoCmd(ctx *cmdutil.YugatoolContext) *cobra.Command {
	options := &ClusterInfoOptions{}
	cmd := &cobra.Command{
		Use:   "stream_info",
		Short: "Export xCluster stream information",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := ctx.WithCmd(cmd).WithOptions(options).Setup()
			if err != nil {
				return err
			}
			defer ctx.Client.Close()

			return streamInfo(ctx, options)
		},
	}
	options.AddFlags(cmd)

	return cmd
}

type ClusterInfoOptions struct {
	GetLastUpdated bool
}

func (o *ClusterInfoOptions) AddFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.BoolVar(&o.GetLastUpdated, "get-last-updated", false, "Obtain the last updated date via YCQL")
}

func (o *ClusterInfoOptions) Validate() error {
	return nil
}

var _ cmdutil.CommandOptions = &ClusterInfoOptions{}

func streamInfo(ctx *cmdutil.YugatoolContext, options *ClusterInfoOptions) error {
	c := ctx.Client

	// TODO: set tableId if requested
	listCDCStreamsResponse, err := c.Master.MasterService.ListCDCStreams(&master.ListCDCStreamsRequestPB{
		TableId: nil,
	})
	if err != nil {
		return err
	}

	if listCDCStreamsResponse.GetError() != nil {
		return fmt.Errorf("unable to list cdc streams: %s", listCDCStreamsResponse.Error.String())
	}

	type StreamReport struct {
		StreamInfo *master.CDCStreamInfoPB          `json:"stream_info,omitempty"`
		TableInfo  *master.GetTableSchemaResponsePB `json:"table_info,omitempty"`
	}

	var streamReport []*StreamReport

	for _, stream := range listCDCStreamsResponse.GetStreams() {
		appendStreamReport := func(stream *master.CDCStreamInfoPB, table *master.GetTableSchemaResponsePB) {
			streamReport = append(streamReport, &StreamReport{stream, table})
		}

		if stream.GetTableId() == nil {
			ctx.Log.Error(nil, "stream does not have a TableID", "stream", stream)
			appendStreamReport(stream, nil)
			continue
		}

		tableSchemaResponse, err := c.Master.MasterService.GetTableSchema(&master.GetTableSchemaRequestPB{
			Table: &master.TableIdentifierPB{
				TableId: stream.GetTableId(),
			},
		})
		if err != nil {
			ctx.Log.Error(err, "failed to get table schema", "stream", stream)
		}

		if tableSchemaResponse.GetError() != nil {
			ctx.Log.Error(fmt.Errorf("%s", tableSchemaResponse.GetError()), "failed to get table schema")
		}

		appendStreamReport(stream, tableSchemaResponse)
	}

	cdcStreamReport := format.Output{
		JSONObject: streamReport,
		OutputType: ctx.GlobalOptions.Output,
		TableColumns: []format.Column{
			{Name: "STREAM_ID", Expr: "base64_decode(@.stream_info.streamId)"},
			{Name: "TABLE_ID", Expr: "base64_decode(@.stream_info.tableId)"},
			{Name: "TABLE", JSONPath: "$.table_info.identifier.tableName"},
			{Name: "NAMESPACE", JSONPath: "$.table_info.identifier.namespace.name"},
			{Name: "RECORD_TYPE", Expr: "base64_decode(@.stream_info.options[?(@.key == 'record_type')].value)"},
			{Name: "RECORD_FORMAT", Expr: "base64_decode(@.stream_info.options[?(@.key == 'record_format')].value)"},
			{Name: "STATE", Expr: "base64_decode(@.stream_info.options[?(@.key == 'state')].value)"},
		},
	}

	return cdcStreamReport.Print()
}
