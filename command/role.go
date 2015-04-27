package command

import (
  "fmt"
  "os"
  
  "github.com/jwaldrip/odin/cli"
  "github.com/hashicorp/consul/api"
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
  case "list": roleList(c)
  case "set": roleSet(c)
  default: cli.ShowUsage(c)
  }
}

func roleList(c cli.Command) {
  client, _ := api.NewClient(api.DefaultConfig())
  agent := client.Agent()

  services, err := agent.Services()

  if err != nil {
    fmt.Println("err: ", err)
    os.Exit(1)
  }

  self, err := agent.Self()

  if err != nil {
    fmt.Println("err: ", err)
    os.Exit(1)
  }

  for _, service := range services {
    if service.Service == "cascade" {
      fmt.Println(self["Config"]["NodeName"], self["Config"]["AdvertiseAddr"].(string) + ":")
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
    fmt.Println("err: ", err)
    os.Exit(1)
  }

  roleList(c)
}
