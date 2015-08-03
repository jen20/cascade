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

var Role = cli.NewSubCommand("role", "Role operations", roleRun)

func init() {
	Role.DefineParams("action")

	Role.SetLongDescription(`
Interact with cascade roles

Actions (current node only):
  list - list roles
  set <roles> - set roles
  `)
}

func roleRun(c cli.Command) {
	switch c.Param("action").String() {
	case "list":
		roleList(c)
	case "set":
		roleSet(c)
	default:
		cli.ShowUsage(c)
	}
}

func roleList(c cli.Command) {
	client, _ := api.NewClient(api.DefaultConfig())
	agent := client.Agent()

	services, err := agent.Services()

	if err != nil {
		log.Fatalln("err: ", err)
	}

	self, err := agent.Self()

	if err != nil {
		log.Fatalln("err: ", err)
	}

	for _, service := range services {
		if service.Service == "cascade" {
			fmt.Println(self["Config"]["NodeName"], self["Config"]["AdvertiseAddr"].(string)+":")
			for _, role := range service.Tags {
				fmt.Println("  -", role)
			}
		}
	}
}

func roleSet(c cli.Command) {
	client, _ := api.NewClient(api.DefaultConfig())
	agent := client.Agent()

	reg := &api.AgentServiceRegistration{
		Name: "cascade",
		Tags: c.Args().Strings(),
	}

	if err := agent.ServiceRegister(reg); err != nil {
		log.Fatalln("err: ", err)
	}

	roleList(c)
}
