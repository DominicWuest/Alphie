package image_generation_server

import (
	"context"
	"math/rand"
	"time"

	pb "github.com/DominicWuest/Alphie/rpc/image_generation_server/image_generation_pb"
	"github.com/DominicWuest/Alphie/rpc/image_generation_server/image_generators"
	"google.golang.org/grpc"
)

type ImageGenerationServer struct {
	pb.UnimplementedImageGenerationServer
}

type ImageGenerator interface {
	GenerateImage(int64) (string, error)
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

func (s *ImageGenerationServer) Bounce(ctx context.Context, in *pb.ImageRequest) (*pb.ImageResponse, error) {
	seed := getSeed(in)
	var imageGenerator ImageGenerator = &image_generators.Bounce{}
	path, err := imageGenerator.GenerateImage(seed)
	if err != nil {
		return nil, err
	}

	return &pb.ImageResponse{ContentPath: path}, nil
}
