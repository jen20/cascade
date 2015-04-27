package command

import (
  "fmt"
  "os"

  "github.com/jwaldrip/odin/cli"
  "github.com/hashicorp/consul/api"
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
  case "list": nodeList(c)
  default: cli.ShowUsage(c)
  }
}

func nodeList(c cli.Command) {
  client, _ := api.NewClient(api.DefaultConfig())
  catalog := client.Catalog()

  nodes, meta, err := catalog.Service("cascade", c.Flag("role").String(), nil)

  if err != nil {
    fmt.Println("err: ", err)
    os.Exit(1)
  }

  if meta.LastIndex == 0 {
    fmt.Println("Bad: ", meta)
    os.Exit(1)
  }

  for _,node := range nodes {
    fmt.Println(node.Node, node.Address + ":")
    for _,role := range node.ServiceTags {
      fmt.Println("  -", role)
    }
  }
}
