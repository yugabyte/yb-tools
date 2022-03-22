package format_test

import (
	"bytes"
	json "encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/yugabyte/yb-tools/pkg/format"
)

var _ = Describe("format table unit tests", func() {
	var (
		outputBuffer *bytes.Buffer
		jsonObject   interface{}
		output       *format.Output
	)
	BeforeEach(func() {
		outputBuffer = new(bytes.Buffer)
		format.SetOut(outputBuffer)

		jsonObject = new(interface{})
	})
	Context("error path", func() {
		var (
			jsonInput string
			err       error
		)

		BeforeEach(func() {
			jsonInput = "{}"
			output = &format.Output{
				OutputType: "table",
			}
		})
		JustBeforeEach(func() {
			err = json.Unmarshal([]byte(jsonInput), jsonObject)
			Expect(err).NotTo(HaveOccurred())

			output.JSONObject = jsonObject

			err = output.Print()
		})
		When("no columns are set", func() {
			It("returns an error", func() {
				Expect(err).To(MatchError("no output columns have been set"))
			})
		})
		When("no output type is set on the table", func() {
			BeforeEach(func() {
				jsonInput = `{"value": true}`
				output.TableColumns = []format.Column{{
					Name:     "ColumnHeader",
					JSONPath: "$.value",
				}}
			})
			It("prints a table", func() {
				table := " ColumnHeader \n"
				table += " true         \n"
				Expect(outputBuffer.String()).To(Equal(table))
			})
		})
		When("no output type is set on the table", func() {
			BeforeEach(func() {
				output.OutputType = "fake"
				output.TableColumns = []format.Column{{}}
			})
			It("returns an error", func() {
				Expect(err).To(MatchError("unsupported output type: fake"))
			})
		})
		When("no column name is specified", func() {
			BeforeEach(func() {
				output.TableColumns = []format.Column{{}}
			})
			It("returns an error", func() {
				Expect(err).To(MatchError(`unable to format path row[1] {}: no expression or jsonpath set for column [1]`))
			})
		})
		When("no expression is set in the table", func() {
			BeforeEach(func() {
				output.TableColumns = []format.Column{{Name: "Column1"}}
			})
			It("returns an error", func() {
				Expect(err).To(MatchError(`unable to format path row[1] {}: no expression or jsonpath set for column Column1[1]`))
			})
		})

		When("an object is evaluated", func() {
			BeforeEach(func() {
				jsonInput = `{"test":666}`
				output.TableColumns = []format.Column{{Name: "Column1", JSONPath: "$.test"}}
			})

			When("output type is table", func() {
				It("prints a table", func() {
					Expect(outputBuffer).To(ContainSubstring("Column1"))
					Expect(outputBuffer).To(ContainSubstring("666"))
					Expect(LineCount(outputBuffer)).To(Equal(1))
				})
			})

			When("A filter has been set", func() {
				It("prints a table", func() {
					Expect(outputBuffer).To(ContainSubstring("Column1"))
					Expect(outputBuffer).To(ContainSubstring("666"))
					Expect(LineCount(outputBuffer)).To(Equal(1))
				})
			})
		})

		When("a list is evaluated", func() {
			BeforeEach(func() {
				jsonInput = `[{"test":666,"test2":"string"},{"test":667,"test3":52.6},{"test":668}]`
				output.TableColumns = []format.Column{{Name: "Column1", JSONPath: "$.test"}}
			})

			When("output type is table", func() {
				It("prints a table", func() {
					Expect(outputBuffer).To(ContainSubstring("Column1"))
					Expect(outputBuffer).To(ContainSubstring("666"))
					Expect(outputBuffer).To(ContainSubstring("667"))
					Expect(outputBuffer).To(ContainSubstring("668"))
					Expect(LineCount(outputBuffer)).To(Equal(3))
				})
			})

			When("A filter has been set", func() {
				BeforeEach(func() {
					output.Filter = `@.test > 666`
				})
				It("prints a table", func() {
					Expect(outputBuffer).To(ContainSubstring("Column1"))
					Expect(outputBuffer).NotTo(ContainSubstring("666"))
					Expect(outputBuffer).To(ContainSubstring("667"))
					Expect(outputBuffer).To(ContainSubstring("668"))
					Expect(LineCount(outputBuffer)).To(Equal(2))
				})

				When("output type is json", func() {
					BeforeEach(func() {
						output.OutputType = "json"
					})
					It("prints a table", func() {
						Expect(outputBuffer).NotTo(ContainSubstring("666"))
						Expect(outputBuffer).To(ContainSubstring("667"))
						Expect(outputBuffer).To(ContainSubstring("668"))
					})
				})
				When("output type is yaml", func() {
					BeforeEach(func() {
						output.OutputType = "yaml"
					})
					It("prints a table", func() {
						Expect(outputBuffer).NotTo(ContainSubstring("666"))
						Expect(outputBuffer).To(ContainSubstring("667"))
						Expect(outputBuffer).To(ContainSubstring("668"))
					})
				})
			})
		})
	})
})

func LineCount(table *bytes.Buffer) int {
	return bytes.Count(table.Bytes(), []byte("\n")) - 1
}
