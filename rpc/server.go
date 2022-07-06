package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"

	"github.com/DominicWuest/Alphie/rpc/image_generation_server"
	"github.com/DominicWuest/Alphie/rpc/lecture_clip_server"
)

// Creates the server to listen for gRPC requests
func main() {
	port := os.Getenv("GRPC_PORT")
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	grpcServer := grpc.NewServer()

	image_generation_server.Register(grpcServer)
	lecture_clip_server.Register(grpcServer)

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}
