package tfsdk

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/apparentlymart/terraform-sdk/internal/tfplugin5"
	"go.rpcplugin.org/rpcplugin"
	"go.rpcplugin.org/rpcplugin/plugintrace"
	"google.golang.org/grpc"
	grpcCodes "google.golang.org/grpc/codes"
)

// ServeProviderPlugin starts a plugin server for the given provider, which will
// first deal with the plugin protocol handshake and then, once initialized,
// serve RPC requests from the client (usually Terraform CLI).
//
// This should be called in the main function for the plugin program.
// ServeProviderPlugin returns only once the plugin has been requested to exit
// by its client.
func ServeProviderPlugin(p *Provider) {
	ctx := plugintrace.WithServerTracer(context.Background(), &plugintrace.ServerTracer{
		Listening: func(addr net.Addr, tlsConfig *tls.Config, protoVersion int) {
			log.Printf("[INFO] provider plugin server (protocol %d) listening on %s", protoVersion, addr)
		},
	})

	err := rpcplugin.Serve(ctx, &rpcplugin.ServerConfig{
		Handshake: rpcplugin.HandshakeConfig{
			CookieKey:   "TF_PLUGIN_MAGIC_COOKIE",
			CookieValue: "d602bf8f470bc67ca7faa0386276bbdd4330efaf76d1a219cb4d6991ca9872b2",
		},
		ProtoVersions: map[int]rpcplugin.Server{
			5: protocolVersion5{p},
		},
	})

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
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
	resp := &tfplugin5.GetProviderSchema_Response{}

	resp.Provider = &tfplugin5.Schema{
		Block: convertSchemaBlockToTFPlugin5(s.p.ConfigSchema),
	}

	resp.ResourceSchemas = make(map[string]*tfplugin5.Schema)
	for name, rt := range s.p.ManagedResourceTypes {
		schema, version := rt.getSchema()
		resp.ResourceSchemas[name] = &tfplugin5.Schema{
			Version: version,
			Block:   convertSchemaBlockToTFPlugin5(schema),
		}
	}

	resp.DataSourceSchemas = make(map[string]*tfplugin5.Schema)
	for name, rt := range s.p.DataResourceTypes {
		schema := rt.getSchema()
		resp.DataSourceSchemas[name] = &tfplugin5.Schema{
			Block: convertSchemaBlockToTFPlugin5(schema),
		}
	}

	return resp, nil
}

// requireManagedResourceType is a helper to conveniently retrieve a particular
// managed resource type or produce an error message if it is invalid.
//
// The usage pattern for this method is:
//
//    var rt ManagedResourceType
//    	if rt = s.requireManagedResourceType(req.TypeName, &resp.Diagnostics); rt == nil {
//    	return resp, nil
//    }
func (s *tfplugin5Server) requireManagedResourceType(typeName string, diagsPtr *[]*tfplugin5.Diagnostic) ManagedResourceType {
	rt := s.p.managedResourceType(typeName)
	if rt == nil {
		var diags Diagnostics
		diags = diags.Append(Diagnostic{
			Severity: Error,
			Summary:  "Unsupported resource type",
			Detail:   fmt.Sprintf("This provider does not support managed resource type %q", typeName),
		})
		*diagsPtr = encodeDiagnosticsToTFPlugin5(diags)
	}
	return rt
}

// requireDataResourceType is a helper to conveniently retrieve a particular
// data resource type or produce an error message if it is invalid.
//
// The usage pattern for this method is:
//
//    var rt DataResourceType
//    	if rt = s.requireDataResourceType(req.TypeName, &resp.Diagnostics); rt == nil {
//    	return resp, nil
//    }
func (s *tfplugin5Server) requireDataResourceType(typeName string, diagsPtr *[]*tfplugin5.Diagnostic) DataResourceType {
	rt := s.p.dataResourceType(typeName)
	if rt == nil {
		var diags Diagnostics
		diags = diags.Append(Diagnostic{
			Severity: Error,
			Summary:  "Unsupported resource type",
			Detail:   fmt.Sprintf("This provider does not support data resource type %q", typeName),
		})
		*diagsPtr = encodeDiagnosticsToTFPlugin5(diags)
	}
	return rt
}

func (s *tfplugin5Server) PrepareProviderConfig(ctx context.Context, req *tfplugin5.PrepareProviderConfig_Request) (*tfplugin5.PrepareProviderConfig_Response, error) {
	resp := &tfplugin5.PrepareProviderConfig_Response{}

	proposedVal, diags := decodeTFPlugin5DynamicValue(req.Config, s.p.ConfigSchema)
	if diags.HasErrors() {
		resp.Diagnostics = encodeDiagnosticsToTFPlugin5(diags)
		return resp, nil
	}

	preparedVal, diags := s.p.prepareConfig(proposedVal)
	resp.PreparedConfig = encodeTFPlugin5DynamicValue(preparedVal, s.p.ConfigSchema)
	resp.Diagnostics = encodeDiagnosticsToTFPlugin5(diags)
	return resp, nil
}

func (s *tfplugin5Server) ValidateResourceTypeConfig(ctx context.Context, req *tfplugin5.ValidateResourceTypeConfig_Request) (*tfplugin5.ValidateResourceTypeConfig_Response, error) {
	resp := &tfplugin5.ValidateResourceTypeConfig_Response{}

	var rt ManagedResourceType
	if rt = s.requireManagedResourceType(req.TypeName, &resp.Diagnostics); rt == nil {
		return resp, nil
	}

	schema, _ := rt.getSchema()
	configVal, diags := decodeTFPlugin5DynamicValue(req.Config, schema)
	if diags.HasErrors() {
		resp.Diagnostics = encodeDiagnosticsToTFPlugin5(diags)
		return resp, nil
	}

	diags = rt.validate(configVal)
	resp.Diagnostics = encodeDiagnosticsToTFPlugin5(diags)
	return resp, nil
}

func (s *tfplugin5Server) ValidateDataSourceConfig(ctx context.Context, req *tfplugin5.ValidateDataSourceConfig_Request) (*tfplugin5.ValidateDataSourceConfig_Response, error) {
	resp := &tfplugin5.ValidateDataSourceConfig_Response{}

	var rt DataResourceType
	if rt = s.requireDataResourceType(req.TypeName, &resp.Diagnostics); rt == nil {
		return resp, nil
	}

	schema := rt.getSchema()
	configVal, diags := decodeTFPlugin5DynamicValue(req.Config, schema)
	if diags.HasErrors() {
		resp.Diagnostics = encodeDiagnosticsToTFPlugin5(diags)
		return resp, nil
	}

	diags = rt.validate(configVal)
	resp.Diagnostics = encodeDiagnosticsToTFPlugin5(diags)
	return resp, nil
}

func (s *tfplugin5Server) UpgradeResourceState(context.Context, *tfplugin5.UpgradeResourceState_Request) (*tfplugin5.UpgradeResourceState_Response, error) {
	return nil, grpc.Errorf(grpcCodes.Unimplemented, "not implemented")
}

func (s *tfplugin5Server) Configure(ctx context.Context, req *tfplugin5.Configure_Request) (*tfplugin5.Configure_Response, error) {
	resp := &tfplugin5.Configure_Response{}

	configVal, diags := decodeTFPlugin5DynamicValue(req.Config, s.p.ConfigSchema)
	if diags.HasErrors() {
		resp.Diagnostics = encodeDiagnosticsToTFPlugin5(diags)
		return resp, nil
	}

	stoppableCtx := s.stoppableContext(ctx)
	diags = s.p.configure(stoppableCtx, configVal)
	resp.Diagnostics = encodeDiagnosticsToTFPlugin5(diags)
	return resp, nil
}

func (s *tfplugin5Server) ReadResource(ctx context.Context, req *tfplugin5.ReadResource_Request) (*tfplugin5.ReadResource_Response, error) {
	resp := &tfplugin5.ReadResource_Response{}

	var rt ManagedResourceType
	if rt = s.requireManagedResourceType(req.TypeName, &resp.Diagnostics); rt == nil {
		return resp, nil
	}
	schema, _ := rt.getSchema()

	currentVal, diags := decodeTFPlugin5DynamicValue(req.CurrentState, schema)
	if diags.HasErrors() {
		resp.Diagnostics = encodeDiagnosticsToTFPlugin5(diags)
		return resp, nil
	}

	stoppableCtx := s.stoppableContext(ctx)
	newVal, diags := s.p.readResource(stoppableCtx, rt, currentVal)

	// Safety check
	wantTy := schema.ImpliedCtyType()
	for _, err := range newVal.Type().TestConformance(wantTy) {
		diags = diags.Append(Diagnostic{
			Severity: Error,
			Summary:  "Invalid result from provider",
			Detail:   fmt.Sprintf("Provider produced an invalid new object for %s: %s", req.TypeName, FormatError(err)),
		})
	}

	resp.NewState = encodeTFPlugin5DynamicValue(newVal, schema)
	resp.Diagnostics = encodeDiagnosticsToTFPlugin5(diags)
	return resp, nil
}

func (s *tfplugin5Server) PlanResourceChange(ctx context.Context, req *tfplugin5.PlanResourceChange_Request) (*tfplugin5.PlanResourceChange_Response, error) {
	resp := &tfplugin5.PlanResourceChange_Response{}

	var rt ManagedResourceType
	if rt = s.requireManagedResourceType(req.TypeName, &resp.Diagnostics); rt == nil {
		return resp, nil
	}
	schema, _ := rt.getSchema()

	priorVal, diags := decodeTFPlugin5DynamicValue(req.PriorState, schema)
	if diags.HasErrors() {
		resp.Diagnostics = encodeDiagnosticsToTFPlugin5(diags)
		return resp, nil
	}
	configVal, diags := decodeTFPlugin5DynamicValue(req.Config, schema)
	if diags.HasErrors() {
		resp.Diagnostics = encodeDiagnosticsToTFPlugin5(diags)
		return resp, nil
	}
	proposedVal, diags := decodeTFPlugin5DynamicValue(req.ProposedNewState, schema)
	if diags.HasErrors() {
		resp.Diagnostics = encodeDiagnosticsToTFPlugin5(diags)
		return resp, nil
	}

	stoppableCtx := s.stoppableContext(ctx)
	plannedVal, requiresReplace, diags := s.p.planResourceChange(stoppableCtx, rt, priorVal, configVal, proposedVal)

	// Safety check
	wantTy := schema.ImpliedCtyType()
	for _, err := range plannedVal.Type().TestConformance(wantTy) {
		diags = diags.Append(Diagnostic{
			Severity: Error,
			Summary:  "Invalid result from provider",
			Detail:   fmt.Sprintf("Provider produced an invalid planned new object for %s: %s", req.TypeName, FormatError(err)),
		})
	}

	resp.PlannedState = encodeTFPlugin5DynamicValue(plannedVal, schema)
	resp.RequiresReplace = encodeAttrPathSetToTFPlugin5(requiresReplace)
	resp.Diagnostics = encodeDiagnosticsToTFPlugin5(diags)
	return resp, nil
}

func (s *tfplugin5Server) ApplyResourceChange(ctx context.Context, req *tfplugin5.ApplyResourceChange_Request) (*tfplugin5.ApplyResourceChange_Response, error) {
	resp := &tfplugin5.ApplyResourceChange_Response{}

	var rt ManagedResourceType
	if rt = s.requireManagedResourceType(req.TypeName, &resp.Diagnostics); rt == nil {
		return resp, nil
	}
	schema, _ := rt.getSchema()

	priorVal, diags := decodeTFPlugin5DynamicValue(req.PriorState, schema)
	if diags.HasErrors() {
		resp.Diagnostics = encodeDiagnosticsToTFPlugin5(diags)
		return resp, nil
	}
	plannedVal, diags := decodeTFPlugin5DynamicValue(req.PlannedState, schema)
	if diags.HasErrors() {
		resp.Diagnostics = encodeDiagnosticsToTFPlugin5(diags)
		return resp, nil
	}

	stoppableCtx := s.stoppableContext(ctx)
	newVal, diags := s.p.applyResourceChange(stoppableCtx, rt, priorVal, plannedVal)

	// Safety check
	wantTy := schema.ImpliedCtyType()
	for _, err := range newVal.Type().TestConformance(wantTy) {
		diags = diags.Append(Diagnostic{
			Severity: Error,
			Summary:  "Invalid result from provider",
			Detail:   fmt.Sprintf("Provider produced an invalid new object for %s: %s", req.TypeName, FormatError(err)),
		})
	}

	resp.NewState = encodeTFPlugin5DynamicValue(newVal, schema)
	resp.Diagnostics = encodeDiagnosticsToTFPlugin5(diags)
	return resp, nil
}

func (s *tfplugin5Server) ImportResourceState(context.Context, *tfplugin5.ImportResourceState_Request) (*tfplugin5.ImportResourceState_Response, error) {
	return nil, grpc.Errorf(grpcCodes.Unimplemented, "not implemented")
}

func (s *tfplugin5Server) ReadDataSource(ctx context.Context, req *tfplugin5.ReadDataSource_Request) (*tfplugin5.ReadDataSource_Response, error) {
	resp := &tfplugin5.ReadDataSource_Response{}

	var rt DataResourceType
	if rt = s.requireDataResourceType(req.TypeName, &resp.Diagnostics); rt == nil {
		return resp, nil
	}
	schema := rt.getSchema()

	currentVal, diags := decodeTFPlugin5DynamicValue(req.Config, schema)
	if diags.HasErrors() {
		resp.Diagnostics = encodeDiagnosticsToTFPlugin5(diags)
		return resp, nil
	}

	stoppableCtx := s.stoppableContext(ctx)
	newVal, diags := s.p.readDataSource(stoppableCtx, rt, currentVal)

	// Safety check
	wantTy := schema.ImpliedCtyType()
	for _, err := range newVal.Type().TestConformance(wantTy) {
		diags = diags.Append(Diagnostic{
			Severity: Error,
			Summary:  "Invalid result from provider",
			Detail:   fmt.Sprintf("Provider produced an invalid new object for %s: %s", req.TypeName, FormatError(err)),
		})
	}

	resp.State = encodeTFPlugin5DynamicValue(newVal, schema)
	resp.Diagnostics = encodeDiagnosticsToTFPlugin5(diags)
	return resp, nil
}

func (s *tfplugin5Server) Stop(context.Context, *tfplugin5.Stop_Request) (*tfplugin5.Stop_Response, error) {
	// This cancels our server's root context, in the hope that the provider
	// operations will respond to this by safely cancelling their in-flight
	// actions and returning (possibly with an error) as quickly as possible.
	s.stop()
	return &tfplugin5.Stop_Response{}, nil
}

// stoppableContext returns a new context that will get cancelled if either the
// given context is cancelled or if the provider is asked to stop.
//
// This function starts a goroutine that exits only when the given context is
// cancelled, so it's important that the given context be cancelled shortly
// after the request it represents is completed.
func (s *tfplugin5Server) stoppableContext(ctx context.Context) context.Context {
	stoppable, cancel := context.WithCancel(s.ctx)
	go func() {
		<-ctx.Done()
		cancel()
	}()
	return stoppable
}

// protocolVersion5 is an implementation of rpcplugin.Server that implements
// protocol version 5.
type protocolVersion5 struct {
	p *Provider
}

var _ rpcplugin.Server = protocolVersion5{}

func (p protocolVersion5) RegisterServer(server *grpc.Server) error {
	tfplugin5.RegisterProviderServer(server, p.p.tfplugin5Server())
	return nil
}
