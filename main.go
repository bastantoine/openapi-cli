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

type Data struct {
	Tags           []string
	EndpointsByTag map[string][]Endpoint
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
[::i]Responses   [-:-:-]
{{ processResponses .Responses }}
`

	responses := map[string]Response{}
	for status_code, resp := range e.operationObj.Responses {
		headers := map[string]string{}
		for name, header := range resp.Value.Headers {
			headers[name] = header.Value.Description
		}
		responses[status_code] = Response{
			Description: *resp.Value.Description,
			Headers:     headers,
		}
	}

	data := struct {
		Title       string
		Description string
		Tags        string
		Color       string
		Responses   map[string]Response
		Headers     map[string]map[string]string
	}{
		Title:       e.title(),
		Description: e.operationObj.Description,
		Tags:        strings.Join(e.operationObj.Tags, ", "),
		Color:       colorString(OPERATION_COLORS_MAPPING[e.operation]),
		Responses:   responses,
	}

	var out bytes.Buffer
	t := template.Must(template.New("").Funcs(template.FuncMap{
		"processResponses": func(responses map[string]Response) string {
			var out bytes.Buffer
			for status_code, resp := range responses {
				out.WriteString(fmt.Sprintf("[::i]%s[-:-:-]:\n", status_code))
				out.WriteString(tabsOffsetString(resp.detailedInfos()))
			}
			return tabsOffsetString(out.String())
		},
	}).Parse(tmpl))
	if err := t.Execute(&out, data); err != nil {
		panic(err)
	}
	return out.String()
}

type Response struct {
	Description string
	Headers     map[string]string
}

func (r *Response) detailedInfos() string {
	tmpl := `[::i]Description [-:-:-]: {{ .Description }}
{{ if gt (len .Headers) 0 }}[::i]Headers     [-:-:-]:
{{ processHeaders .Headers }}{{ end }}
`

	var out bytes.Buffer
	t := template.Must(template.New("").Funcs(template.FuncMap{
		"processHeaders": func(headers map[string]string) string {
			var out bytes.Buffer
			for name, value := range headers {
				out.WriteString(fmt.Sprintf("[::i]%s[-:-:-]: %s\n", name, value))
			}
			return tabsOffsetString(out.String())
		},
	}).Parse(tmpl))
	if err := t.Execute(&out, r); err != nil {
		panic(err)
	}
	return out.String()
}

func tabsOffsetString(str string) string {
	parts := strings.Split(str, "\n")
	for i, part := range parts {
		if part != "" {
			parts[i] = "\t" + part
		}
	}
	return strings.Join(parts, "\n")
}

func prepare_data() (Data, error) {
	doc, err := openapi3.NewLoader().LoadFromFile("openapi.json")
	if err != nil {
		return Data{}, err
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

	return Data{
		Tags:           tags,
		EndpointsByTag: sortedEndpoints,
	}, nil
}

func prepare_gui(app *tview.Application, data Data) {
	// Endpoints tree
	root := tview.NewTreeNode("")
	endpoint_tree := tview.NewTreeView().
		SetRoot(root).
		SetCurrentNode(root).
		SetTopLevel(1)

	for _, tag := range data.Tags {
		tag_node := tview.NewTreeNode(tag).
			SetExpanded(false).
			SetReference(tag).
			SetSelectable(true)
		endpoints := data.EndpointsByTag[tag]
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

	app.SetRoot(grid, true)
}

func main() {
	data, err := prepare_data()
	if err != nil {
		panic(err)
	}

	app := tview.NewApplication()
	prepare_gui(app, data)

	if err := app.Run(); err != nil {
		panic(err)
	}

}
