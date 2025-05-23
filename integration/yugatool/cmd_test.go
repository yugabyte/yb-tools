package yugatool_test

import (
	"encoding/json"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/tablet"

	"github.com/google/uuid"
	. "github.com/icza/gox/gox"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/yugabyte/yb-tools/integration/util"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/common"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/consensus"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/master"
	"github.com/yugabyte/yb-tools/yugatool/api/yb/tserver"
	"github.com/yugabyte/yb-tools/yugatool/cmd"
	"github.com/yugabyte/yb-tools/yugatool/pkg/client"
)

var _ = Describe("Yugatool Integration Tests", func() {
	Context("TLS connection", func() {
		When("listing cluster info", func() {
			var (
				err error
			)
			BeforeEach(func() {
				universe := CreateTLSTestUniverseIfNotExists()

				_, err = universe.RunYugatoolCommand("cluster_info")
			})
			It("returns successfully", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Context("cluster_info", func() {
		When("listing cluster info", func() {
			var (
				command       []string
				clusterConfig master.SysClusterConfigEntryPB
				masterServers MasterServersReport
				tabletServers TabletServersReport
				tabletReports []*TabletReport
			)
			BeforeEach(func() {
				command = []string{"cluster_info", "-o", "json"}
			})
			JustBeforeEach(func() {
				universe := CreateTestUniverseIfNotExists()

				clusterInfo, err := universe.RunYugatoolCommand(command...)
				Expect(err).NotTo(HaveOccurred())
				dec := json.NewDecoder(clusterInfo)

				Expect(err).NotTo(HaveOccurred())

				Expect(dec.More()).To(BeTrue())
				err = dec.Decode(&clusterConfig)
				Expect(err).NotTo(HaveOccurred())

				Expect(dec.More()).To(BeTrue())
				err = dec.Decode(&masterServers)
				Expect(err).NotTo(HaveOccurred())

				Expect(dec.More()).To(BeTrue())
				err = dec.Decode(&tabletServers)
				Expect(err).NotTo(HaveOccurred())

				tabletReports = []*TabletReport{}
				for dec.More() {
					report := &TabletReport{}

					err = dec.Decode(&report)
					Expect(err).NotTo(HaveOccurred())
					tabletReports = append(tabletReports, report)
				}
			})
			It("returns successfully", func() {
				_, err := uuid.Parse(*clusterConfig.ClusterUuid)
				Expect(err).NotTo(HaveOccurred())

				for _, master := range masterServers.Content {
					_, err := uuid.Parse(string(master.InstanceId.PermanentUuid))
					Expect(err).NotTo(HaveOccurred())
				}

				for _, tserver := range tabletServers.Content {
					_, err := uuid.Parse(string(tserver.InstanceId.PermanentUuid))
					Expect(err).NotTo(HaveOccurred())
				}

				Expect(tabletReports).To(BeEmpty())
			})
			When("collecting tablet reports", func() {
				BeforeEach(func() {
					command = append(command, "--tablet-report")

					universe := CreateTestUniverseIfNotExists()

					db := universe.YSQLConnection()
					defer db.Close()

					_, err := db.Exec(`CREATE TABLE IF NOT EXISTS foo(a int);`)
					Expect(err).NotTo(HaveOccurred())
				})
				It("collects a tablet report from every tserver", func() {
					Expect(tabletReports).To(HaveLen(len(tabletServers.Content)))

					for _, report := range tabletReports {
						for _, tablet := range report.Content {
							_, err := uuid.Parse(tablet.Tablet.GetTabletStatus().GetTabletId())
							Expect(err).NotTo(HaveOccurred())
						}
					}
				})
				When("specifying a table name", func() {
					BeforeEach(func() {
						command = append(command, "--table", "foo")
					})
					It("only returns tablets for that table", func() {
						for _, report := range tabletReports {
							for _, tablet := range report.Content {
								Expect(tablet.Tablet.GetTabletStatus().GetTableName()).To(Equal("foo"))
							}
						}
					})
				})

				When("specifying a namespace", func() {
					BeforeEach(func() {
						command = append(command, "--namespace", "yugabyte")
					})
					It("only returns tablets for that namespace", func() {
						for _, report := range tabletReports {
							for _, tablet := range report.Content {
								Expect(tablet.Tablet.GetTabletStatus().GetNamespaceName()).To(Equal("yugabyte"))
							}
						}
					})
				})

				When("specifying a leaders only", func() {
					BeforeEach(func() {
						command = append(command, "--leaders-only")
					})
					It("only returns tablets that have the leader lease", func() {
						for _, report := range tabletReports {
							for _, tablet := range report.Content {
								Expect(tablet.ConsensusState.GetLeaderLeaseStatus()).To(Equal(consensus.LeaderLeaseStatus_HAS_LEASE))
							}
						}
					})
				})

				// TODO: This should be extracted into utility function to RemoveReplica()
				When("showing tombstoned tablets", func() {
					var universe *util.YugatoolTestContext
					BeforeEach(func() {
						command = append(command, "--show-tombstoned")

						universe = CreateTestUniverseIfNotExists()

						resp, err := universe.Master.MasterClusterService.ChangeLoadBalancerState(&master.ChangeLoadBalancerStateRequestPB{
							IsEnabled: NewBool(false),
						})
						Expect(err).NotTo(HaveOccurred())
						Expect(resp.Error).To(BeNil())

						nodes, errs := universe.AllTservers()
						for err := range errs {
							Expect(err).NotTo(HaveOccurred())
						}
						leader := nodes[0]

						listTabletsResponse, err := leader.TabletServerService.ListTabletsForTabletServer(&tserver.ListTabletsForTabletServerRequestPB{})
						Expect(err).NotTo(HaveOccurred())

						var tabletToDelete []byte
						for _, tablet := range listTabletsResponse.Entries {
							if tablet.GetTableName() == "foo" && tablet.GetIsLeader() {
								tabletToDelete = tablet.GetTabletId()
								break
							}
						}

						var follower *client.HostState
						for _, server := range nodes {
							listTabletsResponse, err := server.TabletServerService.ListTabletsForTabletServer(&tserver.ListTabletsForTabletServerRequestPB{})
							Expect(err).NotTo(HaveOccurred())

							for _, tablet := range listTabletsResponse.Entries {
								if string(tablet.GetTabletId()) == string(tabletToDelete) && !tablet.GetIsLeader() {
									follower = server
									break
								}
							}
						}
						Expect(follower).NotTo(BeNil())

						changeConfigType := consensus.ChangeConfigType_REMOVE_SERVER
						deleteType := tablet.TabletDataState_TABLET_DATA_TOMBSTONED

						changeConfigResp, err := leader.ConsensusService.ChangeConfig(&consensus.ChangeConfigRequestPB{
							DestUuid: leader.Status.NodeInstance.PermanentUuid,
							TabletId: tabletToDelete,
							Type:     &changeConfigType,
							Server:   &consensus.RaftPeerPB{PermanentUuid: follower.Status.NodeInstance.PermanentUuid},
						})
						Expect(err).NotTo(HaveOccurred())
						Expect(changeConfigResp.Error).To(BeNil())

						deleteTabletResponse, err := follower.TabletServerAdminService.DeleteTablet(&tserver.DeleteTabletRequestPB{
							DestUuid:   follower.Status.NodeInstance.PermanentUuid,
							TabletId:   tabletToDelete,
							Reason:     NewString("retire a tablet for test"),
							DeleteType: &deleteType,
							HideOnly:   NewBool(false),
						})

						Expect(err).NotTo(HaveOccurred())
						Expect(deleteTabletResponse.Error).To(BeNil())

					})
					JustBeforeEach(func() {
						resp, err := universe.Master.MasterClusterService.ChangeLoadBalancerState(&master.ChangeLoadBalancerStateRequestPB{
							IsEnabled: NewBool(true),
						})
						Expect(err).NotTo(HaveOccurred())
						Expect(resp.Error).To(BeNil())
					})
					It("returns tombstoned tablets", func() {
						hasTombstoned := false
						for _, report := range tabletReports {
							for _, tabletInstance := range report.Content {
								if tabletInstance.Tablet.GetTabletStatus().GetTabletDataState() == tablet.TabletDataState_TABLET_DATA_TOMBSTONED {
									hasTombstoned = true
								}
							}
						}
						Expect(hasTombstoned).To(BeTrue())
					})
				})
			})
		})
	})
})

type MasterServersReport struct {
	Message string                  `json:"msg"`
	Content []*common.ServerEntryPB `json:"content"`
}
type TabletServersReport struct {
	Message string                                      `json:"msg"`
	Content []*master.ListTabletServersResponsePB_Entry `json:"content"`
}

type TabletReport struct {
	Message string           `json:"msg"`
	Content []cmd.TabletInfo `json:"content"`
}
