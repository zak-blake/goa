package codegen

import (
	"os"
	"path/filepath"
	"strings"

	"goa.design/goa/codegen"
	"goa.design/goa/expr"
)

// ExampleCLI returns an example gRPC client tool implementation.
func ExampleCLI(genpkg string, root *expr.RootExpr) *codegen.File {
	path := filepath.Join("cmd", codegen.SnakeCase(codegen.Goify(root.API.Name, true))+"_cli", "grpc.go")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return nil // file already exists, skip it.
	}
	idx := strings.LastIndex(genpkg, string(os.PathSeparator))
	rootPath := "."
	if idx > 0 {
		rootPath = genpkg[:idx]
	}
	apiPkg := strings.ToLower(codegen.Goify(root.API.Name, false))
	specs := []*codegen.ImportSpec{
		{Path: "context"},
		{Path: "encoding/json"},
		{Path: "flag"},
		{Path: "fmt"},
		{Path: "google.golang.org/grpc"},
		{Path: "os"},
		{Path: "time"},
		{Path: "goa.design/goa/grpc", Name: "goagrpc"},
		{Path: rootPath, Name: apiPkg},
		{Path: genpkg + "/grpc/cli"},
	}
	data := map[string]interface{}{
		"APIPkg":  apiPkg,
		"APIName": root.API.Name,
	}
	sections := []*codegen.SectionTemplate{
		codegen.Header("", "main", specs),
		&codegen.SectionTemplate{
			Name:   "do-grpc-cli",
			Source: doGRPCT,
			Data:   data,
		},
	}
	return &codegen.File{Path: path, SectionTemplates: sections}
}

// input: map[string]interface{}{"APIPkg": string, "APIName": string}
const doGRPCT = `func grpcDo(addr string, timeout int, debug bool) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
    fmt.Fprintln(os.Stderr, fmt.Sprintf("could not connect to GRPC server at %s: %v", addr, err))
  }
	defer conn.Close()

	endpoint, payload, err := cli.ParseEndpoint(conn)
	if err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, err.Error())
		fmt.Fprintln(os.Stderr, "run '"+os.Args[0]+" --help' for detailed usage.")
		os.Exit(1)
	}

	data, err := endpoint(context.Background(), payload)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if data != nil && !debug {
		m, _ := json.MarshalIndent(data, "", "    ")
		fmt.Println(string(m))
	}
}

func grpcUsageCommands() string {
	return cli.UsageCommands()
}

func grpcUsageExamples() string {
	return cli.UsageExamples()
}
`
