package main

import (
	"fmt"
	"path"

	"github.com/getkin/kin-openapi/openapi3"
)

func printInfoEndpoint(baseServer, name string, endpoint *openapi3.PathItem) {
	target := path.Join(baseServer, name)
	pattern := "\t%s %s\n"
	if endpoint.Get != nil {
		fmt.Printf(pattern, "GET", target)
	}
	if endpoint.Post != nil {
		fmt.Printf(pattern, "POST", target)
	}
	if endpoint.Put != nil {
		fmt.Printf(pattern, "PUT", target)
	}
	if endpoint.Patch != nil {
		fmt.Printf(pattern, "PATCH", target)
	}
	if endpoint.Delete != nil {
		fmt.Printf(pattern, "DELETE", target)
	}
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
	for tag, endpoints := range sortedEndpoints {
		fmt.Printf("%s:\n", tag)
		for name, endpoint := range endpoints {
			printInfoEndpoint(baseServer, name, endpoint)
		}
	}
}
