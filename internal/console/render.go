package console

import (
	"encoding/json"
	"html/template"
	"strconv"
	"strings"
)

// page is the whole surface. It is server-rendered with no client framework,
// no build step and no external asset: every byte is served from this
// process, so LAC-V0-002's "no outbound connection" holds for the page as
// well as for the server.
//
// Every interpolation goes through html/template's contextual escaping, which
// is what makes ticket titles, review text and agent output inert
// (LAC-V0-020). Nothing here linkifies text, and no value reaches an
// attribute or script context.
var page = template.Must(template.New("page").Funcs(template.FuncMap{
	"or_dash": func(s string) string {
		if strings.TrimSpace(s) == "" {
			return "—"
		}
		return s
	},
	"refusalOf": func(heading string, source Source, err string) Refusal {
		return Refusal{Heading: heading, Err: err, Source: source}
	},
	"refusalOfEnvelope": func(heading string, source Source, e *Envelope) Refusal {
		return Refusal{Heading: heading, Outcome: e.Outcome, Codes: e.Codes, Warnings: e.Warnings, Source: source}
	},
	// originField names the record fields that answer "where does this ticket
	// come from". Everything else stays on the record itself.
	"originField": func(key string) bool {
		switch key {
		case "source", "createdAt", "updatedAt", "updatedBy", "supersedes", "supersededBy",
			"requirementRefs", "acceptanceCriteria", "labels", "body", "shadowOverlay":
			return true
		}
		return false
	},
	// render turns one decoded JSON value into text. It never emits markup:
	// every value it returns is escaped by the template that prints it.
	"render": renderValue,
}).Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>{{.Title}} · Corvint Console</title>
<style>
:root{--bg:#f6f7f9;--fg:#1b1f24;--dim:#5b6470;--line:#d7dbe0;--card:#fff;--warn:#8a5a00;
--warnbg:#fff6e0;--bad:#8a1f1f;--badbg:#ffecec;--ok:#0f5132;--unstated:#6b7280;--unstatedbg:#eef0f3}
*{box-sizing:border-box}
body{margin:0;font:14px/1.5 -apple-system,BlinkMacSystemFont,"Segoe UI",Helvetica,Arial,sans-serif;
background:var(--bg);color:var(--fg)}
header{background:#fff;border-bottom:1px solid var(--line);padding:12px 20px;position:sticky;top:0;z-index:5}
header h1{margin:0;font-size:15px;font-weight:600}
header nav{margin-top:6px;font-size:12px;color:var(--dim)}
header nav a{color:#0b57d0;margin-right:14px}
main{padding:20px;max-width:100%}
.board{display:flex;gap:14px;overflow-x:auto;padding-bottom:12px;align-items:flex-start}
.col{flex:0 0 300px;background:#eceef1;border-radius:8px;padding:10px}
.col h2{margin:0 0 8px;font-size:12px;letter-spacing:.04em;text-transform:uppercase;color:var(--dim)}
.col.unmapped{background:var(--warnbg);outline:1px solid #e5c98a}
.card{background:var(--card);border:1px solid var(--line);border-radius:6px;padding:9px 10px;margin-bottom:8px}
.card a{color:inherit;text-decoration:none;font-weight:600;display:block}
.meta{margin-top:6px;font-size:11px;color:var(--dim);display:flex;gap:8px;flex-wrap:wrap}
.pill{border:1px solid var(--line);border-radius:99px;padding:1px 7px;font-size:11px;background:#fafbfc}
.refusal{background:var(--badbg);border:1px solid #e0b4b4;border-left:4px solid var(--bad);
padding:12px 14px;border-radius:6px;margin-bottom:16px}
.refusal h2{margin:0 0 6px;font-size:13px;color:var(--bad)}
.note{background:var(--warnbg);border:1px solid #e5c98a;border-radius:6px;padding:10px 12px;margin-bottom:14px}
.panel{background:#fff;border:1px solid var(--line);border-radius:8px;padding:14px 16px;margin-bottom:16px}
.panel h2{margin:0 0 10px;font-size:13px;text-transform:uppercase;letter-spacing:.04em;color:var(--dim)}
table{border-collapse:collapse;width:100%;font-size:13px}
td,th{text-align:left;padding:4px 8px 4px 0;vertical-align:top}
th{color:var(--dim);font-weight:500;white-space:nowrap;width:1%}
.axes{display:flex;gap:6px;flex-wrap:wrap;margin-top:8px}
.axis{font-size:11px;border-radius:4px;padding:2px 7px;border:1px solid var(--line);background:#fafbfc}
.axis.unstated{background:var(--unstatedbg);color:var(--unstated);border-style:dashed}
.axis b{font-weight:600}
.src{margin-top:10px;font-size:11px;color:var(--dim);border-top:1px dashed var(--line);padding-top:8px}
.src code{background:#f0f2f4;padding:1px 5px;border-radius:3px;font-size:11px;
font-family:ui-monospace,SFMono-Regular,Menlo,monospace;word-break:break-all}
pre{background:#0f1115;color:#e6e9ef;padding:12px;border-radius:6px;overflow-x:auto;font-size:12px;
font-family:ui-monospace,SFMono-Regular,Menlo,monospace;margin:0}
.unknown{font-size:11px;color:var(--warn);background:var(--warnbg);border-radius:4px;padding:2px 6px;
display:inline-block;margin-top:4px}
.num{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:12px}
.unmeasured{color:var(--unstated);background:var(--unstatedbg);border:1px dashed var(--line);
border-radius:4px;padding:1px 6px;font-size:11px}
tr.metric td{border-top:1px solid #eef0f3}
.grp{font-size:11px;color:var(--dim);text-transform:uppercase;letter-spacing:.04em;margin:14px 0 4px}
form.mut{display:flex;gap:6px;align-items:flex-start;margin:6px 0;flex-wrap:wrap}
form.mut input[type=text]{font-family:ui-monospace,Menlo,monospace;font-size:12px;padding:4px 6px;
border:1px solid var(--line);border-radius:4px;min-width:280px;flex:1}
button{font-size:12px;padding:4px 12px;border-radius:5px;border:1px solid var(--line);
background:#fff;cursor:pointer}
button[disabled]{cursor:not-allowed;opacity:.5;background:#f0f0f2}
.disabled-reason{font-size:11px;color:var(--dim);flex-basis:100%;margin:-2px 0 4px}
footer{padding:14px 20px;color:var(--dim);font-size:11px;border-top:1px solid var(--line);background:#fff}
.ticket-form{max-width:48rem;display:grid;gap:.75rem}
.ticket-form label{display:grid;gap:.3rem;font-weight:600}
.ticket-form input,.ticket-form textarea{box-sizing:border-box;width:100%;font:inherit;padding:.6rem;border:1px solid #9ca3af;border-radius:4px}
.ticket-form textarea{min-height:7rem;resize:vertical}
.ticket-form button{justify-self:start;padding:.6rem 1rem}
.ticket-form :focus-visible{outline:2px solid #2563eb;outline-offset:2px}
.panel,.card{overflow-wrap:anywhere}
</style></head><body>
<header>
<h1>{{.Title}}</h1>
<nav><a href="/">Board</a><a href="/specs">Specs</a><a href="/evidence">Evidence</a><a href="/dogfood">Dogfood</a><a href="/benchmarks">Benchmarks</a><a href="/backlogs">Backlogs</a>
<span>repo {{.Repo}}</span>
<span>· revision {{if .Revision.Commit}}{{slice .Revision.Commit 0 12}}{{else}}unknown{{end}}</span>
{{if .Revision.Dirty}}<span>· worktree dirty ({{len .Revision.DirtyPaths}} paths)</span>{{end}}
<span>· compiled at {{.CompiledAt}}</span></nav>
</header>
<main>

{{if .Boundary}}<div class="note"><b>This page is historical.</b> It was compiled at
{{.CompiledAt}} against revision {{or_dash .Revision.Commit}}. Reload to observe again;
nothing on it is guaranteed to still hold.</div>{{end}}

{{template "body" .}}

</main>
<footer>
Corvint Console · loopback only, no account, no database, no outbound connection.
It renders what <code>corvint-tasks</code>, <code>corvint-dashboard-snapshot</code> and <code>git</code>
report and performs a change only by invoking the owning tool's own verb. It is never a gate: it cannot accept a specification,
advance a delivery stage, or qualify anything.
</footer>
</body></html>`))

// mustView builds one page variant. Each view gets its own clone of the base
// set, so the "body" each defines cannot collide with another view's.
func mustView(body string) *template.Template {
	clone := template.Must(base().Clone())
	return template.Must(clone.Parse(ticketFormTemplate + body))
}

// renderValue is the text form of one decoded JSON value.
func renderValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return "—"
	case string:
		if strings.TrimSpace(typed) == "" {
			return "—"
		}
		return typed
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "—"
	}
	return string(encoded)
}

// axesTemplate renders the six axes of one value. An axis its source did not
// state is visibly distinct from one that was measured, which is the whole
// point of LAC-V0-008.
var axesTemplate = template.Must(page.New("axes").Parse(`
<div class="axes">
{{range .Pairs}}<span class="axis{{if not .Stated}} unstated{{end}}"><b>{{.Axis}}</b> {{.Value}}</span>{{end}}
</div>`))

// sourceTemplate renders one value's attribution: the exact invocation, the
// tool, and the revision it answered about (LAC-V0-006).
var sourceTemplate = template.Must(page.New("source").Parse(`
<div class="src">
<div>read by <code>{{.Line}}</code></div>
{{if .Tool}}<div>tool {{.Tool}}</div>{{end}}
{{if .Revision}}<div>revision <code>{{.Revision}}</code></div>{{end}}
<div>observed at {{.ObservedAt.Format "2006-01-02T15:04:05Z"}}</div>
{{if .Outcome}}<div>envelope outcome {{.Outcome}}{{if .Codes}} · codes {{range .Codes}}{{.}} {{end}}{{end}}</div>{{end}}
{{template "axes" .Axes}}
</div>`))

// refusalTemplate renders a refusal as itself, with the tool's exact codes
// and warning text. A refusal is never an empty board (LAC-V0-009).
var refusalTemplate = template.Must(page.New("refusal").Parse(`
<div class="refusal">
<h2>{{.Heading}}</h2>
{{if .Outcome}}<div>outcome <b>{{.Outcome}}</b>{{if .Codes}} · codes {{range .Codes}}<code>{{.}}</code> {{end}}{{end}}</div>{{end}}
{{range .Warnings}}<div>{{.}}</div>{{end}}
{{if .Err}}<div>{{.Err}}</div>{{end}}
<div style="margin-top:8px;font-size:12px">This is the tool's own refusal, shown as a refusal.
It is not an empty result, and no count on this page should be read as zero.</div>
{{template "source" .Source}}
</div>`))

// Refusal is what refusalTemplate renders.
type Refusal struct {
	Heading  string
	Outcome  string
	Codes    []string
	Warnings []string
	Err      string
	Source   Source
}

// base is the page shell plus the three shared partials, ready to be cloned.
func base() *template.Template {
	return sharedTemplates
}

var sharedTemplates = func() *template.Template {
	_ = axesTemplate
	_ = sourceTemplate
	_ = refusalTemplate
	return page
}()
