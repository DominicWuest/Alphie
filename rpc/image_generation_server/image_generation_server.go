package image_generation_server

import (
	"context"
	"math/rand"
	"time"

	pb "github.com/DominicWuest/Alphie/rpc/image_generation_server/image_generation_pb"
	"github.com/DominicWuest/Alphie/rpc/image_generation_server/image_generators"

	"google.golang.org/grpc"
)

// Struct of the gRPC server
type ImageGenerationServer struct {
	pb.UnimplementedImageGenerationServer
}

// Registers the image generation server and initialises needed variables
func Register(srv *grpc.Server) {
	image_generators.Init()
	pb.RegisterImageGenerationServer(srv, &ImageGenerationServer{})
}

// Gets the seed from an image request or generates one if not specified
func getSeed(in *pb.ImageRequest) int64 {
	seed := in.GetSeed()
	if in.Seed == nil {
		rand.Seed(time.Now().UnixMilli())
		seed = rand.Int63()
	}
	return seed
}

// Generates an image that bounces a ball around a square
func (s *ImageGenerationServer) Bounce(ctx context.Context, in *pb.ImageRequest) (*pb.ImageResponse, error) {
	return image_generators.GenerateImage(in, &image_generators.Bounce{}, getSeed(in))
}

// Generates a fluid simulation
func (s *ImageGenerationServer) Fluid(ctx context.Context, in *pb.ImageRequest) (*pb.ImageResponse, error) {
	return image_generators.GenerateImage(in, &image_generators.Fluid{}, getSeed(in))
}
