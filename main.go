package main

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/gdamore/tcell/v2"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/rivo/tview"
)

const UNTAGGED_ENDPOINT = "[::i]< untagged >[-:-:-]"

func colorString(c tcell.Color) string {
	for name, color := range tcell.ColorNames {
		if color == c {
			return name
		}
	}
	r, g, b := c.RGB()
	return fmt.Sprintf("#%d%d%d", r, g, b)
}

var OPERATION_COLORS_MAPPING = map[string]tcell.Color{
	"GET":    tcell.ColorBlue,
	"POST":   tcell.ColorGreen,
	"DELETE": tcell.ColorRed,
	"PUT":    tcell.ColorYellow,
	"PATCH":  tcell.ColorYellow,
}

type Endpoint struct {
	name         string
	operation    string
	operationObj openapi3.Operation
}

func (e *Endpoint) title() string {
	return fmt.Sprintf("%s@%s", e.operation, e.name)
}

func (e *Endpoint) detailedInfos() string {
	tmpl := `[{{ .Color }}::b]{{ .Title }}[-:-:-]

[::i]Description [-:-:-]: {{ .Description }}
[::i]Tags        [-:-:-]: {{ .Tags }}
[::i]Responses   [-:-:-]:
{{ tmplResponses .Responses }}
`

	responses := map[string]struct{ Description string }{}
	for status_code, resp := range e.operationObj.Responses {
		responses[status_code] = struct{ Description string }{Description: *resp.Value.Description}
	}

	data := struct {
		Title       string
		Description string
		Tags        string
		Color       string
		Responses   map[string]struct{ Description string }
	}{
		Title:       e.title(),
		Description: e.operationObj.Description,
		Tags:        strings.Join(e.operationObj.Tags, ", "),
		Color:       colorString(OPERATION_COLORS_MAPPING[e.operation]),
		Responses:   responses,
	}

	var out bytes.Buffer
	t := template.Must(template.New("").Funcs(template.FuncMap{
		"tmplResponses": func(responses map[string]struct{ Description string }) string {
			var out bytes.Buffer
			status_codes := make([]string, 0, len(responses))
			for status_code := range responses {
				status_codes = append(status_codes, status_code)
			}
			sort.Strings(status_codes)
			for _, status_code := range status_codes {
				resp := responses[status_code]
				out.WriteString(fmt.Sprintf("\t[::i]%s[-:-:-]: %s\n", status_code, resp.Description))
			}
			return out.String()
		},
	}).Parse(tmpl))
	if err := t.Execute(&out, data); err != nil {
		panic(err)
	}
	return out.String()
}

func main() {
	doc, err := openapi3.NewLoader().LoadFromFile("openapi.json")
	if err != nil {
		panic(err)
	}

	// Sort the endpoints by their tags
	sortedEndpoints := make(map[string][]Endpoint)
	sortedEndpoints[UNTAGGED_ENDPOINT] = []Endpoint{}
	for name, endpoint := range doc.Paths {
		for _, operation := range []struct {
			name string
			obj  *openapi3.Operation
		}{
			{
				name: "GET",
				obj:  endpoint.Get,
			},
			{
				name: "POST",
				obj:  endpoint.Post,
			},
			{
				name: "PUT",
				obj:  endpoint.Put,
			},
			{
				name: "PATCH",
				obj:  endpoint.Patch,
			},
			{
				name: "DELETE",
				obj:  endpoint.Delete,
			},
		} {
			if operation.obj != nil {
				endpoint := Endpoint{
					name:         name,
					operation:    operation.name,
					operationObj: *operation.obj,
				}
				if len(operation.obj.Tags) == 0 {
					sortedEndpoints[UNTAGGED_ENDPOINT] = append(sortedEndpoints[UNTAGGED_ENDPOINT], endpoint)
				} else {
					for _, tag := range operation.obj.Tags {
						if _, ok := sortedEndpoints[tag]; !ok {
							sortedEndpoints[tag] = []Endpoint{}
						}
						sortedEndpoints[tag] = append(sortedEndpoints[tag], endpoint)
					}
				}
			}
		}
	}

	tags := []string{}
	for tag := range sortedEndpoints {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	app := tview.NewApplication()

	// Endpoints tree
	root := tview.NewTreeNode("")
	endpoint_tree := tview.NewTreeView().
		SetRoot(root).
		SetCurrentNode(root).
		SetTopLevel(1)

	for _, tag := range tags {
		tag_node := tview.NewTreeNode(tag).
			SetExpanded(false).
			SetReference(tag).
			SetSelectable(true)
		endpoints := sortedEndpoints[tag]
		if len(endpoints) > 0 {
			for _, endpoint := range endpoints {
				tag_node.AddChild(tview.NewTreeNode(endpoint.title()).
					SetReference(endpoint).
					SetSelectable(true).
					SetColor(OPERATION_COLORS_MAPPING[endpoint.operation]),
				)
			}
		}
		root.AddChild(tag_node)
	}

	// Bottom panel
	bottom_panel := tview.NewTextView().
		SetText("[e]: list of endpoints | [q]: quit")

	endpoint_infos_box := tview.NewTextView().
		SetDynamicColors(true)

	endpoint_tree.SetSelectedFunc(func(node *tview.TreeNode) {
		ref := node.GetReference()
		if ref == nil {
			return
		}
		if _, ok := ref.(string); ok {
			// Ref is a string, this means it's a tag
			node.SetExpanded(!node.IsExpanded())
			return
		}
		endpoint := ref.(Endpoint)
		endpoint_infos_box.SetText(endpoint.detailedInfos())
	})

	grid := tview.NewGrid().
		SetRows(0, 1).
		SetColumns(80, 0).
		SetBorders(true).
		AddItem(endpoint_tree, 0, 0, 1, 1, 0, 0, false).
		AddItem(bottom_panel, 1, 0, 1, 3, 0, 0, false).
		AddItem(endpoint_infos_box, 0, 1, 1, 2, 0, 0, false)

	grid.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'e', 'E':
			app.SetFocus(endpoint_tree)
			return nil
		case 'q', 'Q':
			app.Stop()
			return nil
		default:
			return event
		}

	})

	if err := app.SetRoot(grid, true).Run(); err != nil {
		panic(err)
	}

}
