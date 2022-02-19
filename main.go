package main

import (
	"fmt"
	"path"

	"github.com/getkin/kin-openapi/openapi3"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

func printInfoEndpoint(baseServer, name string, endpoint *openapi3.PathItem) string {
	target := path.Join(baseServer, name)
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

func addEndpoints(baseServer string, endpoints map[string]openapi3.Paths) *widgets.List {
	l := widgets.NewList()
	l.Title = "Endpoints"
	rows := make([]string, 0)
	for _, endpoints := range endpoints {
		for name, endpoint := range endpoints {
			rows = append(rows, printInfoEndpoint(baseServer, name, endpoint))
		}
	}
	l.Rows = rows
	l.TextStyle = ui.NewStyle(ui.ColorYellow)
	l.WrapText = false
	l.SetRect(0, 0, 40, 40)

	return l
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

	l := addEndpoints(baseServer, sortedEndpoints)
	ui.Render(l)

	previousKey := ""
	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		switch e.ID {
		case "q", "<C-c>":
			return
		case "j", "<Down>":
			l.ScrollDown()
		case "k", "<Up>":
			l.ScrollUp()
		case "<C-d>":
			l.ScrollHalfPageDown()
		case "<C-u>":
			l.ScrollHalfPageUp()
		case "<C-f>":
			l.ScrollPageDown()
		case "<C-b>":
			l.ScrollPageUp()
		case "g":
			if previousKey == "g" {
				l.ScrollTop()
			}
		case "<Home>":
			l.ScrollTop()
		case "G", "<End>":
			l.ScrollBottom()
		}

		if previousKey == "g" {
			previousKey = ""
		} else {
			previousKey = e.ID
		}

		ui.Render(l)
	}

}
