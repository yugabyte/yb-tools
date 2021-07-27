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
package cmd

import (
	"fmt"

	"github.com/blang/vfs"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	. "github.com/icza/gox/gox"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/cdc"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/common"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/master"
	"github.com/yugabyte/yb-tools/yugatool/api/yugatool/config"
	"github.com/yugabyte/yb-tools/yugatool/api/yugatool/healthcheck"
	"github.com/yugabyte/yb-tools/yugatool/cmd/util"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client"
	"google.golang.org/protobuf/encoding/prototext"
)

// clusterInfoCmd represents the clusterInfo command
var cdcProducerCheckCmd = &cobra.Command{
	Use:   "cdc_producer_check",
	Short: "Check for xCluster replication configuration issues",
	Long:  `Check for xCluster replication configuration issues between yugabyte clusters.`,
	RunE:  cdcProducerCheck,
}

func init() {
	rootCmd.AddCommand(cdcProducerCheckCmd)
}

func cdcProducerCheck(_ *cobra.Command, _ []string) error {
	log, err := util.GetLogger("cdc_producer_check", debug)
	if err != nil {
		return err
	}

	hosts, err := util.ValidateMastersFlag(masterAddresses)
	if err != nil {
		return err
	}

	consumerClient := &client.YBClient{
		Log: log.WithName("ConsumerClient"),
		Fs:  vfs.OS(),
		Config: &config.UniverseConfigPB{
			Masters:        hosts,
			TimeoutSeconds: &dialTimeout,
			SslOpts: &config.SslOptionsPB{
				SkipHostVerification: &skipHostVerification,
				CaCertPath:           &caCert,
				CertPath:             &clientCert,
				KeyPath:              &clientKey,
			},
		},
	}

	err = consumerClient.Connect()
	if err != nil {
		return err
	}
	defer consumerClient.Close()

	return RunCDCProducerCheck(log, consumerClient)
}

type CDCProducerReport struct {
	*healthcheck.CDCProducerReportPB

	Log                   logr.Logger
	ConsumerClient        *client.YBClient
	ConsumerClusterConfig *master.SysClusterConfigEntryPB
	Producer              *cdc.ProducerEntryPB
}

func NewCDCProducerReport(log logr.Logger, consumerClient *client.YBClient, consumerClusterConfig *master.SysClusterConfigEntryPB, producerID string, producer *cdc.ProducerEntryPB) *CDCProducerReport {
	return &CDCProducerReport{
		CDCProducerReportPB: &healthcheck.CDCProducerReportPB{
			ProducerId:              &producerID,
			ProducerMasterAddresses: producer.GetMasterAddrs(),
		},
		Log:                   log.WithName("CDCProducerReport").WithValues("producerID", producerID),
		ConsumerClient:        consumerClient,
		ConsumerClusterConfig: consumerClusterConfig,
		Producer:              producer,
	}
}

func (r *CDCProducerReport) RunCheck() error {
	reportErrors := &healthcheck.CDCProducerReportPBErrorlist{}
	errorSet := false
	if r.GetProducerId() == r.ConsumerClusterConfig.GetClusterUuid() {
		reportErrors.MastersReferenceSelf = NewBool(true)
		// TODO: check master addresses directly
		r.Log.V(1).Info("consumer and producer UUID match")
	}

	r.Log.V(1).Info("connecting to producer")
	producerClient := &client.YBClient{
		Log: r.Log.WithName("ProducerClient"),
		Fs:  vfs.OS(),
		Config: &config.UniverseConfigPB{
			Masters:        r.Producer.GetMasterAddrs(),
			TimeoutSeconds: &dialTimeout,
			SslOpts: &config.SslOptionsPB{
				SkipHostVerification: &skipHostVerification,
				CaCertPath:           &caCert,
				CertPath:             &clientCert,
				KeyPath:              &clientKey,
			},
		},
	}

	err := producerClient.Connect()
	if err != nil {
		return err
	}

	defer producerClient.Close()

	for streamID, streamEntry := range r.Producer.GetStreamMap() {
		streamReport, err := NewCDCProducerStreamReport(r.Log, r.ConsumerClient, producerClient, streamID, streamEntry)
		if err != nil {
			return err
		}

		err = streamReport.RunCheck()
		if err != nil {
			return err
		}

		if streamReport.Errors != nil {
			errorSet = true
			reportErrors.StreamReports = append(reportErrors.StreamReports, streamReport.CDCProducerStreamReportPB)
		}
	}
	if errorSet {
		r.CDCProducerReportPB.Errors = reportErrors
	}
	return nil
}

type CDCProducerStreamReport struct {
	*healthcheck.CDCProducerStreamReportPB

	Log            logr.Logger
	ConsumerClient *client.YBClient
	ProducerClient *client.YBClient
	StreamEntry    *cdc.StreamEntryPB
	ConsumerSchema *master.GetTableSchemaResponsePB
	ProducerSchema *master.GetTableSchemaResponsePB
}

func NewCDCProducerStreamReport(log logr.Logger, consumerClient *client.YBClient, producerClient *client.YBClient, streamID string, streamEntry *cdc.StreamEntryPB) (*CDCProducerStreamReport, error) {
	consumerTableSchema, err := consumerClient.Master.MasterService.GetTableSchema(&master.GetTableSchemaRequestPB{
		Table: &master.TableIdentifierPB{
			TableId: []byte(streamEntry.ConsumerTableId),
		},
	})
	if err != nil {
		return nil, err
	}

	producerTableSchema, err := producerClient.Master.MasterService.GetTableSchema(&master.GetTableSchemaRequestPB{
		Table: &master.TableIdentifierPB{
			TableId: []byte(streamEntry.ProducerTableId),
		},
	})
	if err != nil {
		return nil, err
	}

	// If the table query failed on the consumer, fall back to the producer
	var tableSchema = consumerTableSchema.Identifier
	if consumerTableSchema.Error != nil {
		tableSchema = producerTableSchema.Identifier
	}

	streamReport := &CDCProducerStreamReport{
		CDCProducerStreamReportPB: &healthcheck.CDCProducerStreamReportPB{
			Table:           tableSchema,
			ProducerTableId: &streamEntry.ProducerTableId,
			ConsumerTableId: &streamEntry.ConsumerTableId,
			StreamId:        &streamID,
		},
		Log: log.WithName("CDCProducerStreamReport").
			WithValues("stream", streamID,
				"consumerTableId", streamEntry.ConsumerTableId,
				"producerTableId", streamEntry.ProducerTableId,
				"table", tableSchema),
		ConsumerClient: consumerClient,
		ProducerClient: producerClient,
		StreamEntry:    streamEntry,
		ConsumerSchema: consumerTableSchema,
		ProducerSchema: producerTableSchema,
	}
	return streamReport, nil
}

func (r *CDCProducerStreamReport) RunCheck() error {
	reportErrors := &healthcheck.CDCProducerStreamReportPBErrorlist{}
	errorSet := false
	var consumerTablets, producerTablets []string
	for consumerTablet, producerTabletList := range r.StreamEntry.GetConsumerProducerTabletMap() {
		consumerTablets = append(consumerTablets, consumerTablet)
		producerTablets = append(producerTablets, producerTabletList.GetTablets()...)
	}

	// Finish the report
	r.ConsumerTabletCount = NewUint32(uint32(len(consumerTablets)))
	r.ProducerTabletCount = NewUint32(uint32(len(producerTablets)))

	// Get replication lag
	replicatedIndexes, err := GetReplicatedIndexes(r.Log, r.ProducerClient, r.GetStreamId(), producerTablets...)
	if err != nil {
		return err
	}
	reportErrors.TabletsWithReplicationLag = GetReplicationLag(replicatedIndexes)
	if len(reportErrors.TabletsWithReplicationLag.ReplicatedIndexList) > 0 {
		errorSet = true
	}

	// Check for schema errors
	if r.ConsumerSchema.Error != nil {
		reportErrors.ConsumerSchemaError = r.ConsumerSchema.Error
		errorSet = true
	}

	if r.ProducerSchema.Error != nil {
		reportErrors.ProducerSchemaError = r.ProducerSchema.Error
		errorSet = true
	}
	// Check consumer tablets exist
	missingConsumerTablets, err := GetMissingTablets(r.ConsumerClient, consumerTablets)
	if err != nil {
		return err
	}
	if len(missingConsumerTablets) > 0 {
		r.Log.V(1).Info("missing tablets on consumer", "consumerTablets", missingConsumerTablets)
		reportErrors.MissingTabletsConsumer = append(reportErrors.MissingTabletsProducer, missingConsumerTablets...)
		errorSet = true
	}

	// Check producer tablets exist
	missingProducerTablets, err := GetMissingTablets(r.ProducerClient, producerTablets)
	if err != nil {
		return err
	}
	if len(missingProducerTablets) > 0 {
		r.Log.V(1).Info("missing tablets on producer", "producerTablets", missingProducerTablets)
		reportErrors.MissingTabletsProducer = append(reportErrors.MissingTabletsProducer, missingProducerTablets...)
		errorSet = true
	}

	// Only add errors to the report when they are detected
	if errorSet {
		r.CDCProducerStreamReportPB.Errors = reportErrors
	}
	return nil
}

func GetMissingTablets(client *client.YBClient, tablets []string) ([]string, error) {
	var missingTablets []string
	var tabletBytes [][]byte

	for _, tablet := range tablets {
		tabletBytes = append(tabletBytes, []byte(tablet))
	}
	response, err := client.Master.MasterService.GetTabletLocations(&master.GetTabletLocationsRequestPB{
		TabletIds: tabletBytes,
	})
	if err != nil {
		return missingTablets, err
	}

	if response.Error != nil {
		return missingTablets, errors.Errorf("master error: %s", prototext.Format(response.Error))
	}

	for _, tabletError := range response.Errors {
		if tabletError.Status.GetCode() == common.AppStatusPB_NOT_FOUND {
			missingTablets = append(missingTablets, string(tabletError.TabletId))
		} else {
			return missingTablets, errors.Errorf("unexpected error message: %s", tabletError)
		}
	}

	return missingTablets, nil
}

func GetReplicatedIndexes(log logr.Logger, client *client.YBClient, streamID string, tablets ...string) (*healthcheck.CDCReplicatedIndexListPB, error) {
	replicatedIndexes := &healthcheck.CDCReplicatedIndexListPB{
		ReplicatedIndexList: []*healthcheck.CDCReplicatedIndexPB{},
	}

	var tabletBytes [][]byte
	for _, tablet := range tablets {
		tabletBytes = append(tabletBytes, []byte(tablet))
	}

	response, err := client.Master.MasterService.GetTabletLocations(&master.GetTabletLocationsRequestPB{
		TabletIds: tabletBytes,
	})
	if err != nil {
		return replicatedIndexes, err
	}
	for _, tablet := range response.TabletLocations {
		if len(tablet.Replicas) > 0 {
			for _, replica := range tablet.Replicas {
				if replica.GetRole() == common.RaftPeerPB_LEADER {
					tserverUUID, err := uuid.ParseBytes(replica.GetTsInfo().GetPermanentUuid())
					if err != nil {
						return replicatedIndexes, err
					}
					host, ok := client.TServersUUIDMap[tserverUUID]
					if !ok {
						return replicatedIndexes, errors.Errorf("did not find server %s in client UUID map: %v", tserverUUID, client.TServersUUIDMap)
					}
					checkpoint, err := host.CDCService.GetCheckpoint(&cdc.GetCheckpointRequestPB{
						StreamId: []byte(streamID),
						TabletId: tablet.TabletId,
					})
					if err != nil {
						return replicatedIndexes, err
					}
					// The checkpoint location of the stream will only show up on the producer
					if checkpoint.Error == nil {
						latestOpID, err := host.CDCService.GetLatestEntryOpId(&cdc.GetLatestEntryOpIdRequestPB{TabletId: tablet.TabletId})
						if err != nil {
							return replicatedIndexes, err
						}
						replicatedIndexes.ReplicatedIndexList = append(replicatedIndexes.GetReplicatedIndexList(), &healthcheck.CDCReplicatedIndexPB{
							Tablet:             NewString(string(tablet.GetTabletId())),
							LatestOpid:         latestOpID.GetOpId(),
							CheckpointLocation: checkpoint.GetCheckpoint().GetOpId(),
						})
					} else {
						log.Info("error determining replicated index", "GetCheckpointError", checkpoint)
					}
				}
			}
		}
	}

	return replicatedIndexes, err
}

func GetReplicationLag(replicationIndexes *healthcheck.CDCReplicatedIndexListPB) *healthcheck.CDCReplicatedIndexListPB {
	tabletsWithLag := &healthcheck.CDCReplicatedIndexListPB{
		ReplicatedIndexList: []*healthcheck.CDCReplicatedIndexPB{},
	}
	for _, replicationIndex := range replicationIndexes.GetReplicatedIndexList() {
		hasLag := false
		if replicationIndex.LatestOpid.GetTerm() > replicationIndex.CheckpointLocation.GetTerm() {
			hasLag = true
		} else if replicationIndex.LatestOpid.GetIndex() > replicationIndex.CheckpointLocation.GetIndex() {
			hasLag = true
		}

		if hasLag {
			tabletsWithLag.ReplicatedIndexList = append(tabletsWithLag.ReplicatedIndexList, replicationIndex)
		}
	}
	return tabletsWithLag
}

func RunCDCProducerCheck(log logr.Logger, consumerClient *client.YBClient) error {
	clusterConfig, err := consumerClient.Master.MasterService.GetMasterClusterConfig(&master.GetMasterClusterConfigRequestPB{})
	if err != nil {
		return err
	}

	if clusterConfig.GetError() != nil {
		return errors.Errorf("Failed to get cluster config: %s", clusterConfig.GetError())
	}

	for producerID, producer := range clusterConfig.GetClusterConfig().GetConsumerRegistry().GetProducerMap() {
		producerReport := NewCDCProducerReport(log, consumerClient, clusterConfig.GetClusterConfig(), producerID, producer)

		err := producerReport.RunCheck()
		if err != nil {
			return err
		}
		if producerReport.Errors != nil {
			fmt.Println(prototext.Format(producerReport))
		}

		// TODO: Check that the schema matches for both tables - check for problems such as adding/removing columns
		// TODO: Check that all masters are actually valid on producer
		//        - The master is actually in the producer cluster, and not in another cluster
		//        - The master is reachable
		// TODO: Check that every master in the producer is valid in the producer map
		// TODO: Confirm that the UUID of the universe actually matches the name of the CDC stream (it is never actually checked when creating the stream)
		// TODO: Report if streams are disabled
		// TODO: Check for multiple copies of replication of the same table
		// TODO: Check if table was deleted
		// TODO: Check that the producer isn't on the same system as this one
	}
	return nil
}
