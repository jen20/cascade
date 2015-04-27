package command

import (
  "fmt"
  "encoding/json"
  "os"
  "time"

  "github.com/jwaldrip/odin/cli"
  "github.com/hashicorp/consul/api"
)

type CascadeEvent struct {
  Source string `json:"source"`
  Msg string `json:"msg"`
  Ref string `json:"ref"`
}

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
    fmt.Println("Node not managemed by cascade")
    os.Exit(1)
  }

  fmt.Println("Running CM...")
  _cmDispatch(self["Config"]["NodeName"].(string))
}

func cmRoll(c cli.Command) {
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

  fmt.Printf("Rolling (%v) nodes..\n", len(nodes))
  for _,node := range nodes {
    dispatch := _cmDispatch(node.Node)
    if dispatch == "fail" {
      fmt.Println("roll stopped")
      os.Exit(1)
    }
  }
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

  fmt.Println("Running CM...")
  _cmDispatch(c.Arg(0).String())
}

func _cmDispatch(host string) string {
  client, _ := api.NewClient(api.DefaultConfig())
  event := client.Event()

  cascadeEvent := CascadeEvent{"cascade cli", "run", ""}
  payload, _ := json.Marshal(cascadeEvent)
  params := &api.UserEvent{Name: "cascade.cm", Payload: payload, NodeFilter: host}
  
  id, _, err := event.Fire(params, nil)
  
  if err != nil {
    fmt.Println("err: ", err)
    os.Exit(1)
  }

  seen := make(map[string]bool)

  for {
    events, _, err := event.List("cascade.cm", nil)
    if err != nil {
      fmt.Println("err: ", err)
      os.Exit(1)
    }

    for _, event := range events {
      var e CascadeEvent
      err := json.Unmarshal(event.Payload, &e)

      if err != nil {
        fmt.Println("err: ", err)
      }

      if e.Ref == id {
        if !seen[event.ID] {
          if len(seen) == 0 {
            fmt.Println(e.Source)
          }
          fmt.Println("  -", e.Msg)
          seen[event.ID] = true
        }
        if e.Msg == "success" || e.Msg == "fail" {
          return e.Msg
        }
      }
    }

    time.Sleep(1 * time.Second)
  }
}
