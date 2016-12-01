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

Actions:
  list - list local roles
  listAll - list all nodes and roles
  find <role> - list nodes with role
  set <roles> - set local roles (replaces current)
  append <roles> - append roles to local set
  rm <roles> - remove roles from local set
  `)
}

func roleRun(c cli.Command) {
	switch c.Param("action").String() {
	case "list":
		roleList(c)
	case "set":
		roleSet(c)
	case "listAll":
		roleListAll(c)
	case "find":
		roleFind(c)
	case "append":
		roleAppend(c)
	case "rm":
		roleRm(c)
	default:
		cli.ShowUsage(c)
	}
}

func roleListAll(_ cli.Command) {

	roles, err := allNodeRoles()
	if err != nil {
		log.Fatalln("err: ", err)
	}

	for k, v := range roles {
		printRole(k, v)
	}
}

func roleList(_ cli.Command) {

	nodeRoles, err := allNodeRoles()
	if err != nil {
		log.Fatalln("err: ", err)
	}

	myKey, err := selfKey()
	if err != nil {
		log.Fatalln("err: ", err)
	}
	printRole(myKey, nodeRoles[myKey])
}

func roleFind(c cli.Command) {
	cmdRoles := c.Args().Strings()
	if len(cmdRoles) == 0 {
		log.Fatalln("Must specify a role to find")
	}
	if len(cmdRoles) != 1 {
		// maybe we could support multiple but i don't think it's necessary
		log.Fatalln("One role only is supported for `find` command")
	}
	role := cmdRoles[0]
	allRoles, err := allNodeRoles()
	if err != nil {
		log.Fatalln("err: ", err)
	}

	fmt.Printf("All nodes containing role %s:\n\n", role)
	for node, r := range allRoles {
		if StrContains(r, role) {
			printRole(node, r)
		}
	}
}

func roleSet(c cli.Command) {
	roleActualSet(c.Args().Strings(), c)
}

func roleActualSet(roles []string, c cli.Command) {
	client, _ := api.NewClient(api.DefaultConfig())
	agent := client.Agent()

	reg := &api.AgentServiceRegistration{
		Name: "cascade",
		Tags: roles,
	}

	if err := agent.ServiceRegister(reg); err != nil {
		log.Fatalln("err: ", err)
	}

	roleList(c)
}

func roleAppend(c cli.Command) {
	nodeRoles, err := allNodeRoles()
	if err != nil {
		log.Fatalln("err: ", err)
	}

	myKey, err := selfKey()
	if err != nil {
		log.Fatalln("err: ", err)
	}

	var finalSet []string
	for _, role := range nodeRoles[myKey] {
		finalSet = append(finalSet, role)
	}

	for _, role := range c.Args().Strings() {
		if !StrContains(finalSet, role) {
			finalSet = append(finalSet, role)
		}
	}

	roleActualSet(finalSet, c)
}

func roleRm(c cli.Command) {
	rmRoles := c.Args().Strings()
	if (len(rmRoles) == 0) {
		log.Fatalln("Must specify some role[s] to remove")
	}

	nodeRoles, err := allNodeRoles()
	if err != nil {
		log.Fatalln("err: ", err)
	}

	myKey, err := selfKey()
	if err != nil {
		log.Fatalln("err: ", err)
	}

	var finalSet []string
	for _, role := range nodeRoles[myKey] {
		if (!StrContains(rmRoles, role)) {
			finalSet = append(finalSet, role)
		} else {
			fmt.Printf("removing role `%s`\n", role)
		}
	}
	roleActualSet(finalSet, c)
}


func allNodeRoles() (map[string][]string, error) {
	roleMap := make(map[string][]string)
	client, _ := api.NewClient(api.DefaultConfig())
	catalog := client.Catalog()
	cascadeServices, _, err := catalog.Service("cascade", "", nil)
	if err != nil {
		return nil, err
	}

	for _, service := range cascadeServices {
		roleMap[ makeKey(service.Node, service.Address) ] = service.ServiceTags
	}
	return roleMap, nil
}

func selfKey() (string, error) {
	client, _ := api.NewClient(api.DefaultConfig())
	agent := client.Agent()

	self, err := agent.Self()

	if err != nil {
		return "", err
	}

	return makeKey(self["Config"]["NodeName"].(string), self["Config"]["AdvertiseAddr"].(string)), nil
}

func makeKey(hostName string, hostAddr string) (string) {
	return fmt.Sprintf("%s (%s)", hostName, hostAddr)
}

func printRole(key string, roles []string) {
	fmt.Println(key + ":")
	for _, role := range roles {
		fmt.Println("  -", role)
	}
}

// StrContains checks if a list contains a string
func StrContains(l []string, s string) bool {
	for _, v := range l {
		if v == s {
			return true
		}
	}
	return false
}
