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

package roll

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/watch"
	"gopkg.in/yaml.v2"
)

const (
	RollKey     = "cascade/roll"
	RunOrderKey = "cascade/run_order"
	ConsulHost  = "127.0.0.1:8500"
)

type CascadeEvent struct {
	Source string `json:"source"`
	Msg    string `json:"msg"`
	Ref    string `json:"ref"`
}

type Roll struct {
	Nodes []string
	Msg   chan string

	client  *api.Client
	session *api.Session
	kv      *api.KV
	event   *api.Event

	sessionID string

	pair  *api.KVPair
	watch *watch.WatchPlan
	curID string
}

func NewRoll(role string) (*Roll, error) {
	client, _ := api.NewClient(api.DefaultConfig())
	session := client.Session()
	kv := client.KV()
	event := client.Event()

	user := os.Getenv("USER")

	if user == "root" && os.Getenv("SUDO_USER") != "" {
		user = os.Getenv("SUDO_USER")
	}

	nodes, err := GetNodes(role)
	if err != nil {
		return nil, err
	}

	se := &api.SessionEntry{
		Name:     "cascade",
		TTL:      "250s",
		Behavior: api.SessionBehaviorDelete,
	}

	sessionID, _, err := session.Create(se, nil)
	if err != nil {
		return nil, err
	}

	value := []byte(user)
	pair := &api.KVPair{Key: "cascade/roll", Value: value, Session: sessionID}
	if work, _, err := kv.Acquire(pair, nil); err != nil {
		return nil, err
	} else if !work {
		pair, _, err := kv.Get(RollKey, nil)
		if err != nil {
			return nil, err
		}

		if pair != nil {
			return nil, errors.New(fmt.Sprintf("err: failed to obtain lock: %s has the lock", string(pair.Value[:])))
		} else {
			return nil, errors.New("err: possibly a stale lock, try again shortly")
		}
	}

	// Setup channel
	msg := make(chan string, 3)

	return &Roll{nodes, msg, client, session, kv, event, sessionID, pair, nil, ""}, nil
}

func (r *Roll) Roll() error {
	for _, node := range r.Nodes {

		// roll the thing
		err := r.Dispatch(node)
		// hack for now (debug possible event dedup, watch exec race)
		time.Sleep(1 * time.Second)

		if err != nil {
			return err
		}

		renew, _, err := r.session.Renew(r.sessionID, nil)
		if err != nil {
			return err
		}

		if renew == nil {
			return errors.New("err: session renewal failed")
		}
	}

	return nil
}

func (r *Roll) Dispatch(host string) error {
	// Setup event
	cascadeEvent := CascadeEvent{"cascade cli", "run", ""}
	payload, _ := json.Marshal(cascadeEvent)
	nodeFilter := fmt.Sprintf("^%s", host)
	params := &api.UserEvent{Name: "cascade.cm", Payload: payload, NodeFilter: nodeFilter}

	var errExit error

	// Setup watch
	watchParams := make(map[string]interface{})
	watchParams["type"] = "event"
	watchParams["name"] = "cascade.cm"

	watch, err := watch.Parse(watchParams)
	if err != nil {
		return err
	}

	r.watch = watch

	// Set handler
	r.watch.Handler = func(idx uint64, data interface{}) {
		events := data.([]*api.UserEvent)

		for _, event := range events {
			var e CascadeEvent
			err := json.Unmarshal(event.Payload, &e)

			if err != nil {
				fmt.Println("err: ", err)
			}

			if e.Ref == r.curID {
				r.Msg <- e.Msg

				if e.Msg == "success" || e.Msg == "fail" {
					r.watch.Stop()

					if e.Msg == "fail" {
						errExit = errors.New("err: failure roll stopped")
					}
				}
			}
		}
	}

	// Fire event
	id, _, err := r.event.Fire(params, nil)

	r.curID = id

	if err != nil {
		return err
	}

	// Send the host we are watching for
	r.Msg <- host

	// Execute Watch
	if err := r.watch.Run(ConsulHost); err != nil {
		return errors.New(fmt.Sprintf("err: querying Consul agent: %s", err))
	}

	return errExit
}

func (r *Roll) Destroy() error {
	r.watch.Stop()

	if work, _, err := r.kv.Release(r.pair, nil); err != nil {
		return err
	} else if !work {
		return errors.New("err: failed to release lock")
	}
	r.session.Destroy(r.sessionID, nil)

	return nil
}

func GetNodes(role string) ([]string, error) {
	client, _ := api.NewClient(api.DefaultConfig())
	catalog := client.Catalog()
	kv := client.KV()
	var err error

	// We have to use arrays to preserve order :(
	seen := make(map[string]bool)
	result := make([]string, 0)

	nodes, _, err := catalog.Service("cascade", role, nil)

	if err != nil {
		return nil, err
	}

	pair, _, err := kv.Get(RunOrderKey, nil)

	if err != nil {
		return nil, err
	}

	if pair == nil {
		for _, node := range nodes {
			result = append(result, node.Node)
		}

		sort.Strings(result)
	} else {
		roles := make([]string, 0)

		err = yaml.Unmarshal([]byte(pair.Value), &roles)
		if err != nil {
			return nil, err
		}

		// TODO this is gross
		for _, role := range roles {
			tmp := make([]string, 0)

			for _, node := range nodes {
				for _, nodeRole := range node.ServiceTags {
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

	if len(result) == 0 {
		err = errors.New(fmt.Sprintf("err: no nodes matching role: %s found", role))

	}

	return result, err
}
