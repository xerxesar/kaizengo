package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"kaizengo/internal/auth"
	"kaizengo/internal/engine"
	"kaizengo/internal/module"
	"kaizengo/packages/sdk-go/acl"
	pluginpb "kaizengo/packages/sdk-go/plugin/proto"
)

// hostServer implements the Host service plugins dial through the go-plugin broker.
type hostServer struct {
	pluginpb.UnimplementedHostServer
	host   *module.Host
	doms   *engine.Domains
	source string
}

type roleLister interface {
	Roles(ctx context.Context, userID, orgID string) ([]string, error)
}

func (s *hostServer) ApplyDomain(ctx context.Context, req *pluginpb.ApplyDomainRequest) (*pluginpb.ApplyDomainResponse, error) {
	res, err := s.doms.Apply(ctx, req.GetSpec(), s.source, req.GetDryRun())
	if err != nil {
		return nil, err
	}
	return &pluginpb.ApplyDomainResponse{App: res.Domain.App, Revision: res.Domain.Revision, Changes: res.Changes}, nil
}

func (s *hostServer) GetDomain(ctx context.Context, req *pluginpb.GetDomainRequest) (*pluginpb.GetDomainResponse, error) {
	d, err := s.doms.Get(ctx, req.GetApp())
	if err != nil {
		return nil, err
	}
	if d == nil {
		return &pluginpb.GetDomainResponse{}, nil
	}
	return &pluginpb.GetDomainResponse{
		Found: true, App: d.App, Spec: d.Spec, Revision: d.Revision,
		Source: d.Source, Loaded: d.Loaded, Error: d.Error,
	}, nil
}

func (s *hostServer) Authorize(ctx context.Context, req *pluginpb.AuthorizeRequest) (*pluginpb.AuthorizeResponse, error) {
	authz, err := s.authorizer()
	if err != nil {
		return nil, err
	}
	record, err := decode(req.GetRecordJson())
	if err != nil {
		return nil, err
	}
	caller := req.GetCaller()
	d, err := authz.Can(s.asCaller(ctx, caller, false), acl.Check{
		OrgID: caller.GetOrgId(), Resource: req.GetResource(), Action: req.GetAction(),
		Record: record, Fields: req.GetFields(),
	})
	if err != nil {
		return nil, err
	}
	return &pluginpb.AuthorizeResponse{
		Allowed:     d.Allowed,
		ReadAll:     d.FieldsRead == nil,
		FieldsRead:  d.FieldsRead,
		WriteAll:    d.FieldsWrite == nil,
		FieldsWrite: d.FieldsWrite,
	}, nil
}

func (s *hostServer) Principal(ctx context.Context, req *pluginpb.PrincipalRequest) (*pluginpb.PrincipalResponse, error) {
	caller := req.GetCaller()
	out := &pluginpb.PrincipalResponse{UserId: caller.GetUserId(), OrgId: caller.GetOrgId()}
	raw, ok := s.host.Lookup(acl.ServiceName)
	if !ok {
		return out, nil
	}
	lister, ok := raw.(roleLister)
	if !ok || caller.GetUserId() == "" {
		return out, nil
	}
	roles, err := lister.Roles(ctx, caller.GetUserId(), caller.GetOrgId())
	if err != nil {
		return nil, err
	}
	out.Roles = roles
	return out, nil
}

func (s *hostServer) GetRecord(ctx context.Context, req *pluginpb.GetRecordRequest) (*pluginpb.RecordResponse, error) {
	models, err := engine.ModelsFromHost(s.host, req.GetApp())
	if err != nil {
		return nil, err
	}
	rec, err := models.Get(s.asCaller(ctx, req.GetCaller(), req.GetSystem()), req.GetCaller().GetOrgId(), req.GetModel(), req.GetId())
	if err != nil {
		return nil, err
	}
	return recordResponse(rec)
}

func (s *hostServer) ListRecords(ctx context.Context, req *pluginpb.ListRecordsRequest) (*pluginpb.ListRecordsResponse, error) {
	models, err := engine.ModelsFromHost(s.host, req.GetApp())
	if err != nil {
		return nil, err
	}
	list, err := models.List(s.asCaller(ctx, req.GetCaller(), req.GetSystem()), req.GetCaller().GetOrgId(), req.GetModel())
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []engine.Record{}
	}
	raw, err := json.Marshal(list)
	if err != nil {
		return nil, err
	}
	return &pluginpb.ListRecordsResponse{RecordsJson: raw}, nil
}

func (s *hostServer) CreateRecord(ctx context.Context, req *pluginpb.CreateRecordRequest) (*pluginpb.RecordResponse, error) {
	models, err := engine.ModelsFromHost(s.host, req.GetApp())
	if err != nil {
		return nil, err
	}
	fields, err := decode(req.GetFieldsJson())
	if err != nil {
		return nil, err
	}
	c := req.GetCaller()
	rec, err := models.Create(s.asCaller(ctx, c, req.GetSystem()), c.GetOrgId(), c.GetUserId(), req.GetModel(), fields)
	if err != nil {
		return nil, err
	}
	return recordResponse(rec)
}

func (s *hostServer) UpdateRecord(ctx context.Context, req *pluginpb.UpdateRecordRequest) (*pluginpb.RecordResponse, error) {
	models, err := engine.ModelsFromHost(s.host, req.GetApp())
	if err != nil {
		return nil, err
	}
	fields, err := decode(req.GetFieldsJson())
	if err != nil {
		return nil, err
	}
	c := req.GetCaller()
	rec, err := models.Update(s.asCaller(ctx, c, req.GetSystem()), c.GetOrgId(), req.GetModel(), req.GetId(), fields)
	if err != nil {
		return nil, err
	}
	return recordResponse(rec)
}

func (s *hostServer) DeleteRecord(ctx context.Context, req *pluginpb.DeleteRecordRequest) (*pluginpb.DeleteRecordResponse, error) {
	models, err := engine.ModelsFromHost(s.host, req.GetApp())
	if err != nil {
		return nil, err
	}
	c := req.GetCaller()
	if err := models.Delete(s.asCaller(ctx, c, req.GetSystem()), c.GetOrgId(), req.GetModel(), req.GetId()); err != nil {
		return nil, err
	}
	return &pluginpb.DeleteRecordResponse{}, nil
}

// asCaller makes ctx look like a signed-in request from caller, so the engine
// and permissions evaluate ACL as that user.
func (s *hostServer) asCaller(ctx context.Context, c *pluginpb.Caller, system bool) context.Context {
	if c.GetUserId() != "" {
		ctx = auth.WithPrincipal(ctx, &auth.Principal{UserID: c.GetUserId(), OrgID: c.GetOrgId()})
	}
	if system {
		ctx = engine.WithInternal(ctx)
	}
	return ctx
}

func (s *hostServer) authorizer() (acl.Authorizer, error) {
	raw, ok := s.host.Lookup(acl.ServiceName)
	if !ok {
		return nil, fmt.Errorf("permissions service not loaded")
	}
	authz, ok := raw.(acl.Authorizer)
	if !ok {
		return nil, fmt.Errorf("permissions service has unexpected type %T", raw)
	}
	return authz, nil
}

func decode(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func recordResponse(rec engine.Record) (*pluginpb.RecordResponse, error) {
	raw, err := json.Marshal(rec)
	if err != nil {
		return nil, err
	}
	return &pluginpb.RecordResponse{RecordJson: raw}, nil
}
