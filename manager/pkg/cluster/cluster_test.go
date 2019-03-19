// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// AUTO-GENERATED CODE. DO NOT EDIT.

package cluster

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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	gstatus "google.golang.org/grpc/status"
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

func (s *mockClusterManagerServer) GetServerConfig(ctx context.Context, req *containerpb.GetServerConfigRequest) (*containerpb.ServerConfig, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*containerpb.ServerConfig), nil
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
		log.Fatal(err)
	}
	go serv.Serve(lis)

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	clientOpt = option.WithGRPCConn(conn)

	os.Exit(m.Run())
}

func TestLatestMasterVersionForReleaseSeries(t *testing.T) {
	var validMasterVersions = []string{
		"1.10.1-gke.5",
		"1.9.2-gke.1",
		"1.9.2-gke.0",
		"1.9.1-gke.0",
	}
	var expectedResponse = &containerpb.ServerConfig{
		ValidMasterVersions: validMasterVersions,
	}

	client, _ := container.NewClusterManagerClient(context.Background(), clientOpt)

	cluster := NewManagedCluster(client, "testing", "hello", "wassup", 0)

	mockClusterManager.err = nil
	mockClusterManager.reqs = nil

	mockClusterManager.resps = append(mockClusterManager.resps[:0], expectedResponse)

	resp, err := cluster.LatestMasterVersionForReleaseSeries(context.Background(), "1.10")

	if err != nil {
		t.Fatal(err)
	}

	if want, got := expectedResponse.ValidMasterVersions[0], resp; want != got {
		t.Errorf("wrong response %q, want %q)", got, want)
	}

	resp, err = cluster.LatestMasterVersionForReleaseSeries(context.Background(), "1.9")

	if err != nil {
		t.Fatal(err)
	}

	if want, got := expectedResponse.ValidMasterVersions[1], resp; want != got {
		t.Errorf("wrong response %q, want %q)", got, want)
	}

	resp, err = cluster.LatestMasterVersionForReleaseSeries(context.Background(), "1.9.1")

	if err != nil {
		t.Fatal(err)
	}

	if want, got := expectedResponse.ValidMasterVersions[3], resp; want != got {
		t.Errorf("wrong response %q, want %q)", got, want)
	}
}

func TestClusterManagerGetServerConfigError(t *testing.T) {
	errCode := codes.PermissionDenied
	mockClusterManager.err = gstatus.Error(errCode, "test error")

	var projectId string = "projectId-1969970175"
	var zone string = "zone3744684"
	var request = &containerpb.GetServerConfigRequest{
		ProjectId: projectId,
		Zone:      zone,
	}

	c, err := container.NewClusterManagerClient(context.Background(), clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.GetServerConfig(context.Background(), request)

	if st, ok := gstatus.FromError(err); !ok {
		t.Errorf("got error %v, expected grpc error", err)
	} else if c := st.Code(); c != errCode {
		t.Errorf("got error code %q, want %q", c, errCode)
	}
	_ = resp
}