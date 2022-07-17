package lecture_clip_server

import (
	"bytes"
	"context"
	"os"
	"sync"
	"time"

	pb "github.com/DominicWuest/Alphie/rpc/lecture_clip_server/lecture_clip_pb"

	"google.golang.org/grpc"
)

// Struct of the gRPC server
type LectureClipServer struct {
	pb.UnimplementedLectureClipServer
	// Base url to send requests to for video fragments
	lectureClipBaseUrl string
	// Active clippers currently tracking active lectures
	activeClippers []*lectureClipper
}

type lectureClipper struct {
	// ID identifying the lecture that is being clipped
	lectureId string
	// Where to send requests to for the video fragments
	roomUrl string
	// Used to stop clipper
	recording bool
	// Buffer holding the recent video fragments for the clip
	buffer *bytes.Buffer
	// Mutex for the buffer to ensure no new fragments are added while reading the buffer for sending
	bufferMutex *sync.Mutex
	// Current position in the buffer
	bufferPos int
}

const (
	// How many video fragments should be cached, decides lectureClipper buffer size
	clipFragmentCacheLength int = 180
)

// Registers the image generation server and initialises needed variables
func Register(srv *grpc.Server) {
	lectureClipBaseUrl := os.Getenv("LECTURE_CLIP_BASE_URL")
	if lectureClipBaseUrl == "" {
		panic("LECTURE_CLIP_BASE_URL environment variable not set")
	}
	pb.RegisterLectureClipServer(srv, &LectureClipServer{
		lectureClipBaseUrl: lectureClipBaseUrl,
		activeClippers:     make([]*lectureClipper, 0),
	})
}

func (s *LectureClipServer) Clip(ctx context.Context, in *pb.ClipRequest) (*pb.ClipResponse, error) {
	return &pb.ClipResponse{}, nil
}

// Should be called as a goroutine, starts recording for the clips
func (s *lectureClipper) startRecording() error {
	// Reset the clipper
	s.buffer = bytes.NewBuffer(make([]byte, clipFragmentCacheLength))
	s.bufferPos = 0
	s.recording = true

	for s.recording {
		time.Sleep(time.Second) // Temporary until logic is implemented
	}

	return nil
}

// Stops the clippers recording
func (s *lectureClipper) stopRecording() error {
	s.recording = false
	return nil
}

// Creates the clip and returns the url where it was stored
func (s *lectureClipper) clip(id string) (string, error) {
	return "", nil
}
