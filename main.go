package main

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	xds "github.com/envoyproxy/go-control-plane/pkg/server/v2"
)

type ADDR struct {
	Address string
	Port    uint32
}

type NodeConfig struct {
	node      *core.Node
	endpoints []types.Resource
	clusters  []types.Resource
	routes    []types.Resource
	listeners []types.Resource
	runtimes  []types.Resource
}

//implement cache.NodeHash
func (n NodeConfig) ID(node *core.Node) string {
	return node.GetId()
}

func ClusterStatic(name string, address []ADDR) *api.Cluster {
	lbEndpoints := make([]*endpoint.LbEndpoint, len(address))
	for idx, addr := range address {
		lbEndpoint := &endpoint.LbEndpoint{
			HostIdentifier: &endpoint.LbEndpoint_Endpoint{
				Endpoint: &endpoint.Endpoint{
					Address: &core.Address{
						Address: &core.Address_SocketAddress{
							SocketAddress: &core.SocketAddress{
								Protocol: core.SocketAddress_TCP,
								Address:  addr.Address,
								PortSpecifier: &core.SocketAddress_PortValue{
									PortValue: addr.Port,
								},
							},
						},
					},
				},
			},
		}
		lbEndpoints[idx] = lbEndpoint
	}

	localityLbEndpoints := &endpoint.LocalityLbEndpoints{
		LbEndpoints: lbEndpoints,
	}

	endpoints := make([]*endpoint.LocalityLbEndpoints, 0)
	endpoints = append(endpoints, localityLbEndpoints)

	clusterLoadAssignment := &api.ClusterLoadAssignment{
		ClusterName: name,
		Endpoints:   endpoints,
	}

	cluster := &api.Cluster{
		Name:        name,
		AltStatName: name,
		ClusterDiscoveryType: &api.Cluster_Type{
			Type: api.Cluster_STATIC,
		},
		EdsClusterConfig:              nil,
		ConnectTimeout:                ptypes.DurationProto(1 * time.Second),
		PerConnectionBufferLimitBytes: nil, // default 1MB
		LbPolicy:                      api.Cluster_ROUND_ROBIN,
		LoadAssignment:                clusterLoadAssignment,
	}
	return cluster
}

func UpdateSnapshotCache(s cache.SnapshotCache, n *NodeConfig, version string) {
	err := s.SetSnapshot(n.ID(n.node), cache.NewSnapshot(version, n.endpoints, n.clusters, n.routes, n.listeners, n.runtimes))
	if err != nil {
		glog.Error(err)
	}
}

//func Update_SnapshotCache(s cache.SnapshotCache, n *NodeConfig
func main() {
	snapshotCache := cache.NewSnapshotCache(false, cache.IDHash{}, nil)
	server := xds.NewServer(context.Background(), snapshotCache, nil)
	grpcServer := grpc.NewServer()
	lis, _ := net.Listen("tcp", ":5678")
	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	api.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	api.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	api.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	api.RegisterListenerDiscoveryServiceServer(grpcServer, server)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			glog.Error(err)
		}
	}()

	node := &core.Node{ // 根据yaml文件中定义的id和名称
		Id:      "envoy-64.58",
		Cluster: "test",
	}
	nodeConf := &NodeConfig{
		node:      node,
		endpoints: []types.Resource{},
		clusters:  []types.Resource{},
		routes:    []types.Resource{},
		listeners: []types.Resource{},
		runtimes:  []types.Resource{},
	}
	input := ""
	{
		clusterName := "Cluster_With_Static_Endpoint"
		fmt.Printf("Enter to update: %s", clusterName)
		_, _ = fmt.Scanf("\n", &input)
		var addrs []ADDR
		addrs = append(addrs, ADDR{
			Address: "127.0.0.1",
			Port:    8081,
		})
		cluster := ClusterStatic(clusterName, addrs)
		nodeConf.clusters = append(nodeConf.clusters, cluster)
		UpdateSnapshotCache(snapshotCache, nodeConf, time.Now().String())
		glog.Info(clusterName + " updated")
	}
	select {}
}
