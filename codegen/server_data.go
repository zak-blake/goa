package codegen

import (
	"strings"

	"goa.design/goa/expr"
)

// Servers holds the server data computed from the design needed to generate
// the example server and client.
var Servers = make(ServersData)

type (
	// ServersData holds the server data from the service design indexed by
	// server name.
	ServersData map[string]*ServerData

	// ServerData contains the data about a single server.
	ServerData struct {
		// Name is the server name.
		Name string
		// Description is the server description.
		Description string
		// Services is the list of services supported by the server.
		Services []string
		// Schemes is the list of supported schemes by the server.
		Schemes []string
		// Hosts is the list of hosts defined in the server.
		Hosts []*HostData
		// Variables is the list of URL parameters defined in every host.
		Variables []*VariableData
	}

	// HostData contains the data about a single host in a server.
	HostData struct {
		// Name is the host name.
		Name string
		// Description is the host description.
		Description string
		// Schemes is the list of schemes supported by the host.
		Schemes []string
		// URIs is the list of URLs defined in the host.
		URIs []*URIData
		// Variables is the list of URL parameters.
		Variables []*VariableData
	}

	// VariableData contains the data about a URL variable.
	VariableData struct {
		// Name is the name of the variable.
		Name string
		// Description is the variable description.
		Description string
		// VarName is the variable name used in generating flag variables.
		VarName string
		// DefaultValue is the default value for the variable. It is set to the
		// default value defined in the variable attribute if exists, or else set
		// to the first value in the enum expression.
		DefaultValue interface{}
	}

	// URIData contains the data about a URL.
	URIData struct {
		// URL is the underlying URL.
		URL string
		// Scheme is the URL scheme.
		Scheme string
		// Transport is the transport type for the URL.
		Transport Transport
	}

	// Transport is type for supported goa transports.
	Transport string
)

const (
	// TransportHTTP is the HTTP transport.
	TransportHTTP Transport = "http"
	// TransportGRPC is the gRPC transport.
	TransportGRPC = "grpc"
)

// Get returns the server data for the given server expression.
func (d ServersData) Get(svr *expr.ServerExpr) *ServerData {
	if data, ok := d[svr.Name]; ok {
		return data
	}
	sd := buildServerData(svr)
	d[svr.Name] = sd
	return sd
}

// DefaultHost returns the first host defined in the server expression.
func (s *ServerData) DefaultHost() *HostData {
	return s.Hosts[0]
}

// AvailableHosts returns a list of available host names.
func (s *ServerData) AvailableHosts() []string {
	hosts := make([]string, 0, len(s.Hosts))
	for _, h := range s.Hosts {
		hosts = append(hosts, h.Name)
	}
	return hosts
}

// URL returns the first URL defined for the given transport.
func (h *HostData) URL(transport Transport) string {
	for _, u := range h.URIs {
		if u.Transport == transport {
			return u.URL
		}
	}
	return ""
}

func buildServerData(svr *expr.ServerExpr) *ServerData {
	var (
		hosts     []*HostData
		variables []*VariableData
	)
	{
		for _, h := range svr.Hosts {
			hosts = append(hosts, buildHostData(h))
		}
		foundVars := make(map[string]struct{})
		for _, h := range hosts {
			for _, v := range h.Variables {
				if _, ok := foundVars[v.Name]; ok {
					continue
				}
				variables = append(variables, v)
				foundVars[v.Name] = struct{}{}
			}
		}
	}
	return &ServerData{
		Name:        svr.Name,
		Description: svr.Description,
		Services:    svr.Services,
		Schemes:     svr.Schemes(),
		Hosts:       hosts,
		Variables:   variables,
	}
}

func buildHostData(host *expr.HostExpr) *HostData {
	var (
		uris      []*URIData
		variables []*VariableData
	)
	{
		uris = make([]*URIData, len(host.URIs))
		for i, uv := range host.URIs {
			var (
				t      Transport
				scheme string

				ustr = string(uv)
			)
			{
				// Did not use url package to find scheme because the url may
				// contain params (i.e. http://{version}.example.com) which needs
				// substition for url.Parse to succeed. Also URIs in host must have
				// a scheme otherwise validations would have failed.
				switch {
				case strings.HasPrefix(ustr, "https"):
					scheme = "https"
					t = TransportHTTP
				case strings.HasPrefix(ustr, "http"):
					scheme = "http"
					t = TransportHTTP
				case strings.HasPrefix(ustr, "grpcs"):
					scheme = "grpcs"
					t = TransportGRPC
				case strings.HasPrefix(ustr, "grpc"):
					scheme = "grpc"
					t = TransportGRPC

					// No need for default case here because we only support the above
					// possibilites for the scheme. Invalid scheme would have failed
					// validations in the first place.
				}
			}
			uris[i] = &URIData{
				Scheme:    scheme,
				URL:       ustr,
				Transport: t,
			}
		}
		vars := expr.AsObject(host.Variables.Type)
		if len(*vars) > 0 {
			variables = make([]*VariableData, len(*vars))
			for i, v := range *vars {
				def := v.Attribute.DefaultValue
				if def == nil {
					// DSL ensures v.Attribute has either a
					// default value or an enum validation
					def = v.Attribute.Validation.Values[0]
				}
				variables[i] = &VariableData{
					Name:         v.Name,
					Description:  v.Attribute.Description,
					VarName:      Goify(v.Name, false),
					DefaultValue: def,
				}
			}
		}
	}
	return &HostData{
		Name:        host.Name,
		Description: host.Description,
		Schemes:     host.Schemes(),
		URIs:        uris,
		Variables:   variables,
	}
}
