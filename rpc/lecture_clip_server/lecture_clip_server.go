package lecture_clip_server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	pb "github.com/DominicWuest/Alphie/rpc/lecture_clip_server/lecture_clip_pb"

	"github.com/quangngotan95/go-m3u8/m3u8"
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
	cache *[]*byte
	// Position of the next entry to the cache, with the index being cachePos % len(cache)
	cachePos int
	// The last media sequence number captured
	seqNum int
}

const (
	// How many video fragments should be cached, decides lectureClipper buffer size
	clipFragmentCacheLength int = 180
)

var lectureClipBaseUrl string

// Registers the lecture clip server and initialises needed variables
func Register(srv *grpc.Server) {
	lectureClipBaseUrl = os.Getenv("LECTURE_CLIP_BASE_URL")
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
	newCache := make([]*byte, clipFragmentCacheLength)
	s.cache = &newCache
	s.cachePos = 0
	s.recording = true

	// Main loop
	for s.recording {
		sleepDuration, err := s.getNewFragments()
		if err != nil {
			s.recording = false
			s.stopped = true
			return err
		}
		time.Sleep(sleepDuration)
	}
	// Confirm we stopped
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
	cache := *s.cache
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

// Gets the new fragments and returns how long to wait until calling the function again
func (s *lectureClipper) getNewFragments() (time.Duration, error) {
	// TODO: Use url.JoinPath in go v1.19
	// Get the playlist
	playlistUrl := lectureClipBaseUrl + "/" + s.roomUrl + "/index.m3u8"
	res, err := http.Get(playlistUrl)
	if err != nil {
		return 0, err
	}

	playlist, err := m3u8.Read(res.Body)
	if err != nil {
		return 0, err
	}

	return s.fetchMissingFragments(playlist)
}

// Checks which fragments are still missing and fetches them, returns how long to wait until calling the function again
func (s *lectureClipper) fetchMissingFragments(playlist *m3u8.Playlist) (time.Duration, error) {
	// Slice the fragments from our last seq num to the end
	startingIndex := s.seqNum - playlist.Sequence
	if startingIndex < 0 {
		startingIndex = 0
	}
	missingFragments := playlist.Items[startingIndex:]

	// No new fragments
	if len(missingFragments) == 0 {
		lastItem := playlist.Items[len(playlist.Items)-1].(*m3u8.SegmentItem)
		return time.Duration(lastItem.Duration) * time.Second, nil
	}

	// Playlist items are all Segment Items by specs of the livestreaming service, convert first
	items := []*m3u8.SegmentItem{}
	for _, item := range missingFragments {
		switch item := item.(type) {
		case *m3u8.SegmentItem:
			items = append(items, item)
		default:
			return 0, fmt.Errorf("playlist contains element that is not a segment item, cannot handle")
		}
	}

	fmt.Println(items)
	fmt.Println(items[0].Duration)
	fmt.Println(items[0].Segment)
	fmt.Println(playlist.Sequence)
	fmt.Println(playlist.SegmentSize())

	s.seqNum = playlist.Sequence + playlist.SegmentSize()

	return time.Second, nil
}
