package lecture_clip_server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
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
}

type lectureClipper struct {
	// To ensure consistency between clipping and recording
	sync.Mutex
	// The ID of the clipper itself
	clipperId string
	// Where to send requests to for the video fragments
	roomUrl string
	// Used to stop clipper
	recording bool
	// Used to confirm the clipper stopped
	stopped bool
	// Cache holding the recent video fragments for the clip
	cache [][]byte
	// Position of the next entry to the cache, with the index being cachePos % len(cache)
	cachePos int
	// The last media sequence number captured
	seqNum int
}

// Struct for the response we get when posting a clip to the CDN server
type postResponse struct {
	Filename string `json:"filename"`
}

const (
	// How many video fragments should be cached, decides lectureClipper buffer size
	clipFragmentCacheLength int = 180
	// Where to post the clips to on our CDN
	cdnURL string = "/lecture_clips"
)

var (
	// Active clippers currently tracking active lectures
	activeClippersByID map[string]*lectureClipper
	// List of active clippers to address by index
	activeClippers []*lectureClipper
	// Ensure consistency when adding and removing clippers
	clippersMutex *sync.Mutex
)

var (
	// Base URL of the streaming service
	lectureClipBaseUrl string
	// Hostname of the CDN server
	cdnHostname string
	// Port of the CDN server
	cdnPort string
	// Where to post clip requests to
	cdnConnString string
)

// Registers the lecture clip server and initialises needed variables
func Register(srv *grpc.Server) {
	lectureClipBaseUrl = os.Getenv("LECTURE_CLIP_BASE_URL")
	if lectureClipBaseUrl == "" {
		panic("LECTURE_CLIP_BASE_URL environment variable not set")
	}

	hostname := os.Getenv("CDN_HOSTNAME")
	port := os.Getenv("CDN_REST_PORT")
	if len(hostname)*len(port) == 0 {
		panic("No CDN_HOSTNAME or CDN_REST_PORT set")
	}
	cdnHostname = hostname
	cdnPort = port
	cdnConnString = "http://" + cdnHostname + ":" + cdnPort + cdnURL

	activeClippersByID = make(map[string]*lectureClipper)
	clippersMutex = &sync.Mutex{}

	// Temporary test clipper
	testClipper := lectureClipper{
		clipperId: "test",
		roomUrl:   "hg-d-1-1",
	}
	go func() {
		// Shutdown test clipper after 30 seconds
		time.AfterFunc(30*time.Second, func() { fmt.Println("Stopped recording: ", testClipper.stopRecording()) })
		// Start test clipper
		testClipper.startRecording()
	}()

	pb.RegisterLectureClipServer(srv, &LectureClipServer{})
}

func (s *LectureClipServer) Clip(ctx context.Context, in *pb.ClipRequest) (*pb.ClipResponse, error) {
	clips := []*pb.Clip{}
	// Make sure the clippers are consistent during the clipping
	clippersMutex.Lock()
	defer clippersMutex.Unlock()
	if in.LectureId == nil { // Clip all lectures
		for _, clipper := range activeClippers {
			clipUrl, err := clipper.clip()
			if err != nil {
				return nil, err
			}
			clips = append(clips, &pb.Clip{
				Id:          clipper.clipperId,
				ContentPath: clipUrl,
			})
		}
	} else { // Clip specific lecture
		var clipper *lectureClipper
		// If an index was supplied
		if index, err := strconv.Atoi(in.GetLectureId()); err == nil {
			clipper = activeClippers[index]
		} else {
			tmp, found := activeClippersByID[in.GetLectureId()]
			clipper = tmp
			if !found {
				return nil, status.Error(codes.InvalidArgument, "invalid lecture ID")
			}
		}
		clipUrl, err := clipper.clip()
		if err != nil {
			return nil, err
		}
		clips = append(clips, &pb.Clip{
			Id:          clipper.clipperId,
			ContentPath: clipUrl,
		})
	}

	return &pb.ClipResponse{
		Clips: clips,
	}, nil
}

func (s *LectureClipServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListResponse, error) {
	clippersMutex.Lock()
	defer clippersMutex.Unlock()

	res := pb.ListResponse{}

	for i, clipper := range activeClippers {
		res.Ids = append(res.Ids, &pb.ClipperID{
			Index: strconv.Itoa(i),
			Id:    clipper.clipperId,
		})
	}

	return &res, nil
}

// Should be called as a goroutine, starts recording for the clips
func (s *lectureClipper) startRecording() error {
	// Insert the clipper into the list of active clippers
	clippersMutex.Lock()

	activeClippers = append(activeClippers, s)
	activeClippersByID[s.clipperId] = s

	clippersMutex.Unlock()

	// Reset the clipper
	newCache := make([][]byte, clipFragmentCacheLength)
	s.cache = newCache
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
	// Remove the clipper from the list of active clippers
	clippersMutex.Lock()

	var newClippers []*lectureClipper
	for _, clipper := range activeClippers {
		if clipper.clipperId != s.clipperId {
			newClippers = append(newClippers, clipper)
		}
	}
	activeClippers = newClippers
	delete(activeClippersByID, s.clipperId)

	clippersMutex.Unlock()

	// Shut down the clipper
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
	// Capture the clip
	clip := new(bytes.Buffer)

	s.Lock()

	clipEnd := s.cachePos
	clipStart := clipEnd - len(s.cache)
	if clipStart < 0 { // Ensure we don't read unwritten entries
		clipStart = 0
	}

	// Stick fragments together
	for i := clipStart; i < clipEnd; i++ {
		fragment := s.cache[i%len(s.cache)]

		if _, err := clip.Write(fragment); err != nil {
			return "", err
		}
	}

	s.Unlock()

	// Post the clip to the CDN
	res, err := http.Post(cdnConnString, "video/MP2T", clip)
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to post clip: %+v", res)
	}

	// Read where the clip was stored
	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	response := postResponse{}
	if err := json.Unmarshal(content, &response); err != nil {
		return "", err
	}

	return response.Filename, nil
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

	// Fetch all the missing fragments
	for _, item := range missingFragments {
		switch item := item.(type) {
		case *m3u8.SegmentItem:
			if err := s.cachePlaylistItem(item.Segment, s.cachePos%len(s.cache)); err != nil {
				return 0, err
			}
			s.cachePos++
		default:
			return 0, fmt.Errorf("playlist contains element that is not a segment item, cannot handle")
		}
	}

	s.seqNum = playlist.Sequence + playlist.SegmentSize()

	return time.Second, nil
}

// Fetches the item specified by the url and inserts it into the cache at the given index
func (s *lectureClipper) cachePlaylistItem(url string, index int) error {
	itemUrl := lectureClipBaseUrl + "/" + s.roomUrl + "/" + url

	res, err := http.Get(itemUrl)
	if err != nil {
		return err
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	s.Lock()

	s.cache[index] = bytes

	s.Unlock()

	return nil
}
