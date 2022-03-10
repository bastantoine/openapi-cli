package main

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/rivo/tview"
)

type Endpoint struct {
	name         string
	operation    string
	operationObj openapi3.Operation
}

func main() {
	doc, err := openapi3.NewLoader().LoadFromFile("openapi.json")
	if err != nil {
		panic(err)
	}

	sortedEndpoints := make(map[string][]Endpoint)
	sortedEndpoints[""] = []Endpoint{}
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
					sortedEndpoints[""] = append(sortedEndpoints[""], endpoint)
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
	fmt.Println(sortedEndpoints)

	box := tview.NewBox().SetBorder(true).SetTitle("Hello, world!")
	if err := tview.NewApplication().SetRoot(box, true).Run(); err != nil {
		panic(err)
	}

}
