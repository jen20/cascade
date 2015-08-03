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

	"github.com/hashicorp/consul/api"
	"github.com/jwaldrip/odin/cli"
)

var Node = cli.NewSubCommand("node", "Node operations", nodeRun)

func init() {
	Node.DefineParams("action")
	Node.DefineStringFlag("role", "", "filter by role")
	Node.AliasFlag('r', "role")

	Node.SetLongDescription(`
Interact with cascade nodes

Actions:
  list - list nodes
  `)
}

func nodeRun(c cli.Command) {
	switch c.Param("action").String() {
	case "list":
		nodeList(c)
	default:
		cli.ShowUsage(c)
	}
}

func nodeList(c cli.Command) {
	client, _ := api.NewClient(api.DefaultConfig())
	catalog := client.Catalog()

	nodes, _, err := catalog.Service("cascade", c.Flag("role").String(), nil)

	if err != nil {
		log.Fatalln("Err: ", err)
	}

	for _, node := range nodes {
		fmt.Println(node.Node, node.Address+":")
		for _, role := range node.ServiceTags {
			fmt.Println("  -", role)
		}
	}
}
