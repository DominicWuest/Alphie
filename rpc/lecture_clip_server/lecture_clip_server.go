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
	streamEndpoint string
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
	err := createAndStartClipper("test", "hg-f-1", 30*time.Second)
	if err != nil {
		fmt.Println("Failed to start test clipper: ", err)
	}

	pb.RegisterLectureClipServer(srv, &LectureClipServer{})
}

func (s *LectureClipServer) Clip(ctx context.Context, in *pb.ClipRequest) (*pb.ClipResponse, error) {
	clips := [][]byte{}
	clipperIds := []string{}
	// Make sure the clippers are consistent during the clipping
	clippersMutex.Lock()

	if in.LectureId == nil { // Clip all lectures
		for _, clipper := range activeClippers {
			clip, err := clipper.clip()
			if err != nil {
				return nil, err
			}
			clips = append(clips, clip)
			clipperIds = append(clipperIds, clipper.clipperId)
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
		clip, err := clipper.clip()
		if err != nil {
			return nil, err
		}
		clips = append(clips, clip)
		clipperIds = append(clipperIds, clipper.clipperId)
	}

	clippersMutex.Unlock()

	// Start posting the new clips to the CDN
	response := []*pb.Clip{}

	for i := range clips {
		url, err := postClip(clips[i])
		if err != nil {
			return nil, err
		}
		response = append(response, &pb.Clip{
			Id:          clipperIds[i],
			ContentPath: url,
		})
	}

	return &pb.ClipResponse{
		Clips: response,
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

func createAndStartClipper(clipperId, roomUrl string, recordingDuration time.Duration) error {
	clipper, err := createClipper(clipperId, lectureClipBaseUrl+"/"+roomUrl)
	if err != nil {
		return err
	}

	if err := clipper.start(); err != nil {
		return err
	}

	// Insert the clipper into the list of active clippers
	clippersMutex.Lock()

	activeClippers = append(activeClippers, clipper)
	activeClippersByID[clipperId] = clipper

	clippersMutex.Unlock()

	// Shut down the clipper after the requested duration
	time.AfterFunc(recordingDuration, func() {
		shutdownClipper(clipper)
	})

	return nil
}

func shutdownClipper(clipper *lectureClipper) error {
	// Remove the clipper from the list of active clippers
	clippersMutex.Lock()

	var newClippers []*lectureClipper
	for _, i := range activeClippers {
		if i.clipperId != clipper.clipperId {
			newClippers = append(newClippers, i)
		}
	}
	activeClippers = newClippers
	delete(activeClippersByID, clipper.clipperId)

	clippersMutex.Unlock()

	// Stop the clipper itself
	return clipper.stop()
}

// Sends the clip to the CDN and returns its filename
func postClip(clip []byte) (string, error) {
	// Post the clip to the CDN
	res, err := http.Post(cdnConnString, "video/MP2T", bytes.NewBuffer(clip))
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
