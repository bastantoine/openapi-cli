package main

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

type nodeValue string

func (nv nodeValue) String() string {
	return string(nv)
}

func printInfoEndpoint(baseServer, name string, endpoint *openapi3.PathItem) string {
	target := name
	pattern := "\t%s %s\n"
	if endpoint.Get != nil {
		return fmt.Sprintf(pattern, "GET", target)
	}
	if endpoint.Post != nil {
		return fmt.Sprintf(pattern, "POST", target)
	}
	if endpoint.Put != nil {
		return fmt.Sprintf(pattern, "PUT", target)
	}
	if endpoint.Patch != nil {
		return fmt.Sprintf(pattern, "PATCH", target)
	}
	if endpoint.Delete != nil {
		return fmt.Sprintf(pattern, "DELETE", target)
	}
	return ""
}

func addEndpoints(baseServer string, endpoints map[string]openapi3.Paths, maxWidth, maxHeight int) *widgets.Tree {
	rows := make([]string, 0)
	nodes := []*widgets.TreeNode{}
	for tag, endpoints := range endpoints {
		nodesTag := []*widgets.TreeNode{}
		for name, endpoint := range endpoints {
			nodesTag = append(nodesTag, &widgets.TreeNode{
				Value: nodeValue(printInfoEndpoint(baseServer, name, endpoint)),
			})
			rows = append(rows, printInfoEndpoint(baseServer, name, endpoint))
		}
		nodes = append(nodes, &widgets.TreeNode{
			Value: nodeValue(tag),
			Nodes: nodesTag,
		})
	}

	t := widgets.NewTree()
	t.SetNodes(nodes)
	t.TextStyle = ui.NewStyle(ui.ColorYellow)
	t.WrapText = false
	t.SetRect(0, 0, int(1.0/4.0*float64(maxWidth)), maxHeight)

	return t
}

func main() {
	doc, err := openapi3.NewLoader().LoadFromFile("openapi.json")
	if err != nil {
		panic(err)
	}

	// TODO: what to do when there's multiple servers?
	baseServer := doc.Servers[0].URL

	sortedEndpoints := make(map[string]openapi3.Paths)
	for name, endpoint := range doc.Paths {
		for _, operation := range []*openapi3.Operation{
			endpoint.Get,
			endpoint.Post,
			endpoint.Put,
			endpoint.Patch,
			endpoint.Delete,
		} {
			if operation != nil {
				if len(operation.Tags) == 0 {
					sortedEndpoints[""][name] = endpoint
				} else {
					for _, tag := range operation.Tags {
						if _, ok := sortedEndpoints[tag]; !ok {
							sortedEndpoints[tag] = make(openapi3.Paths)
						}
						sortedEndpoints[tag][name] = endpoint
					}
				}
			}
		}
	}

	if err := ui.Init(); err != nil {
		panic(fmt.Errorf("failed to initialize termui: %v", err))
	}
	defer ui.Close()
	maxWidth, maxHeight := ui.TerminalDimensions()

	t := addEndpoints(baseServer, sortedEndpoints, maxWidth, maxHeight)
	ui.Render(t)

	previousKey := ""
	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		switch e.ID {
		case "q", "<C-c>":
			return
		case "j", "<Down>":
			t.ScrollDown()
		case "k", "<Up>":
			t.ScrollUp()
		case "<C-d>":
			t.ScrollHalfPageDown()
		case "<C-u>":
			t.ScrollHalfPageUp()
		case "<C-f>":
			t.ScrollPageDown()
		case "<C-b>":
			t.ScrollPageUp()
		case "g":
			if previousKey == "g" {
				t.ScrollTop()
			}
		case "<Home>":
			t.ScrollTop()
		case "G", "<End>":
			t.ScrollBottom()
		case "<Enter>":
			t.ToggleExpand()
		case "<Right>":
			t.Expand()
		case "<Left>":
			t.Collapse()
		}

		if previousKey == "g" {
			previousKey = ""
		} else {
			previousKey = e.ID
		}

		ui.Render(t)
	}

}
