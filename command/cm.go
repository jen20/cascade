package command

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/hashicorp/consul/api"
	"github.com/jwaldrip/odin/cli"

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
	case "local":
		cmLocal(c)
	case "roll":
		cmRoll(c)
	case "single":
		cmSingle(c)
	default:
		cli.ShowUsage(c)
	}
}

func cmLocal(c cli.Command) {
	client, _ := api.NewClient(api.DefaultConfig())
	agent := client.Agent()

	self, err := agent.Self()

	if err != nil {
		log.Fatalln("err: ", err)
	}

	services, err := agent.Services()
	if err != nil {
		log.Fatalln("err: ", err)
	}

	if _, ok := services["cascade"]; !ok {
		log.Fatalln("Node not managed by cascade")
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
		log.Fatalln("err: ", err)
	}

	if node == nil {
		log.Fatalln("node not found")
	}

	if node.Services["cascade"] == nil {
		log.Fatalln("node not managed by cascade")
	}

	cmRunRoll("", c.Arg(0).String())
}

func cmRunRoll(role string, host string) {
	roller, err := roll.NewRoll(role)
	defer roller.Destroy()

	if err != nil {
		log.Fatalln("Err: ", err)
	}

	if host != "" {
		roller.Nodes = []string{host}
	}

	// Setup interupt channel
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		roller.Destroy()
		os.Exit(0)
	}()

	// Setup render channel
	go func() {
		for msg := range roller.Msg {
			switch msg {
			case "meta", "start", "success", "fail":
				fmt.Println("  -", msg)
			default:
				fmt.Printf("%s:\n", msg)
			}
		}
	}()

	fmt.Printf("Rolling (%v) nodes..\n", len(roller.Nodes))

	err = roller.Roll()
	if err != nil {
		roller.Destroy()
		log.Fatal("Roll err:", err)
	}
}
