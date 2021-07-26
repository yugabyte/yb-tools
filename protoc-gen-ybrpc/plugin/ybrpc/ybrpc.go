package ybrpc

import (
	"github.com/yugabyte/yb-tools/protoc-gen-ybrpc/pkg/util"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/descriptorpb"
)

func Generate(gen *protogen.Plugin, file *protogen.File) {
	if len(file.Services) > 0 {
		generate := false
		for _, service := range file.Services {
			if len(service.Methods) > 0 {
				generate = true
			}
		}
		if generate {
			_ = generateFile(gen, file)
		}
	}
}

// generateFile generates a _ascii.pb.go file containing gRPC service definitions.
func generateFile(gen *protogen.Plugin, file *protogen.File) *protogen.GeneratedFile {
	filename := file.GeneratedFilenamePrefix + ".pb.ybrpc.go"
	g := gen.NewGeneratedFile(filename, file.GoImportPath)

	util.GenerateHeader(g, file)

	generateImports(g)

	for _, service := range file.Services {
		generateService(g, service)

		generateServiceImpl(g, service)

		for _, method := range service.Methods {
			generateServiceMethod(g, service, method)
		}
	}
	g.P()

	return g
}

// TODO: This statically outputs imports- how is this _actually_ supposed to be implemened?
func generateImports(g *protogen.GeneratedFile) {
	g.P()
	g.P(`import (`)
	g.P(`    "github.com/go-logr/logr"`)
	g.P(`    "github.com/yugabyte/yb-tools/protoc-gen-ybrpc/pkg/message"`)
	g.P(`)`)
	g.P()
}

func generateService(g *protogen.GeneratedFile, service *protogen.Service) {
	g.P("// service: ", service.Desc.FullName())
	g.P("// service: ", service.GoName)

	util.GenerateComments(g, service.Comments, service.Desc.Options().(*descriptorpb.ServiceOptions).GetDeprecated())
	g.P("type ", service.GoName, " interface {")
	for _, method := range service.Methods {
		generateMethodSigniture(g, method)
	}
	g.P("}")
	g.P()
}

func generateMethodSigniture(g *protogen.GeneratedFile, method *protogen.Method) {
	g.P(method.GoName, "(request *", method.Input.GoIdent.GoName+")"+"(*"+method.Output.GoIdent.GoName+", error)")
}

func generateServiceImpl(g *protogen.GeneratedFile, service *protogen.Service) {
	g.P("type ", service.GoName, "Impl struct {")
	g.P("    Log logr.Logger")
	g.P("    Messenger message.Messenger")
	g.P("}")
	g.P()
}

func generateServiceMethod(g *protogen.GeneratedFile, service *protogen.Service, method *protogen.Method) {
	util.GenerateComments(g, method.Comments, method.Desc.Options().(*descriptorpb.MethodOptions).GetDeprecated())
	g.P("func (s *" + service.GoName + "Impl)" + method.GoName + "(request *" + method.Input.GoIdent.GoName + ")" + "(*" + method.Output.GoIdent.GoName + ", error) {")
	g.P(`    s.Log.V(1).Info("sending RPC message", "service", "`, string(service.Desc.FullName()), `", "method", "`, string(method.Desc.Name()), `", "message", request)`)
	g.P("    response := &" + method.Output.GoIdent.GoName + "{}")
	g.P()
	g.P(`    err := s.Messenger.SendMessage("`, string(service.Desc.FullName()), `", "`, string(method.Desc.Name()), `", request.ProtoReflect().Interface(), response.ProtoReflect().Interface())`)
	g.P("    if err != nil {")
	g.P("        return nil, err")
	g.P("    }")
	g.P()
	g.P(`    s.Log.V(1).Info("received RPC response", "service", "`, string(service.Desc.FullName()), `", "method", "`, string(method.Desc.Name()), `", "message", response)`)
	g.P()
	g.P("    return response, nil")
	g.P("}")
	g.P("")
}
