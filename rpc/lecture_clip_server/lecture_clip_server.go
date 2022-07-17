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
	// To ensure consistency between clipping and recording
	sync.Mutex
	// Where to send requests to for the video fragments
	roomUrl string
	// Used to stop clipper
	recording bool
	// Used to confirm the clipper stopped
	stopped bool
	// Cache holding the recent video fragments for the clip
	cache *bytes.Buffer
	// Position of the next entry to the cache, with the index being cachePos % len(cache)
	cachePos int
}

const (
	// How many video fragments should be cached, decides lectureClipper buffer size
	clipFragmentCacheLength int = 180
)

// Registers the lecture clip server and initialises needed variables
func Register(srv *grpc.Server) {
	lectureClipBaseUrl := os.Getenv("LECTURE_CLIP_BASE_URL")
	if lectureClipBaseUrl == "" {
		panic("LECTURE_CLIP_BASE_URL environment variable not set")
	}

	activeClippers := make(map[string]*lectureClipper)
	activeClippers["test"] = &lectureClipper{ // Temporary, used for testing
		roomUrl: "hg-d-1-1",
	}
	go func(clipper *lectureClipper) {
		fmt.Println("Starting test clipper")
		go clipper.startRecording()
		// No graceful shutdown yet, to be implemented
	}(activeClippers["test"])

	pb.RegisterLectureClipServer(srv, &LectureClipServer{
		lectureClipBaseUrl: lectureClipBaseUrl,
		activeClippers:     activeClippers,
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
	s.cache = bytes.NewBuffer(make([]byte, clipFragmentCacheLength))
	s.cachePos = 0
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
		return fmt.Errorf("failed to stop recording of %s (timed out)", s.roomUrl)
	}

	return nil
}

// Creates the clip and returns the url where it was stored
func (s *lectureClipper) clip() (string, error) {
	// Capturing the clip
	s.Lock()

	// Make local copy of needed attributes
	cache := s.cache.Bytes()
	clipEnd := s.cachePos

	s.Unlock()

	clipStart := clipEnd - len(cache)
	if clipStart < 0 { // Ensure we don't read unwritten entries
		clipStart = 0
	}

	// Stick fragments together
	for i := clipStart; i < clipEnd; i++ {
		fragment := cache[i%len(cache)]

		fmt.Println(fragment) // Temporary, suppress unused error
	}

	// Post the clip to the CDN

	return "", nil
}
