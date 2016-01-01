// Copyright 2015 CoreOS, Inc.
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

package command

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/coreos/etcd/Godeps/_workspace/src/github.com/spf13/cobra"
	"github.com/coreos/etcd/Godeps/_workspace/src/golang.org/x/net/context"
	"github.com/coreos/etcd/Godeps/_workspace/src/google.golang.org/grpc"
	pb "github.com/coreos/etcd/etcdserver/etcdserverpb"
)

// NewWatchCommand returns the cobra command for "watch".
func NewWatchCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "watch",
		Short: "Watch watches the events happening or happened.",
		Run:   watchCommandFunc,
	}
}

// watchCommandFunc executes the "watch" command.
func watchCommandFunc(cmd *cobra.Command, args []string) {
	endpoint, err := cmd.Flags().GetString("endpoint")
	if err != nil {
		ExitWithError(ExitInvalidInput, err)
	}
	conn, err := grpc.Dial(endpoint)
	if err != nil {
		ExitWithError(ExitBadConnection, err)
	}

	wAPI := pb.NewWatchClient(conn)
	wStream, err := wAPI.Watch(context.TODO())
	if err != nil {
		ExitWithError(ExitBadConnection, err)
	}

	go recvLoop(wStream)

	reader := bufio.NewReader(os.Stdin)

	for {
		l, err := reader.ReadString('\n')
		if err != nil {
			ExitWithError(ExitInvalidInput, fmt.Errorf("Error reading watch request line: %v", err))
		}
		l = strings.TrimSuffix(l, "\n")

		// TODO: support start and end revision
		segs := strings.Split(l, " ")
		if len(segs) != 2 {
			fmt.Fprintf(os.Stderr, "Invalid watch request format: use watch key or watchprefix prefix\n")
			continue
		}

		var r *pb.WatchRequest
		switch segs[0] {
		case "watch":
			r = &pb.WatchRequest{CreateRequest: &pb.WatchCreateRequest{Key: []byte(segs[1])}}
		case "watchprefix":
			r = &pb.WatchRequest{CreateRequest: &pb.WatchCreateRequest{Prefix: []byte(segs[1])}}
		default:
			fmt.Fprintf(os.Stderr, "Invalid watch request format: use watch key or watchprefix prefix\n")
			continue
		}

		err = wStream.Send(r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error sending request to server: %v\n", err)
		}
	}
}

func recvLoop(wStream pb.Watch_WatchClient) {
	for {
		resp, err := wStream.Recv()
		if err == io.EOF {
			os.Exit(ExitSuccess)
		}
		if err != nil {
			ExitWithError(ExitError, err)
		}
		evs := resp.Events
		for _, ev := range evs {
			fmt.Printf("%s: %s %s\n", ev.Type, string(ev.Kv.Key), string(ev.Kv.Value))
		}
	}
}
