package yugaware_client_test

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/yugabyte/yb-tools/pkg/ybversion"
	"github.com/yugabyte/yb-tools/yugaware-client/pkg/client/swagger/models"
)

var _ = Describe("yugaware-client backup integration tests", func() {
	var (
		universe        *models.UniverseResp
		universeVersion ybversion.YBVersion
	)
	BeforeEach(func() {
		err := Provider.ConfigureIfNotExists(ywContext, Options.ProviderName)
		Expect(err).NotTo(HaveOccurred())

		universe = CreateTestUniverseIfNotExists()
		universeVersion = ybversion.MustParse(universe.UniverseDetails.Clusters[0].UserIntent.YbSoftwareVersion)
	})
	When("no storage provider is configured", func() {

	})
	When(fmt.Sprintf("a %s storage provider is configured", Options.StorageProviderType), func() {
		BeforeEach(func() {
			err := Storage.ConfigureIfNotExists(ywContext, Options.StorageProviderName)
			Expect(err).NotTo(HaveOccurred())
		})
		Context("backup create", func() {
			When("no universe is defined", func() {
				It("returns an error", func() {
					_, err := ywContext.RunYugawareCommand("backup", "create")
					Expect(err).To(MatchError("accepts 1 arg(s), received 0"))
				})
			})
			When("a non-extant universe is defined", func() {
				It("returns an error", func() {
					_, err := ywContext.RunYugawareCommand("backup", "create", "--storage-config", Options.StorageProviderName, "non-extant-universe")
					Expect(err).To(MatchError(`unable to validate universe name "non-extant-universe": universe does not exist`))
				})
			})
			When("storage-config is not set", func() {
				It("returns an error", func() {
					_, err := ywContext.RunYugawareCommand("backup", "create", universe.Name)
					Expect(err).To(MatchError(`required flag(s) [storage-config] not set`))
				})
			})
			When("storage is set to a non-extant storage config", func() {
				It("returns an error", func() {
					_, err := ywContext.RunYugawareCommand("backup", "create", universe.Name, "--storage-config", "non-extant")
					Expect(err).To(MatchError(`storage config "non-extant" does not exist`))
				})
			})
			When("the database and cql flags are used together", func() {
				It("returns an error", func() {
					_, err := ywContext.RunYugawareCommand("backup", "create", universe.Name,
						"--storage-config", Options.StorageProviderName, "--type", "cql", "--database", "non-extant")

					Expect(err).To(MatchError(`the "database" flag is incompatible with a CQL backup`))
				})
			})
			When("the keyspace and sql flags are used together", func() {
				It("returns an error", func() {
					_, err := ywContext.RunYugawareCommand("backup", "create", universe.Name,
						"--storage-config", Options.StorageProviderName, "--type", "sql", "--keyspace", "non-extant")

					Expect(err).To(MatchError(`the "keyspace" flag is incompatible with an SQL backup`))
				})
			})
		})
		Context("sql backup integration tests", func() {
			var backupUUID string
			When("an sql schema is created in the database testdb", func() {
				dbname := "testdb"
				BeforeEach(func() {
					setupSQLSchema(universe, dbname)
				})

				It("can create a backup of the database", func() {
					output, err := ywContext.RunYugawareCommand("backup", "create", universe.Name, "--type", "sql",
						"--storage-config", Options.StorageProviderName, "--database", dbname, "-o", "json", "--wait")

					Expect(err).NotTo(HaveOccurred())

					backupCreateOutput := &BackupCreateOutput{}
					err = json.Unmarshal(output, backupCreateOutput)
					Expect(err).NotTo(HaveOccurred())

					Expect(backupCreateOutput.Content).To(HaveLen(1))
					backupUUID = string(backupCreateOutput.Content[0].BackupUUID)
					Expect(backupUUID).NotTo(BeEmpty())
				})

				It("can list backups", func() {
					output, err := ywContext.RunYugawareCommand("backup", "list", universe.Name, "-o", "json")

					Expect(err).NotTo(HaveOccurred())
					Expect(output).To(ContainSubstring(backupUUID))
				})

				// TODO: change schema for backup
				XIt("can restore the backup to a different schema", func() {

				})

				It("cannot restore the backup to the existing schema", func() {
					_, err := ywContext.RunYugawareCommand("backup", "restore", backupUUID, "-o", "json", "--wait")
					if universeVersion.Lt(ybversion.YBVersion{Major: 2, Minor: 13}) {
						Expect(err.Error()).To(ContainSubstring("restore is only supported in version 2.13.0 and above"))
					} else {
						Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(`database "%s" already exists`, dbname)))
					}
				})

				When("the original table has been deleted", func() {
					It("can restore the backup", func() {
						deleteSQLSchema(universe, dbname)

						_, err := ywContext.RunYugawareCommand("backup", "restore", backupUUID, "-o", "json", "--wait")
						if universeVersion.Lt(ybversion.YBVersion{Major: 2, Minor: 13}) {
							Expect(err.Error()).To(ContainSubstring("restore is only supported in version 2.13.0 and above"))
						} else {
							Expect(err).NotTo(HaveOccurred())
						}
					})
				})
				// TODO: delete backup command
				XIt("can delete the backups", func() {

				})

				When("specifying a table in an sql backup", func() {
					It("returns an error", func() {
						_, err := ywContext.RunYugawareCommand("backup", "create", universe.Name, "--table", "foo",
							"--storage-config", Options.StorageProviderName, "--type", "sql", "--database", "non-extant")

						Expect(err).To(MatchError(`table level SQL backups are unsupported`))
					})
				})

				When("creating a backup for an sql keyspace that doesn't exist", func() {
					It("returns an error", func() {
						_, err := ywContext.RunYugawareCommand("backup", "create", universe.Name,
							"--storage-config", Options.StorageProviderName, "--type", "sql", "--database", "non-extant")

						// Version 2.8 and below do not check keyspace for tables before starting a backup
						if universeVersion.Lt(ybversion.YBVersion{Major: 2, Minor: 9}) {
							// Various 2.8 versions have different error codes, so rather than collect them all just
							// expect an error
							Expect(err).To(HaveOccurred())
						} else {
							Expect(err.Error()).To(ContainSubstring(`Cannot initiate backup with empty Keyspace non-extant`))
						}
					})
				})
			})
		})
		Context("cql backup tests", func() {
			var (
				keyspaceName = "testkeyspace"
				backupUUID   string
			)
			BeforeEach(func() {
				setupCQLSchema(universe, keyspaceName)
			})
			It("can create a backup of the keyspace", func() {
				output, err := ywContext.RunYugawareCommand("backup", "create", universe.Name, "--type", "cql",
					"--storage-config", Options.StorageProviderName, "--keyspace", keyspaceName, "-o", "json", "--wait")

				Expect(err).NotTo(HaveOccurred())

				backupCreateOutput := &BackupCreateOutput{}
				err = json.Unmarshal(output, backupCreateOutput)
				Expect(err).NotTo(HaveOccurred())

				Expect(backupCreateOutput.Content).To(HaveLen(1))
				backupUUID = string(backupCreateOutput.Content[0].BackupUUID)
				Expect(backupUUID).NotTo(BeEmpty())
			})

			It("can take a backup of a table", func() {
				output, err := ywContext.RunYugawareCommand("backup", "create", universe.Name, "--type", "cql",
					"--storage-config", Options.StorageProviderName, "--keyspace", keyspaceName, "-o", "json", "--wait",
					"--table", "foo")

				Expect(err).NotTo(HaveOccurred())

				backupCreateOutput := &BackupCreateOutput{}
				err = json.Unmarshal(output, backupCreateOutput)
				Expect(err).NotTo(HaveOccurred())

				Expect(backupCreateOutput.Content).To(HaveLen(1))
				backupUUID = string(backupCreateOutput.Content[0].BackupUUID)
				Expect(backupUUID).NotTo(BeEmpty())

			})

			It("can list backups", func() {
				output, err := ywContext.RunYugawareCommand("backup", "list", universe.Name, "-o", "json")

				Expect(err).NotTo(HaveOccurred())
				Expect(output).To(ContainSubstring(backupUUID))
			})

			// TODO: change schema for backup
			XIt("can restore the backup to a different schema", func() {
			})

			It("cannot restore the backup to the existing schema", func() {
				_, err := ywContext.RunYugawareCommand("backup", "restore", backupUUID, "-o", "json", "--wait")
				if universeVersion.Lt(ybversion.YBVersion{Major: 2, Minor: 13}) {
					Expect(err.Error()).To(ContainSubstring("restore is only supported in version 2.13.0 and above"))
				} else {
					Expect(err.Error()).To(ContainSubstring(fmt.Sprintf(`keyspace "%s" already exists`, keyspaceName)))
				}
			})

			When("the original table has been deleted", func() {
				It("can restore the backup", func() {
					deleteCQLSchema(universe, "testkeyspace")

					_, err := ywContext.RunYugawareCommand("backup", "restore", backupUUID, "-o", "json", "--wait")
					if universeVersion.Lt(ybversion.YBVersion{Major: 2, Minor: 13}) {
						Expect(err.Error()).To(ContainSubstring("restore is only supported in version 2.13.0 and above"))
					} else {
						Expect(err).NotTo(HaveOccurred())
					}
				})
			})
			// TODO: delete backup command
			XIt("can delete the backups", func() {

			})

			When("creating a backup for a cql keyspace that doesn't exist", func() {
				It("returns an error", func() {
					_, err := ywContext.RunYugawareCommand("backup", "create", universe.Name,
						"--storage-config", Options.StorageProviderName, "--type", "cql", "--keyspace", "non-extant")

					// Version 2.8 and below do not check keyspace for tables before starting a backup
					if universeVersion.Lt(ybversion.YBVersion{Major: 2, Minor: 9}) {
						// Various 2.8 versions have different error codes, so rather than collect them all just
						// expect an error
						Expect(err).To(HaveOccurred())
					} else {
						Expect(err.Error()).To(ContainSubstring(`Cannot initiate backup with empty Keyspace non-extant`))
					}
				})
			})
		})
	})

})

func setupSQLSchema(universe *models.UniverseResp, database string) {
	ywContext.CreateYSQLDatabase(universe.Name, database)

	sqlConn := ywContext.YSQLConnection(universe.Name, database)
	defer sqlConn.Close()

	_, err := sqlConn.Exec("create table if not exists backuptest(a int primary key, b int)")
	Expect(err).NotTo(HaveOccurred())
}

func setupCQLSchema(universe *models.UniverseResp, keyspace string) {
	yqlConn := ywContext.YCQLConnection(universe.Name)
	defer yqlConn.Close()

	err := yqlConn.Query(fmt.Sprintf("create keyspace if not exists %s", keyspace)).Exec()
	Expect(err).NotTo(HaveOccurred())

	err = yqlConn.Query(fmt.Sprintf("create table if not exists %s.foo(a int primary key, b int)", keyspace)).Exec()
	Expect(err).NotTo(HaveOccurred())
}

func deleteSQLSchema(universe *models.UniverseResp, database string) {
	ywContext.DropYSQLDatabase(universe.Name, database)
}

func deleteCQLSchema(universe *models.UniverseResp, keyspace string) {
	yqlConn := ywContext.YCQLConnection(universe.Name)
	defer yqlConn.Close()

	err := yqlConn.Query(fmt.Sprintf("drop table if exists %s.foo", keyspace)).Exec()
	Expect(err).NotTo(HaveOccurred())

	err = yqlConn.Query(fmt.Sprintf("drop keyspace if exists %s", keyspace)).Exec()
	Expect(err).NotTo(HaveOccurred())
}

type BackupCreateOutput struct {
	Message string           `json:"msg"`
	Content []*models.Backup `json:"content"`
}
