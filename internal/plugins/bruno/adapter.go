// Package bruno implements the Bruno (.bru) CollectionLoader plugin.
//
// The package wraps internal/model's existing Bruno loader and adapts the
// resulting *model.Collection into the format-agnostic types defined in
// internal/canonical. The TUI never sees model.* types from this loader.
package bruno

import (
	"github.com/luca-trifilio/brio/internal/canonical"
	"github.com/luca-trifilio/brio/internal/model"
)

// adaptCollection converts a *model.Collection into a *canonical.Collection.
func adaptCollection(m *model.Collection) *canonical.Collection {
	if m == nil {
		return nil
	}
	out := &canonical.Collection{
		Name:        m.DisplayName(),
		Description: "",
		Root:        m.Path,
		Format:      formatName,
		Vars:        adaptVars(m.CollectionVars),
		Auth:        adaptAuth(m.CollectionAuth),
	}
	if m.CollectionDoc != nil {
		if pre := m.CollectionDoc.FindBlock("script", "pre-request"); pre != nil {
			out.Scripts.Pre = pre.Raw
		}
		if post := m.CollectionDoc.FindBlock("script", "post-response"); post != nil {
			out.Scripts.Post = post.Raw
		}
	}
	if m.Root != nil {
		out.RootFolder = adaptFolder(m.Root)
	}
	for _, env := range m.Environments {
		out.Environments = append(out.Environments, &canonical.Environment{
			Name: env.Name,
			Path: env.Path,
			Vars: adaptVars(env.Vars),
		})
	}
	return out
}

func adaptFolder(f *model.Folder) *canonical.Folder {
	if f == nil {
		return nil
	}
	out := &canonical.Folder{
		Path: f.Path,
		Name: f.Name,
		Seq:  f.Seq,
		Auth: adaptAuth(f.FolderAuth),
		Vars: adaptVars(f.FolderVars),
	}
	if f.FolderDoc != nil {
		if pre := f.FolderDoc.FindBlock("script", "pre-request"); pre != nil {
			out.Scripts.Pre = pre.Raw
		}
		if post := f.FolderDoc.FindBlock("script", "post-response"); post != nil {
			out.Scripts.Post = post.Raw
		}
	}
	for _, sub := range f.Folders {
		out.Folders = append(out.Folders, adaptFolder(sub))
	}
	for _, r := range f.Requests {
		out.Requests = append(out.Requests, adaptRequest(r))
	}
	return out
}

func adaptRequest(r *model.Request) *canonical.Request {
	if r == nil {
		return nil
	}
	out := &canonical.Request{
		SourcePath:  r.SourcePath,
		Name:        r.Name,
		Seq:         r.Seq,
		Method:      canonical.HTTPMethod(r.Method),
		URL:         r.URL,
		Headers:     adaptHeaders(r.Headers),
		QueryParams: adaptParams(r.QueryParams),
		PathParams:  adaptParams(r.PathParams),
		Vars:        adaptVars(r.Vars),
		PreVars:     adaptVars(r.PreVars),
		Auth:        adaptAuth(r.Auth),
		Body: canonical.Body{
			Type: r.Body.Type,
			Raw:  r.Body.Raw,
		},
		Settings: canonical.Settings{
			EncodeURL: r.Settings.EncodeURL,
			TimeoutMS: r.Settings.TimeoutMS,
		},
		Scripts: adaptScripts(r),
	}
	return out
}

func adaptScripts(r *model.Request) canonical.ScriptBlock {
	out := canonical.ScriptBlock{Post: r.PostResponseScript}
	if r.Doc != nil {
		if pre := r.Doc.FindBlock("script", "pre-request"); pre != nil {
			out.Pre = pre.Raw
		}
	}
	return out
}

func adaptVars(vs []model.Var) []canonical.Var {
	if len(vs) == 0 {
		return nil
	}
	out := make([]canonical.Var, len(vs))
	for i, v := range vs {
		out[i] = canonical.Var{Name: v.Name, Value: v.Value, Disabled: v.Disabled}
	}
	return out
}

func adaptHeaders(hs []model.Header) []canonical.Header {
	if len(hs) == 0 {
		return nil
	}
	out := make([]canonical.Header, len(hs))
	for i, h := range hs {
		out[i] = canonical.Header{Name: h.Name, Value: h.Value, Disabled: h.Disabled}
	}
	return out
}

func adaptParams(ps []model.Param) []canonical.Param {
	if len(ps) == 0 {
		return nil
	}
	out := make([]canonical.Param, len(ps))
	for i, p := range ps {
		out[i] = canonical.Param{Name: p.Name, Value: p.Value, Disabled: p.Disabled}
	}
	return out
}

func adaptAuth(a *model.AuthBlock) *canonical.AuthBlock {
	if a == nil {
		return nil
	}
	out := &canonical.AuthBlock{Mode: canonical.AuthMode(a.Mode)}
	if a.AWSv4 != nil {
		out.AWSv4 = &canonical.AuthAWSv4Cfg{
			AccessKeyID:     a.AWSv4.AccessKeyID,
			SecretAccessKey: a.AWSv4.SecretAccessKey,
			SessionToken:    a.AWSv4.SessionToken,
			Service:         a.AWSv4.Service,
			Region:          a.AWSv4.Region,
			ProfileName:     a.AWSv4.ProfileName,
		}
	}
	return out
}
