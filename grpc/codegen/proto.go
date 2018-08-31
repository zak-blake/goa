package codegen

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"goa.design/goa/codegen"
	"goa.design/goa/expr"
)

// ProtoFiles returns a *.proto file for each gRPC service.
func ProtoFiles(genpkg string, root *expr.RootExpr) {
	for _, svc := range root.API.GRPC.Services {
		f := protoFile(genpkg, svc)
		// Render the .proto file to the disk
		if _, err := f.Render("."); err != nil {
			panic(err)
		}
		protoc(f.Path)
	}
}

func protoFile(genpkg string, svc *expr.GRPCServiceExpr) *codegen.File {
	svcName := codegen.SnakeCase(svc.Name())
	path := filepath.Join(codegen.Gendir, "grpc", svcName, svcName+".proto")
	data := GRPCServices.Get(svc.Name())

	title := fmt.Sprintf("%s protocol buffer definition", svc.Name())
	sections := []*codegen.SectionTemplate{
		Header(title, svc.Name(), []*codegen.ImportSpec{}),
		&codegen.SectionTemplate{
			Name:   "grpc-service",
			Source: serviceT,
			Data:   data,
		},
	}

	for _, m := range data.Messages {
		sections = append(sections, &codegen.SectionTemplate{
			Name:   "grpc-message",
			Source: messageT,
			Data:   m,
		})
	}

	return &codegen.File{Path: path, SectionTemplates: sections}
}

func protoc(path string) {
	args := []string{"--go_out=plugins=grpc:.", path}
	// Run protoc compiler with the protoc-gen-go plugin
	cmd := exec.Command("protoc", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Println(fmt.Sprintf("Error running protoc command:\n%s", string(output)))
		panic(err)
	}
}

const (
	// input: ServiceData
	serviceT = `{{ .Description | comment }}
service {{ .Name }} {
	{{- range .Endpoints }}
	{{ if .Method.Description }}{{ .Method.Description | comment }}{{ end }}
	rpc {{ .Method.VarName }} ({{ .Request.Message.Name }}) returns ({{ .Response.Message.Name }});
	{{- end }}
}
`

	// input: TypeData
	messageT = `{{ comment .Description }}
message {{ .VarName }}{{ .Def }}
`
)
