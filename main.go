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

package main

import (
	"github.com/boundary/cascade/command"
	"github.com/jwaldrip/odin/cli"
)

var cascade = cli.New("0.0.1", "cascade", cli.ShowUsage)

func init() {
	cascade.AddSubCommands(
		command.Cm,
		command.Node,
		command.Role,
		command.Service,
	)
}

func main() {
	cascade.Start()
}
