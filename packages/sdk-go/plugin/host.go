package plugin

import (
	"context"
	"encoding/json"
	"errors"

	"google.golang.org/grpc/status"

	pluginpb "kaizengo/packages/sdk-go/plugin/proto"
)

// HostClient calls platform services on the KaizenGo host. Calls taking a Call
// run as its caller, so ACL applies exactly as for that user's GraphQL calls.
type HostClient struct {
	rpc    pluginpb.HostClient
	system bool
}

// AsSystem returns a client whose record calls run as trusted internal calls:
// no ACL, internal models writable.
func (h *HostClient) AsSystem() *HostClient {
	return &HostClient{rpc: h.rpc, system: true}
}

// Domain is a stored dynamic-domain definition.
type Domain struct {
	App      string
	Spec     string
	Revision int64
	Source   string
	Loaded   bool
	Error    string
}

// ApplyResult is the outcome of ApplyDomain.
type ApplyResult struct {
	App      string
	Revision int64
	Changes  []string
}

// Decision is an ACL decision. Nil field lists mean all fields.
type Decision struct {
	Allowed     bool
	FieldsRead  []string
	FieldsWrite []string
}

// Principal is the caller with their roles in the org.
type Principal struct {
	UserID string
	OrgID  string
	Roles  []string
}

// ApplyDomain validates, migrates (additively), and mounts an app.yaml document.
func (h *HostClient) ApplyDomain(ctx context.Context, spec string, dryRun bool) (*ApplyResult, error) {
	res, err := h.rpc.ApplyDomain(ctx, &pluginpb.ApplyDomainRequest{Spec: spec, DryRun: dryRun})
	if err != nil {
		return nil, unwrap(err)
	}
	return &ApplyResult{App: res.GetApp(), Revision: res.GetRevision(), Changes: res.GetChanges()}, nil
}

// GetDomain returns a stored dynamic domain, or nil when none exists.
func (h *HostClient) GetDomain(ctx context.Context, app string) (*Domain, error) {
	res, err := h.rpc.GetDomain(ctx, &pluginpb.GetDomainRequest{App: app})
	if err != nil {
		return nil, unwrap(err)
	}
	if !res.GetFound() {
		return nil, nil
	}
	return &Domain{
		App: res.GetApp(), Spec: res.GetSpec(), Revision: res.GetRevision(),
		Source: res.GetSource(), Loaded: res.GetLoaded(), Error: res.GetError(),
	}, nil
}

// Authorize evaluates ACL for the caller. record enables domain (row) rules;
// fields lists the fields touched by a write.
func (h *HostClient) Authorize(c Call, resource, action string, record map[string]any, fields ...string) (Decision, error) {
	raw, err := encodeMap(record)
	if err != nil {
		return Decision{}, err
	}
	res, err := h.rpc.Authorize(c, &pluginpb.AuthorizeRequest{
		Caller: caller(c), Resource: resource, Action: action, RecordJson: raw, Fields: fields,
	})
	if err != nil {
		return Decision{}, unwrap(err)
	}
	d := Decision{Allowed: res.GetAllowed()}
	if !res.GetReadAll() {
		d.FieldsRead = append([]string{}, res.GetFieldsRead()...)
	}
	if !res.GetWriteAll() {
		d.FieldsWrite = append([]string{}, res.GetFieldsWrite()...)
	}
	return d, nil
}

// Principal returns the caller's roles.
func (h *HostClient) Principal(c Call) (Principal, error) {
	res, err := h.rpc.Principal(c, &pluginpb.PrincipalRequest{Caller: caller(c)})
	if err != nil {
		return Principal{}, unwrap(err)
	}
	return Principal{UserID: res.GetUserId(), OrgID: res.GetOrgId(), Roles: res.GetRoles()}, nil
}

// GetRecord loads one record of app.model in the caller's org.
func (h *HostClient) GetRecord(c Call, app, model, id string) (map[string]any, error) {
	res, err := h.rpc.GetRecord(c, &pluginpb.GetRecordRequest{Caller: caller(c), App: app, Model: model, Id: id, System: h.system})
	if err != nil {
		return nil, unwrap(err)
	}
	return decodeRecord(res.GetRecordJson())
}

// ListRecords lists records of app.model visible to the caller.
func (h *HostClient) ListRecords(c Call, app, model string) ([]map[string]any, error) {
	res, err := h.rpc.ListRecords(c, &pluginpb.ListRecordsRequest{Caller: caller(c), App: app, Model: model, System: h.system})
	if err != nil {
		return nil, unwrap(err)
	}
	var out []map[string]any
	if raw := res.GetRecordsJson(); len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// CreateRecord creates a record through the engine (hooks, validation, ACL).
func (h *HostClient) CreateRecord(c Call, app, model string, fields map[string]any) (map[string]any, error) {
	raw, err := encodeMap(fields)
	if err != nil {
		return nil, err
	}
	res, err := h.rpc.CreateRecord(c, &pluginpb.CreateRecordRequest{Caller: caller(c), App: app, Model: model, FieldsJson: raw, System: h.system})
	if err != nil {
		return nil, unwrap(err)
	}
	return decodeRecord(res.GetRecordJson())
}

// UpdateRecord patches a record through the engine.
func (h *HostClient) UpdateRecord(c Call, app, model, id string, fields map[string]any) (map[string]any, error) {
	raw, err := encodeMap(fields)
	if err != nil {
		return nil, err
	}
	res, err := h.rpc.UpdateRecord(c, &pluginpb.UpdateRecordRequest{Caller: caller(c), App: app, Model: model, Id: id, FieldsJson: raw, System: h.system})
	if err != nil {
		return nil, unwrap(err)
	}
	return decodeRecord(res.GetRecordJson())
}

// DeleteRecord soft-deletes a record through the engine.
func (h *HostClient) DeleteRecord(c Call, app, model, id string) error {
	_, err := h.rpc.DeleteRecord(c, &pluginpb.DeleteRecordRequest{Caller: caller(c), App: app, Model: model, Id: id, System: h.system})
	return unwrap(err)
}

func caller(c Call) *pluginpb.Caller {
	info := c.CallerInfo()
	return &pluginpb.Caller{OrgId: info.OrgID, UserId: info.UserID}
}

func encodeMap(m map[string]any) ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

func decodeRecord(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// unwrap turns gRPC status errors back into plain messages from the host.
func unwrap(err error) error {
	if err == nil {
		return nil
	}
	if s, ok := status.FromError(err); ok {
		return errors.New(s.Message())
	}
	return err
}
