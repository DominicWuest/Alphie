package lecture_clip_server

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	pb "github.com/DominicWuest/Alphie/rpc/lecture_clip_server/lecture_clip_pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Struct of the gRPC server
type LectureClipServer struct {
	pb.UnimplementedLectureClipServer
	// Base url to send requests to for video fragments
	lectureClipBaseUrl string
	// Active clippers currently tracking active lectures
	activeClippers map[string]*lectureClipper
}

type lectureClipper struct {
	// ID identifying the lecture that is being clipped
	lectureId string
	// Where to send requests to for the video fragments
	roomUrl string
	// Used to stop clipper
	recording bool
	// Used to confirm the clipper stopped
	stopped bool
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
		activeClippers:     make(map[string]*lectureClipper),
	})
}

func (s *LectureClipServer) Clip(ctx context.Context, in *pb.ClipRequest) (*pb.ClipResponse, error) {
	clips := []*pb.Clip{}
	if in.LectureId == nil { // Clip all lectures
		for clipperId, clipper := range s.activeClippers {
			clipUrl, err := clipper.clip()
			if err != nil {
				return nil, err
			}
			clips = append(clips, &pb.Clip{
				Id:          clipperId,
				ContentPath: clipUrl,
			})
		}
	} else { // Clip specific lecture
		clipper, found := s.activeClippers[in.GetLectureId()]
		if !found {
			return nil, status.Error(codes.InvalidArgument, "invalid lecture ID")
		}
		clipUrl, err := clipper.clip()
		if err != nil {
			return nil, err
		}
		clips = append(clips, &pb.Clip{
			Id:          in.GetLectureId(),
			ContentPath: clipUrl,
		})
	}

	return &pb.ClipResponse{
		Clips: clips,
	}, nil
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
	s.stopped = true

	return nil
}

// Stops the clippers recording
func (s *lectureClipper) stopRecording() error {
	s.recording = false
	// Make sure the recorder has stopped
	waitCounter := 50
	waitDuration := 100 * time.Millisecond

	for !s.stopped && waitCounter > 0 {
		time.Sleep(waitDuration)
		waitCounter--
	}

	if waitCounter == 0 {
		return fmt.Errorf("failed to stop recording of %s (timed out)", s.lectureId)
	}

	return nil
}

// Creates the clip and returns the url where it was stored
func (s *lectureClipper) clip() (string, error) {
	return "", nil
}
