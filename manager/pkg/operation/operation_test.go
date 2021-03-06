/*
Copyright 2018 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package operation

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"testing"

	container "cloud.google.com/go/container/apiv1"
	"github.com/golang/protobuf/proto"
	containerpb "google.golang.org/genproto/googleapis/container/v1"

	"github.com/golang/protobuf/ptypes"
	"google.golang.org/api/option"

	status "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var _ = io.EOF
var _ = ptypes.MarshalAny
var _ status.Status

type mockClusterManagerServer struct {
	// Embed for forward compatibility.
	// Tests will keep working if more methods are added
	// in the future.
	containerpb.ClusterManagerServer

	reqs []proto.Message

	// If set, all calls return this error.
	err error

	// responses to return if err == nil
	resps []proto.Message
}

func (s *mockClusterManagerServer) GetOperation(ctx context.Context, req *containerpb.GetOperationRequest) (*containerpb.Operation, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	resp := s.resps[0].(*containerpb.Operation)
	s.resps = s.resps[1:len(s.resps)]
	return resp, nil
}

// clientOpt is the option tests should use to connect to the test server.
// It is initialized by TestMain.
var clientOpt option.ClientOption

var (
	mockClusterManager mockClusterManagerServer
)

func TestMain(m *testing.M) {
	flag.Parse()

	serv := grpc.NewServer()
	containerpb.RegisterClusterManagerServer(serv, &mockClusterManager)

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Print(err)
	}
	go serv.Serve(lis)

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		log.Print(err)
	}
	clientOpt = option.WithGRPCConn(conn)

	os.Exit(m.Run())
}

func TestWait(t *testing.T) {
	var status1 containerpb.Operation_Status = containerpb.Operation_RUNNING
	var expectedResponse1 = &containerpb.Operation{
		Status: status1,
	}
	var status2 containerpb.Operation_Status = containerpb.Operation_DONE
	var expectedResponse2 = &containerpb.Operation{
		Status: status2,
	}

	mockClusterManager.err = nil
	mockClusterManager.reqs = nil

	mockClusterManager.resps = append(mockClusterManager.resps[:0], expectedResponse1)
	mockClusterManager.resps = append(mockClusterManager.resps, expectedResponse2)

	var projectId string = "projectId-1969970175"
	var zone string = "zone3744684"
	var operationId string = "operationId-274116877"

	c, err := container.NewClusterManagerClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	opStatus := make(chan Status)

	go Wait(context.Background(), opStatus, 1, c, projectId, zone, operationId)

	resp := <-opStatus

	if want, got := expectedResponse1.Status, resp.Status; want != got {
		t.Errorf("wrong response %q, want %q)", got, want)
	}

	resp = <-opStatus

	if want, got := expectedResponse2.Status, resp.Status; want != got {
		t.Errorf("wrong response %q, want %q)", got, want)
	}

	resp, ok := <-opStatus

	if ok {
		t.Errorf("Channel was not closed when it should have been")
		close(opStatus)
	}
}
