package healthcheck

import (
	"github.com/blang/vfs"
	"github.com/go-logr/logr"
	"github.com/google/uuid"
	. "github.com/icza/gox/gox"
	"github.com/pkg/errors"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/cdc"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/common"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/master"
	"github.com/yugabyte/yb-tools/yugatool/api/yugatool/config"
	"github.com/yugabyte/yb-tools/yugatool/api/yugatool/healthcheck"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client"
	"google.golang.org/protobuf/encoding/prototext"
)

type CDCProducerReport struct {
	*healthcheck.CDCProducerReportPB

	Log            logr.Logger
	ConsumerClient *client.YBClient
	ConsumerUUID   string
	Producer       *cdc.ProducerEntryPB
	Config         *config.UniverseConfigPB
}

func NewCDCProducerReport(log logr.Logger, consumerClient *client.YBClient, consumerClusterConfig *config.UniverseConfigPB, consumerUUID string, producerID string, producer *cdc.ProducerEntryPB) *CDCProducerReport {
	return &CDCProducerReport{
		CDCProducerReportPB: &healthcheck.CDCProducerReportPB{
			ProducerId:              &producerID,
			ProducerMasterAddresses: producer.GetMasterAddrs(),
		},
		Log:            log.WithName("CDCProducerReport").WithValues("producerID", producerID),
		ConsumerClient: consumerClient,
		ConsumerUUID:   consumerUUID,
		Producer:       producer,
		Config:         consumerClusterConfig,
	}
}

func (r *CDCProducerReport) RunCheck() error {
	reportErrors := &healthcheck.CDCProducerReportPBErrorlist{}
	errorSet := false
	if r.GetProducerId() == r.ConsumerUUID {
		reportErrors.MastersReferenceSelf = NewBool(true)
		// TODO: check master addresses directly
		r.Log.V(1).Info("consumer and producer UUID match")
	}

	r.Log.V(1).Info("connecting to producer")
	producerClient := &client.YBClient{
		Log:    r.Log.WithName("ProducerClient"),
		Fs:     vfs.OS(),
		Config: r.Config,
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
