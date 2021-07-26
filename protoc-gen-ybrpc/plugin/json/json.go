package json

import (
	"github.com/yugabyte/yb-tools/protoc-gen-ybrpc/pkg/util"
	"google.golang.org/protobuf/compiler/protogen"
)

func Generate(gen *protogen.Plugin, file *protogen.File) {
	if len(file.Messages) > 0 {
		generateFile(gen, file)
	}
}

// generateFile generates a _ascii.pb.go file containing gRPC service definitions.
func generateFile(gen *protogen.Plugin, file *protogen.File) *protogen.GeneratedFile {
	filename := file.GeneratedFilenamePrefix + ".pb.json.go"
	g := gen.NewGeneratedFile(filename, file.GoImportPath)

	util.GenerateHeader(g, file)

	generateImports(g)

	for _, message := range file.Messages {
		generateMarshaler(g, message)

		generateUnmarshaler(g, message)
	}
	g.P()

	return g
}

// TODO: This statically outputs imports- how is this _actually_ supposed to be implemened?
func generateImports(g *protogen.GeneratedFile) {
	g.P()
	g.P(`import (`)
	g.P(`    "google.golang.org/protobuf/encoding/protojson"`)
	g.P(`)`)
	g.P()
}

func generateMarshaler(g *protogen.GeneratedFile, message *protogen.Message) {
	g.P(`func (m *`, message.GoIdent.GoName, `) MarshalJSON() ([]byte, error) {`)
	g.P(`    return protojson.MarshalOptions{}.Marshal(m)`)
	g.P(`}`)
	g.P("")
}

func generateUnmarshaler(g *protogen.GeneratedFile, message *protogen.Message) {
	g.P(`func (m *`, message.GoIdent.GoName, `) UnmarshalJSON(b []byte) error {`)
	g.P(`    return protojson.UnmarshalOptions{}.Unmarshal(b, m)`)
	g.P("}")
	g.P("")
}
