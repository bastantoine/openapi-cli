package main

import (
	"fmt"
	"path"

	"github.com/getkin/kin-openapi/openapi3"
)

func main() {
	doc, err := openapi3.NewLoader().LoadFromFile("openapi.json")
	if err != nil {
		panic(err)
	}

	// TODO: what to do when there's multiple servers?
	baseServer := doc.Servers[0].URL

	for name, endpoint := range doc.Paths {
		target := path.Join(baseServer, name)
		pattern := "%s@%s\n"
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
}
