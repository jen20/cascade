package command

import (
  "fmt"
  "encoding/json"
  "os"
  "os/signal"
  "time"
  "sort"

  "github.com/jwaldrip/odin/cli"
  "github.com/hashicorp/consul/api"
  "gopkg.in/yaml.v2"
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
  session := client.Session()
  kv := client.KV()

  nodes := _cmNodes(c.Flag("role").String())

  se := &api.SessionEntry{
    Name: "cascade",
    TTL: "250s",
    Behavior: api.SessionBehaviorDelete,
  }

  session_id, _, err := session.Create(se, nil)
  if err != nil {
    fmt.Println("err: ", err)
    os.Exit(1)
  }
  defer session.Destroy(session_id, nil)

  key := "cascade/roll"
  user := os.Getenv("USER")

  if user == "root" && os.Getenv("SUDO_USER") != "" {
    user = os.Getenv("SUDO_USER")
  }

  value := []byte(user)
  p := &api.KVPair{Key: key, Value: value, Session: session_id}
  if work, _, err := kv.Acquire(p, nil); err != nil {
    fmt.Println("err: ", err)
    os.Exit(1)
  } else if !work {
    fmt.Println("Failed to acquire roll lock")
    
    pair, _, err := kv.Get(key, nil)
    if err != nil {
      fmt.Println("err: ", err)
      os.Exit(1)
    }
    
    if pair != nil {
      fmt.Println("user:", string(pair.Value[:]), "has the lock")
    } else {
      fmt.Println("err: possibly a stale lock, try again shortly")
    }
    
    os.Exit(1)
  }

  ch := make(chan os.Signal, 1)
  signal.Notify(ch, os.Interrupt)
  go func(){
    for range ch {
      _cmCleanUp(kv, p)
      os.Exit(0)
    }
  }()

  fmt.Printf("Rolling (%v) nodes..\n", len(nodes))
  for _,node := range nodes {
    dispatch := _cmDispatch(node)
    if dispatch == "fail" {
      fmt.Println("roll stopped")
      _cmCleanUp(kv, p)
      os.Exit(1)
    }

    renew, _, err := session.Renew(session_id, nil)
    if err != nil {
      fmt.Println("err: ", err)
      os.Exit(1)
    }

    if renew == nil {
       fmt.Println("session renewal failed")
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

func _cmNodes(role string) []string {
  client, _ := api.NewClient(api.DefaultConfig())
  catalog := client.Catalog()
  kv := client.KV()

  // We have to use arrays to preserve order :(
  seen := make(map[string]bool)
  result := make([]string, 0)

  nodes, _, err := catalog.Service("cascade", role, nil)

  if err != nil {
    fmt.Println("err: ", err)
    os.Exit(1)
  }

  pair, _, err := kv.Get("cascade/run_order", nil)
  
  if err != nil {
    fmt.Println("err: ", err)
    os.Exit(1)
  }
  
  if pair == nil {
    for _,node := range nodes {
      result = append(result, node.Node)
    }

    sort.Strings(result)
  } else {
    roles := make([]string, 0)

    err = yaml.Unmarshal([]byte(pair.Value), &roles)
    if err != nil {
      fmt.Println("err: ", err)
      os.Exit(1)
    }

    // TODO fix it, this is gross
    for _,role := range roles {
      tmp := make([]string, 0)

      for _,node := range nodes {
        for _,nodeRole := range node.ServiceTags {
          if role == nodeRole && !seen[node.Node] {
            seen[node.Node] = true
            tmp = append(tmp, node.Node)
          }
        }
      }

      sort.Strings(tmp)
      result = append(result, tmp...)
    }
  }

  return result
}

func _cmCleanUp(kv *api.KV, p *api.KVPair) {
  if work, _, err := kv.Release(p, nil); err != nil {
    fmt.Println("err: ", err)
    os.Exit(1)
  } else if !work {
    fmt.Println("failed to release lock")
    os.Exit(1)
  }
}
