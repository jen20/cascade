package main

import (
  "github.com/jwaldrip/odin/cli"
  "github.com/boundary/cascade/command"
)

var cascade = cli.New("0.0.1", "cascade", cli.ShowUsage)

func init(){
  cascade.AddSubCommands(
    command.Cm,
    command.Node,
    command.Role,
    command.Service,
  ) 
}

func main(){
  cascade.Start()
}
