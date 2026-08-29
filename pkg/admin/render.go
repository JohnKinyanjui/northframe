package admin

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"

	"github.com/JohnKinyanjui/northframe/pkg/auth"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

type renderer struct{ template *template.Template }

type dashboardView struct {
	Title      string
	BasePath   string
	Resources  []Resource
	CSRFToken  string
	Resource   Resource
	Mode       string
	Page       Page
	PageNumber int
	Search     string
	Fields     []fieldView
	RecordID   string
	CanCreate  bool
	CanUpdate  bool
	CanDelete  bool
	HasNext    bool
}

type fieldView struct {
	Field
	Value   string
	Checked bool
	Error   string
}

func newRenderer() (*renderer, error) {
	parsed, err := template.New("admin").Funcs(template.FuncMap{
		"recordValue": func(record Record, name string) string { return fmt.Sprint(record[name]) },
		"recordID":    recordID,
		"add":         func(value, amount int) int { return value + amount },
		"sub":         func(value, amount int) int { return value - amount },
		"queryEscape": url.QueryEscape,
	}).Parse(adminTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse admin template: %w", err)
	}
	return &renderer{template: parsed}, nil
}

func (renderer *renderer) render(writer io.Writer, view dashboardView) error {
	return renderer.template.Execute(writer, view)
}

func resourceListView(server *adminServer, request *http.Request, session auth.Session, resource Resource, page Page, pageNumber int) dashboardView {
	return dashboardView{
		Title: server.options.Title, BasePath: server.options.BasePath,
		Resources: server.registry.VisibleResources(session), Resource: resource,
		Mode: "list", Page: page, PageNumber: pageNumber, Search: request.URL.Query().Get("q"),
		CSRFToken: web.CSRFToken(web.ContextFor(request)), CanCreate: session.Can(resource.Permissions.Create),
		HasNext: int64(pageNumber*server.options.PageSize) < page.Total,
	}
}

func resourceFormView(server *adminServer, request *http.Request, session auth.Session, resource Resource, record Record, errors web.FieldErrors, create bool) dashboardView {
	fields := make([]fieldView, 0, len(resource.Fields))
	for _, field := range resource.Fields {
		if field.Hidden {
			continue
		}
		value := fmt.Sprint(record[field.Name])
		fields = append(fields, fieldView{
			Field: field, Value: value, Checked: value == "true" || value == "1" || value == "on", Error: errors[field.Name],
		})
	}
	mode := "edit"
	if create {
		mode = "new"
	}
	return dashboardView{
		Title: server.options.Title, BasePath: server.options.BasePath,
		Resources: server.registry.VisibleResources(session), Resource: resource,
		Mode: mode, Fields: fields, RecordID: request.PathValue("id"),
		CSRFToken: web.CSRFToken(web.ContextFor(request)),
		CanUpdate: session.Can(resource.Permissions.Update), CanDelete: session.Can(resource.Permissions.Delete),
	}
}

const adminTemplate = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>{{if .Resource.Name}}{{.Resource.PluralLabel}} · {{end}}{{.Title}}</title>
<style>
:root{color-scheme:light;--ink:#171712;--muted:#6b6b61;--line:#deded4;--panel:#fff;--wash:#f4f4ed;--accent:#b7f23a;--danger:#b42318}*{box-sizing:border-box}body{margin:0;background:var(--wash);color:var(--ink);font:14px/1.5 ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif}a{color:inherit}.shell{display:grid;grid-template-columns:240px minmax(0,1fr);min-height:100vh}.side{background:#171712;color:#fff;padding:28px 20px}.brand{font:700 19px/1.2 ui-serif,Georgia,serif;margin:0 0 30px}.nav{display:grid;gap:5px}.nav a{padding:9px 11px;border-radius:8px;text-decoration:none;color:#d9d9cf}.nav a:hover{background:#2b2b25;color:#fff}.main{padding:36px;min-width:0}.head{display:flex;align-items:end;justify-content:space-between;gap:20px;margin-bottom:24px}.eyebrow{color:var(--muted);font-size:11px;font-weight:750;letter-spacing:.14em;text-transform:uppercase}.head h1{font:700 34px/1.1 ui-serif,Georgia,serif;margin:5px 0 0}.button{display:inline-flex;align-items:center;justify-content:center;border:1px solid var(--ink);border-radius:999px;padding:9px 15px;background:var(--ink);color:#fff;font-weight:700;text-decoration:none;cursor:pointer}.button.accent{background:var(--accent);border-color:var(--accent);color:var(--ink)}.button.ghost{background:#fff;color:var(--ink);border-color:var(--line)}.button.danger{background:#fff;color:var(--danger);border-color:#efb4ae}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(230px,1fr));gap:14px}.card,.panel{background:var(--panel);border:1px solid var(--line);border-radius:12px;box-shadow:0 12px 30px rgba(30,30,20,.04)}.card{padding:20px;text-decoration:none}.card strong{display:block;font:700 20px ui-serif,Georgia,serif}.card span{display:block;color:var(--muted);margin-top:4px}.panel{overflow:hidden}.toolbar{display:flex;gap:10px;padding:14px;border-bottom:1px solid var(--line)}.toolbar input{flex:1}.input,select,textarea{width:100%;border:1px solid var(--line);border-radius:8px;background:#fff;padding:10px 11px;color:var(--ink);font:inherit}textarea{min-height:130px;resize:vertical}table{width:100%;border-collapse:collapse}th,td{padding:12px 14px;border-bottom:1px solid var(--line);text-align:left;vertical-align:top}th{background:#fafaf6;color:var(--muted);font-size:11px;letter-spacing:.08em;text-transform:uppercase}tbody tr:hover{background:#fafaf6}.empty{padding:60px 24px;text-align:center;color:var(--muted)}.form{padding:22px;display:grid;gap:18px;max-width:800px}.field{display:grid;gap:6px}.field label{font-weight:700}.hint,.error{font-size:12px}.hint{color:var(--muted)}.error{color:var(--danger)}.actions{display:flex;align-items:center;gap:9px;padding-top:6px}.danger-zone{border-top:1px solid var(--line);margin-top:8px;padding-top:18px}.pagination{display:flex;justify-content:space-between;padding:14px;color:var(--muted)}@media(max-width:760px){.shell{display:block}.side{padding:18px}.nav{display:flex;overflow:auto}.main{padding:22px 14px}.head{align-items:start}.panel{overflow:auto}}
</style></head><body><div class="shell"><aside class="side"><div class="brand">{{.Title}}</div><nav class="nav"><a href="{{.BasePath}}/">Overview</a>{{range .Resources}}<a href="{{$.BasePath}}/{{.Name}}">{{.PluralLabel}}</a>{{end}}</nav></aside><main class="main">
{{if eq .Mode ""}}<header class="head"><div><div class="eyebrow">Internal administration</div><h1>Resources</h1></div></header><section class="grid">{{range .Resources}}<a class="card" href="{{$.BasePath}}/{{.Name}}"><strong>{{.PluralLabel}}</strong><span>{{.Description}}</span></a>{{else}}<div class="card"><strong>No resources available</strong><span>Your account has no registered admin permissions.</span></div>{{end}}</section>{{end}}
{{if eq .Mode "list"}}<header class="head"><div><div class="eyebrow">Resource</div><h1>{{.Resource.PluralLabel}}</h1></div>{{if .CanCreate}}<a class="button accent" href="{{.BasePath}}/{{.Resource.Name}}/new">＋ Add {{.Resource.Label}}</a>{{end}}</header><section class="panel"><form class="toolbar" method="get"><input class="input" name="q" value="{{.Search}}" placeholder="Search {{.Resource.PluralLabel}}"><button class="button ghost">Search</button></form>{{if .Page.Records}}<table><thead><tr>{{range .Resource.Fields}}{{if not .Hidden}}<th>{{.Label}}</th>{{end}}{{end}}</tr></thead><tbody>{{range .Page.Records}}{{$record := .}}<tr>{{range $.Resource.Fields}}{{if not .Hidden}}<td>{{if eq .Name "id"}}<a href="{{$.BasePath}}/{{$.Resource.Name}}/{{recordID $record}}">{{recordValue $record .Name}}</a>{{else}}{{recordValue $record .Name}}{{end}}</td>{{end}}{{end}}</tr>{{end}}</tbody></table>{{else}}<div class="empty">No {{.Resource.PluralLabel}} found.</div>{{end}}<div class="pagination"><span>Page {{.PageNumber}} · {{.Page.Total}} total</span><span>{{if gt .PageNumber 1}}<a href="?q={{queryEscape .Search}}&page={{sub .PageNumber 1}}">Previous</a>{{end}} {{if .HasNext}}<a href="?q={{queryEscape .Search}}&page={{add .PageNumber 1}}">Next</a>{{end}}</span></div></section>{{end}}
{{if or (eq .Mode "new") (eq .Mode "edit")}}<header class="head"><div><div class="eyebrow">{{.Resource.Label}}</div><h1>{{if eq .Mode "new"}}Add {{.Resource.Label}}{{else}}Edit {{.Resource.Label}}{{end}}</h1></div><a class="button ghost" href="{{.BasePath}}/{{.Resource.Name}}">Back</a></header><section class="panel"><form class="form" method="post" action="{{.BasePath}}/{{.Resource.Name}}{{if eq .Mode "edit"}}/{{.RecordID}}{{end}}"><input type="hidden" name="_northframe_csrf" value="{{.CSRFToken}}">{{range .Fields}}<div class="field"><label for="admin-{{.Name}}">{{.Label}}{{if .Required}} *{{end}}</label>{{if eq .Kind "boolean"}}<input id="admin-{{.Name}}" name="{{.Name}}" type="checkbox" {{if .Checked}}checked{{end}} {{if .ReadOnly}}disabled{{end}}>{{else if eq .Kind "long_text"}}<textarea id="admin-{{.Name}}" name="{{.Name}}" {{if .Required}}required{{end}} {{if .ReadOnly}}readonly{{end}}>{{.Value}}</textarea>{{else if eq .Kind "select"}}{{$field := .}}<select id="admin-{{.Name}}" name="{{.Name}}" {{if .Required}}required{{end}} {{if .ReadOnly}}disabled{{end}}>{{range .Options}}<option value="{{.Value}}" {{if eq .Value $field.Value}}selected{{end}}>{{.Label}}</option>{{end}}</select>{{else}}<input class="input" id="admin-{{.Name}}" name="{{.Name}}" value="{{.Value}}" type="{{if eq .Kind "number"}}number{{else if eq .Kind "money"}}number{{else if eq .Kind "date"}}date{{else if eq .Kind "datetime"}}datetime-local{{else}}text{{end}}" {{if .Required}}required{{end}} {{if .ReadOnly}}readonly{{end}}>{{end}}{{if .Error}}<span class="error">{{.Error}}</span>{{end}}</div>{{end}}<div class="actions">{{if or (eq .Mode "new") .CanUpdate}}<button class="button accent" type="submit">Save {{.Resource.Label}}</button>{{end}}<a class="button ghost" href="{{.BasePath}}/{{.Resource.Name}}">Cancel</a></div></form>{{if and (eq .Mode "edit") .CanDelete}}<form class="form danger-zone" method="post" action="{{.BasePath}}/{{.Resource.Name}}/{{.RecordID}}/delete"><input type="hidden" name="_northframe_csrf" value="{{.CSRFToken}}"><div><strong>Danger zone</strong><div class="hint">Deletion is handled by the registered application repository.</div></div><button class="button danger" type="submit">Delete {{.Resource.Label}}</button></form>{{end}}</section>{{end}}
</main></div></body></html>`
