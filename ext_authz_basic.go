package main

import (
	"flag"
	"fmt"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"golang.org/x/net/context"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const (
	testedHeaderName    = "tested-header"
	generatedHeaderName = "generated-header"
	resultHeader        = "x-ext-authz-check-result"
	resultDenied        = "denied"
)

var (
	grpcPort = flag.String("grpc", "9000", "gRPC server port")
	denyBody = "denied by ext_authz"
)

type (
	extAuthzServerV3 struct{}
)

func (s *extAuthzServerV3) logRequest(allow string, request *authv3.CheckRequest) {
	httpAttrs := request.GetAttributes().GetRequest().GetHttp()
	log.Printf("[gRPC v3][%s]: %s%s, attributes: %v\n",
		allow,
		httpAttrs.GetHost(),
		httpAttrs.GetPath(),
		request.GetAttributes())
}

func (s *extAuthzServerV3) allow(request *authv3.CheckRequest) *authv3.CheckResponse {
	s.logRequest("[gRPC v3] allowed", request)
	return &authv3.CheckResponse{
		HttpResponse: &authv3.CheckResponse_OkResponse{
			OkResponse: &authv3.OkHttpResponse{
				Headers: []*corev3.HeaderValueOption{
					{
						Header: &corev3.HeaderValue{
							Key:   generatedHeaderName,
							Value: "hello world",
						},
					},
				},
			},
		},
		Status: &status.Status{Code: int32(codes.OK)},
	}
}

func (s *extAuthzServerV3) deny(request *authv3.CheckRequest) *authv3.CheckResponse {
	s.logRequest("[gRPC v3] denied", request)
	return &authv3.CheckResponse{
		HttpResponse: &authv3.CheckResponse_DeniedResponse{
			DeniedResponse: &authv3.DeniedHttpResponse{
				Status: &typev3.HttpStatus{Code: typev3.StatusCode_Forbidden},
				Body:   denyBody,
				Headers: []*corev3.HeaderValueOption{
					{
						Header: &corev3.HeaderValue{
							Key:   resultHeader,
							Value: resultDenied,
						},
					},
				},
			},
		},
		Status: &status.Status{Code: int32(codes.PermissionDenied)},
	}
}

func (s *extAuthzServerV3) Check(_ context.Context, request *authv3.CheckRequest) (*authv3.CheckResponse, error) {
	log.Printf("[gRPC v3] Check")
	attrs := request.GetAttributes()

	atHeader, contains := attrs.GetRequest().GetHttp().GetHeaders()[testedHeaderName]
	log.Printf("%v: %v", testedHeaderName, atHeader)
	if contains {

		return s.allow(request), nil
	}

	return s.deny(request), nil
}

// ExtAuthzServer implements the ext_authz v3 gRPC
type ExtAuthzServer struct {
	grpcServer *grpc.Server
	grpcV3     *extAuthzServerV3
	grpcPort   chan int
}

func (s *ExtAuthzServer) startGRPC(address string, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
		log.Printf("Stopped gRPC server")
	}()

	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Failed to start gRPC server: %v", err)
	}
	// Store the port for test only.
	s.grpcPort <- listener.Addr().(*net.TCPAddr).Port

	s.grpcServer = grpc.NewServer()
	authv3.RegisterAuthorizationServer(s.grpcServer, s.grpcV3)

	log.Printf("Starting gRPC server at %s", listener.Addr())
	if err := s.grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC server: %v", err)
	}
}

func (s *ExtAuthzServer) run(grpcAddr string) {
	var wg sync.WaitGroup
	wg.Add(1)
	go s.startGRPC(grpcAddr, &wg)
	wg.Wait()
}

func main() {
	flag.Parse()
	s := &ExtAuthzServer{
		grpcV3:   &extAuthzServerV3{},
		grpcPort: make(chan int, 1),
	}
	go s.run(fmt.Sprintf(":%s", *grpcPort))
	defer func() {
		s.grpcServer.Stop()
		log.Printf("GRPC server stopped")
	}()

	// Wait for the process to be shutdown.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
}
