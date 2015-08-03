//
// Author:: Zachary Schneider (<schneider@boundary.com>)
// Copyright:: Copyright (c) 2015 Boundary, Inc.
// License:: Apache License, Version 2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package command

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/jwaldrip/odin/cli"
)

var Service = cli.NewSubCommand("service", "Service operations", serviceRun)

func init() {
	Service.DefineParams("action")
	Service.DefineStringFlag("type", "", "filter by type")
	Service.AliasFlag('t', "type")
	Service.SetLongDescription(`
Interact with cascade services

Actions:
  list - list registered services
  local - list services on current node
  find <servicename> - list nodes with service
  `)
}

func serviceRun(c cli.Command) {
	switch c.Param("action").String() {
	case "list":
		serviceList(c)
	case "local":
		serviceLocal(c)
	case "find":
		serviceFind(c)
	default:
		cli.ShowUsage(c)
	}
}

func serviceList(c cli.Command) {
	client, _ := api.NewClient(api.DefaultConfig())
	catalog := client.Catalog()

	services, meta, err := catalog.Services(nil)

	if err != nil {
		log.Fatalln("Err:", err)
	}

	if meta.LastIndex == 0 {
		log.Fatalln("Bad: ", meta)
	}

	sorted := make([]string, 0)

	for index, _ := range services {
		sorted = append(sorted, index)
	}

	sort.Strings(sorted)

	for _, service := range sorted {
		fmt.Println("  -", service)
	}
}

func serviceLocal(c cli.Command) {
	client, _ := api.NewClient(api.DefaultConfig())
	agent := client.Agent()

	services, err := agent.Services()

	if err != nil {
		log.Fatalln("err: ", err)
	}

	if err != nil {
		log.Fatalln("err: ", err)
	}

	// sigh
	sorted := make([]string, 0)
	seen := make(map[string]bool)

	for _, service := range services {
		if !seen[service.Service] && service.Service != "cascade" {
			sorted = append(sorted, service.Service)
			seen[service.Service] = true
		}
	}

	sort.Strings(sorted)

	for _, service := range sorted {
		fmt.Println(service + ":")
		for _, st := range services {
			if st.Service == service {
				fmt.Println("  - port:", st.Port)
				fmt.Println("    tags:", strings.Join(st.Tags, ", "))
			}
		}
	}
}

func serviceFind(c cli.Command) {
	client, _ := api.NewClient(api.DefaultConfig())
	catalog := client.Catalog()

	if len(c.Args().GetAll()) == 0 {
		log.Fatalln("err: missing <servicename> argument")
	}

	nodes, _, err := catalog.Service(c.Arg(0).String(), c.Flag("type").String(), nil)

	if err != nil {
		log.Fatalln("err: ", err)
	}

	fmt.Println(c.Arg(0).String() + ":")
	for _, node := range nodes {
		fmt.Println("  - host:", node.Node)
		fmt.Println("    address:", node.Address)
		fmt.Println("    port:", node.ServicePort)
		fmt.Println("    tags:", strings.Join(node.ServiceTags, ", "))
	}
}
