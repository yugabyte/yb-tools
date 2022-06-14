package yugaware_client_test

import (
	"context"
	"encoding/pem"
	"time"

	"github.com/go-openapi/strfmt"
	. "github.com/icza/gox/gox"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/access_keys"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/certificate_info"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/cloud_providers"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/session_management"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/universe_cluster_mutations"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/client/universe_management"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/cmdutil"
)

var _ = Describe("Yugaware API Compatibility Tests", func() {
	Context("AccessKeys API", func() {
		// TODO: This will only return access keys when a non-kubernetes provider is used
		PWhen("Listing access keys", func() {
			var (
				accessKeys []*models.AccessKey
			)
			BeforeEach(func() {
				universe := CreateTestUniverseIfNotExists()

				params := access_keys.NewListParams().
					WithCUUID(ywContext.CustomerUUID()).
					WithPUUID(strfmt.UUID(universe.UniverseDetails.Clusters[0].UserIntent.Provider))

				keysResponse, err := ywContext.PlatformAPIs.AccessKeys.List(params, ywContext.SwaggerAuth)
				Expect(err).NotTo(HaveOccurred())
				accessKeys = keysResponse.GetPayload()
			})

			It("Returns a list of access keys", func() {
				Expect(len(accessKeys)).NotTo(BeZero())
			})
		})
	})

	Context("CertificateInfo", func() {
		When("Listing Certificates", func() {
			var (
				certificates []*models.CertificateInfo
				certificate  *models.CertificateRoot
			)

			BeforeEach(func() {
				_ = CreateTestUniverseIfNotExists()

				params := certificate_info.NewGetListOfCertificateParams().WithCUUID(ywContext.CustomerUUID())

				certificatesResponse, err := ywContext.PlatformAPIs.CertificateInfo.GetListOfCertificate(params, ywContext.SwaggerAuth)
				Expect(err).NotTo(HaveOccurred())
				certificates = certificatesResponse.GetPayload()
			})
			It("Returns a list of cloud certificates", func() {
				Expect(len(certificates)).NotTo(BeZero())
			})

			When("Getting a root certificate", func() {
				BeforeEach(func() {
					params := certificate_info.NewGetRootCertParams().
						WithCUUID(ywContext.CustomerUUID()).
						WithRUUID(certificates[0].UUID)

					certificateResponse, err := ywContext.PlatformAPIs.CertificateInfo.GetRootCert(params, ywContext.SwaggerAuth)
					Expect(err).NotTo(HaveOccurred())
					certificate = certificateResponse.GetPayload()
				})

				It("Returns a certificate", func() {
					block, _ := pem.Decode([]byte(certificate.RootCrt))
					Expect(block.Type).To(Equal("CERTIFICATE"))
				})
			})
		})
	})

	Context("CloudProviders API", func() {
		When("Listing cloud providers", func() {
			var (
				providers []*models.Provider
			)

			BeforeEach(func() {
				params := cloud_providers.NewGetListOfProvidersParams().
					WithCUUID(ywContext.CustomerUUID())
				response, err := ywContext.PlatformAPIs.CloudProviders.GetListOfProviders(params, ywContext.SwaggerAuth)
				Expect(err).NotTo(HaveOccurred())
				providers = response.GetPayload()
			})
			It("Returns a list of cloud providers", func() {
				Expect(len(providers)).NotTo(BeZero())
			})
		})
	})

	Context("SessionManagement API", func() {
		When("Determining a server version", func() {
			var (
				appVersionReturn map[string]string
			)
			BeforeEach(func() {
				params := session_management.NewAppVersionParams().WithDefaults()
				response, err := ywContext.PlatformAPIs.SessionManagement.AppVersion(params)
				Expect(err).NotTo(HaveOccurred())

				appVersionReturn = response.GetPayload()
			})
			It("Returns a valid version string", func() {
				Expect(appVersionReturn).To(gstruct.MatchKeys(gstruct.IgnoreExtras, gstruct.Keys{"version": MatchRegexp(`\d+\.\d+\.\d-b\d+`)}))
			})
		})

		When("Requesting the customer count", func() {
			var (
				customerCount *int32
			)
			BeforeEach(func() {
				params := session_management.NewCustomerCountParams().WithDefaults()
				response, err := ywContext.PlatformAPIs.SessionManagement.CustomerCount(params)
				Expect(err).NotTo(HaveOccurred())

				customerCount = response.GetPayload().Count
			})
			It("Returns a valid version string", func() {
				Expect(*customerCount).To(Equal(int32(1)))
			})
		})

		// TODO: This API call had breaking changes in yugabyte-db commit 855eeb0b6b. Need to figure out how to deal
		//       with issues like this
		XWhen("Requesting filtered yugaware logs", func() {
			var (
				universe *models.UniverseResp
				logLines []string
			)
			BeforeEach(func() {
				universe = CreateTestUniverseIfNotExists()

				params := (session_management.NewGetFilteredLogsParams().
					WithMaxLines(NewInt32(5))).
					WithUniverseName(NewString(universe.Name)).
					WithTimeout(time.Duration(options.DialTimeout) * time.Second)

				response, err := ywContext.PlatformAPIs.SessionManagement.GetFilteredLogs(params, ywContext.SwaggerAuth)
				Expect(err).NotTo(HaveOccurred())

				logLines = response.GetPayload().Lines
			})
			It("Returns valid logs", func() {
				Expect(logLines).To(ContainElement(ContainSubstring(string(universe.UniverseUUID))))
			})
		})

		When("Requesting yugaware logs", func() {
			var (
				logLines []string
			)
			BeforeEach(func() {
				params := session_management.NewGetLogsParams().
					WithMaxLines(int32(10))

				response, err := ywContext.PlatformAPIs.SessionManagement.GetLogs(params, ywContext.SwaggerAuth)
				Expect(err).NotTo(HaveOccurred())

				logLines = response.GetPayload().Lines
			})
			It("Returns valid logs", func() {
				Expect(logLines).To(HaveLen(10))
				Expect(logLines).To(ContainElement(MatchRegexp(`^(YW )?\d{4,4}-\d{2,2}-\d{2,2}(T| )\d{2,2}:\d{2,2}:\d{2,2}(,|.)\d{3,3}`)))
			})
		})

		When("Requesting the session info", func() {
			var (
				sessionInfo *models.SessionInfo
			)
			BeforeEach(func() {
				params := session_management.NewGetSessionInfoParams().
					WithDefaults()

				response, err := ywContext.PlatformAPIs.SessionManagement.GetSessionInfo(params, ywContext.SwaggerAuth)
				Expect(err).NotTo(HaveOccurred())

				sessionInfo = response.GetPayload()
			})
			It("Returns a valid password policy", func() {
				Expect(sessionInfo.CustomerUUID).To(Equal(ywContext.CustomerUUID()))
			})
		})
	})

	Context("UniverseClusterMutations API", func() {
		When("Updating an existing universe", func() {
			var (
				newNodeCount int32
				universe     *models.UniverseResp
			)
			BeforeEach(func() {
				originalUniverse := CreateTestUniverseIfNotExists()

				clusters := originalUniverse.UniverseDetails.Clusters
				newNodeCount = clusters[0].UserIntent.NumNodes + 1
				clusters[0].UserIntent.NumNodes = newNodeCount

				params := universe_cluster_mutations.NewUpdatePrimaryClusterParams().
					WithCUUID(ywContext.CustomerUUID()).
					WithUniUUID(originalUniverse.UniverseUUID).
					WithUniverseConfigureTaskParams(&models.UniverseConfigureTaskParams{
						Clusters:     clusters,
						UniverseUUID: originalUniverse.UniverseUUID,
					})

				response, err := ywContext.PlatformAPIs.UniverseClusterMutations.UpdatePrimaryCluster(params, ywContext.SwaggerAuth)
				Expect(err).NotTo(HaveOccurred())
				err = cmdutil.WaitForTaskCompletion(context.Background(), ywContext.YugawareClient, response.GetPayload())
				Expect(err).NotTo(HaveOccurred())

				universe = GetTestUniverse()
			})
			It("The universe increases its node count", func() {
				Expect(universe.UniverseDetails.Clusters[0].UserIntent.NumNodes).To(BeNumerically("==", newNodeCount))

				var tservers []*models.NodeDetailsResp
				for _, node := range universe.UniverseDetails.NodeDetailsSet {
					if node.IsTserver {
						tservers = append(tservers, node)
					}
				}

				Expect(tservers).To(HaveLen(int(newNodeCount)))
			})
		})
	})

	Context("UniverseManagement API", func() {
		When("Resetting the universe version", func() {
			var resetResponse *models.YBPSuccess
			BeforeEach(func() {
				universe := CreateTestUniverseIfNotExists()

				// This requires a dummy value to work around https://yugabyte.atlassian.net/browse/PLAT-2076
				params := universe_management.NewResetUniverseVersionParams().
					WithCUUID(ywContext.CustomerUUID()).
					WithUniUUID(universe.UniverseUUID).
					WithDummy(&models.DummyBody{})

				response, err := ywContext.PlatformAPIs.UniverseManagement.ResetUniverseVersion(params, ywContext.SwaggerAuth)
				Expect(err).NotTo(HaveOccurred())

				resetResponse = response.GetPayload()
			})
			It("The universe version is properly reset", func() {
				Expect(*resetResponse.Success).To(BeTrue())
			})
		})

	})
})
