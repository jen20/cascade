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
