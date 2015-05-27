package command

import (
  "fmt"
  "os"
  "os/signal"

  "github.com/jwaldrip/odin/cli"
  "github.com/hashicorp/consul/api"
  
  "github.com/boundary/cascade/roll"
)

var Cm = cli.NewSubCommand("cm", "Config management operations", cmRun)

func init() {
  Cm.DefineParams("action")
  Cm.DefineStringFlag("role", "", "filter by role")
  Cm.AliasFlag('r', "role")

  Cm.SetLongDescription(`
Run CM on member systems

Actions:
  roll - ordered synchronous run
  local - run CM locally only
  single <nodename> - run on single remote node
  `)
}

func cmRun(c cli.Command) {
  switch c.Param("action").String() {
  case "local": cmLocal(c)
  case "roll": cmRoll(c)
  case "single": cmSingle(c)
  default: cli.ShowUsage(c)
  }
}

func cmLocal(c cli.Command) {
  client, _ := api.NewClient(api.DefaultConfig())
  agent := client.Agent()

  self, err := agent.Self()

  if err != nil {
    fmt.Println("err: ", err)
    os.Exit(1)
  }

  services, err := agent.Services()
  if err != nil {
    fmt.Println("err: ", err)
    os.Exit(1)
  }

  if _, ok := services["cascade"]; !ok {
    fmt.Println("Node not managed by cascade")
    os.Exit(1)
  }
  
  cmRunRoll("", self["Config"]["NodeName"].(string))
}

func cmRoll(c cli.Command) {
  cmRunRoll(c.Flag("role").String(), "")
}

func cmSingle(c cli.Command) {
  client, _ := api.NewClient(api.DefaultConfig())
  catalog := client.Catalog()

  node, _, err := catalog.Node(c.Arg(0).String(), nil)

  if err != nil {
    fmt.Println("err: ", err)
    os.Exit(1)
  }

  if node == nil {
    fmt.Println("node not found")
    os.Exit(1)
  }

  if node.Services["cascade"] == nil {
    fmt.Println("node not managed by cascade")
    os.Exit(1)
  }

  cmRunRoll("", c.Arg(0).String())
}

func cmRunRoll(role string, host string) {
  roller, err := roll.NewRoll(role)
  defer roller.Destroy()

  if err != nil {
    fmt.Println(err)
    os.Exit(1)
  }

  if host != "" {
    roller.Nodes = []string{host}
  }

  // Setup interupt channel
  ch := make(chan os.Signal, 1)
  signal.Notify(ch, os.Interrupt)
  go func(){
    <-ch
    roller.Destroy()
    os.Exit(0)
  }()

  // Setup render channel
  go func(){
    for msg := range roller.Msg {
      switch msg {
        case "meta", "start", "success", "fail": fmt.Println("  -", msg)
        default: fmt.Printf("%s:\n", msg)
      }
    }
  }()

  fmt.Printf("Rolling (%v) nodes..\n", len(roller.Nodes))
  
  err = roller.Roll()
  if err != nil {
    roller.Destroy()
    fmt.Println(err)
    os.Exit(1)
  }
}