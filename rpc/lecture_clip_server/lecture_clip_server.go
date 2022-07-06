package lecture_clip_server

import (
	"context"

	pb "github.com/DominicWuest/Alphie/rpc/lecture_clip_server/lecture_clip_pb"

	"google.golang.org/grpc"
)

// Struct of the gRPC server
type LectureClipServer struct {
	pb.UnimplementedLectureClipServer
}

// Registers the image generation server and initialises needed variables
func Register(srv *grpc.Server) {
	pb.RegisterLectureClipServer(srv, &LectureClipServer{})
}

func (s *LectureClipServer) Clip(ctx context.Context, in *pb.ClipRequest) (*pb.ClipResponse, error) {
	return &pb.ClipResponse{}, nil
}
