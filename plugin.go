package tfsdk

import (
	"context"
	"fmt"
	"net/rpc"

	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	grpcCodes "google.golang.org/grpc/codes"

	"github.com/apparentlymart/terraform-sdk/internal/tfplugin5"
)

// ServeProviderPlugin starts a plugin server for the given provider, which will
// first deal with the plugin protocol handshake and then, once initialized,
// serve RPC requests from the client (usually Terraform CLI).
//
// This should be called in the main function for the plugin program.
// ServeProviderPlugin returns only once the plugin has been requested to exit
// by its client.
func ServeProviderPlugin(p *Provider) {
	impls := map[int]plugin.PluginSet{
		4: {
			"provider": unsupportedProtocolVersion4{},
		},
		5: {
			"provider": protocolVersion5{p},
		},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig:  pluginHandshake,
		VersionedPlugins: impls,
		GRPCServer:       plugin.DefaultGRPCServer,
	})
}

func (p *Provider) tfplugin5Server() tfplugin5.ProviderServer {
	// This single shared context will be passed (directly or indirectly) to
	// each provider method that can make network requests and cancelled if
	// the Terraform operation recieves an interrupt request.
	ctx, cancel := context.WithCancel(context.Background())

	return &tfplugin5Server{
		p:    p,
		ctx:  ctx,
		stop: cancel,
	}
}

type tfplugin5Server struct {
	p    *Provider
	ctx  context.Context
	stop func()
}

func (s *tfplugin5Server) GetSchema(context.Context, *tfplugin5.GetProviderSchema_Request) (*tfplugin5.GetProviderSchema_Response, error) {
	return nil, grpc.Errorf(grpcCodes.Unimplemented, "not implemented")
}

func (s *tfplugin5Server) PrepareProviderConfig(context.Context, *tfplugin5.PrepareProviderConfig_Request) (*tfplugin5.PrepareProviderConfig_Response, error) {
	return nil, grpc.Errorf(grpcCodes.Unimplemented, "not implemented")
}

func (s *tfplugin5Server) ValidateResourceTypeConfig(context.Context, *tfplugin5.ValidateResourceTypeConfig_Request) (*tfplugin5.ValidateResourceTypeConfig_Response, error) {
	return nil, grpc.Errorf(grpcCodes.Unimplemented, "not implemented")
}

func (s *tfplugin5Server) ValidateDataSourceConfig(context.Context, *tfplugin5.ValidateDataSourceConfig_Request) (*tfplugin5.ValidateDataSourceConfig_Response, error) {
	return nil, grpc.Errorf(grpcCodes.Unimplemented, "not implemented")
}

func (s *tfplugin5Server) UpgradeResourceState(context.Context, *tfplugin5.UpgradeResourceState_Request) (*tfplugin5.UpgradeResourceState_Response, error) {
	return nil, grpc.Errorf(grpcCodes.Unimplemented, "not implemented")
}

func (s *tfplugin5Server) Configure(context.Context, *tfplugin5.Configure_Request) (*tfplugin5.Configure_Response, error) {
	return nil, grpc.Errorf(grpcCodes.Unimplemented, "not implemented")
}

func (s *tfplugin5Server) ReadResource(context.Context, *tfplugin5.ReadResource_Request) (*tfplugin5.ReadResource_Response, error) {
	return nil, grpc.Errorf(grpcCodes.Unimplemented, "not implemented")
}

func (s *tfplugin5Server) PlanResourceChange(context.Context, *tfplugin5.PlanResourceChange_Request) (*tfplugin5.PlanResourceChange_Response, error) {
	return nil, grpc.Errorf(grpcCodes.Unimplemented, "not implemented")
}

func (s *tfplugin5Server) ApplyResourceChange(context.Context, *tfplugin5.ApplyResourceChange_Request) (*tfplugin5.ApplyResourceChange_Response, error) {
	return nil, grpc.Errorf(grpcCodes.Unimplemented, "not implemented")
}

func (s *tfplugin5Server) ImportResourceState(context.Context, *tfplugin5.ImportResourceState_Request) (*tfplugin5.ImportResourceState_Response, error) {
	return nil, grpc.Errorf(grpcCodes.Unimplemented, "not implemented")
}

func (s *tfplugin5Server) ReadDataSource(context.Context, *tfplugin5.ReadDataSource_Request) (*tfplugin5.ReadDataSource_Response, error) {
	return nil, grpc.Errorf(grpcCodes.Unimplemented, "not implemented")
}

func (s *tfplugin5Server) Stop(context.Context, *tfplugin5.Stop_Request) (*tfplugin5.Stop_Response, error) {
	// This cancels our server's root context, in the hope that the provider
	// operations will respond to this by safely cancelling their in-flight
	// actions and returning (possibly with an error) as quickly as possible.
	s.stop()
	return &tfplugin5.Stop_Response{}, nil
}

// protocolVersion5 is an implementation of both plugin.Plugin and
// plugin.GRPCPlugin that implements protocol version 5.
type protocolVersion5 struct {
	p *Provider
}

var _ plugin.GRPCPlugin = protocolVersion5{}

func (p protocolVersion5) GRPCClient(context.Context, *plugin.GRPCBroker, *grpc.ClientConn) (interface{}, error) {
	return nil, fmt.Errorf("Terraform SDK can only be used to implement plugin servers, not plugin clients")
}

func (p protocolVersion5) GRPCServer(broker *plugin.GRPCBroker, server *grpc.Server) error {
	tfplugin5.RegisterProviderServer(server, p.p.tfplugin5Server())
	return nil
}

func (p protocolVersion5) Client(*plugin.MuxBroker, *rpc.Client) (interface{}, error) {
	return nil, fmt.Errorf("net/rpc is not valid in protocol version 5")
}

func (p protocolVersion5) Server(*plugin.MuxBroker) (interface{}, error) {
	return nil, fmt.Errorf("net/rpc is not valid in protocol version 5")
}

// unsupportedProtocolVersion4 is an implementation of plugin.Plugin that just
// returns an error stating that the plugin requires Terraform v0.12.0 or later.
type unsupportedProtocolVersion4 struct{}

func (p unsupportedProtocolVersion4) Client(*plugin.MuxBroker, *rpc.Client) (interface{}, error) {
	return nil, fmt.Errorf("Terraform SDK can only be used to implement plugin servers, not plugin clients")
}

func (p unsupportedProtocolVersion4) Server(*plugin.MuxBroker) (interface{}, error) {
	return nil, fmt.Errorf("this plugin requires Terraform v0.12.0 or later")
}

var pluginHandshake = plugin.HandshakeConfig{
	// ProtocolVersion is a legacy setting used to set the default protocol
	// version for clients (old Terraform versions) that do not explicitly
	// specify which protocol versions they support.
	//
	// Protocol version 4 is no longer supported, so in practice any client
	// that does not explicitly select a later version is automatically
	// incompatible with plugins compiled with this SDK version.
	//
	// (The VersionedPlugins field passed to plugin.Serve above is what
	// actually handles our version negotation.)
	ProtocolVersion: 4,

	// The magic cookie values must not be changed, or else a plugin will
	// not correctly recognize that is running as a child process of Terraform.
	MagicCookieKey:   "TF_PLUGIN_MAGIC_COOKIE",
	MagicCookieValue: "d602bf8f470bc67ca7faa0386276bbdd4330efaf76d1a219cb4d6991ca9872b2",
}
