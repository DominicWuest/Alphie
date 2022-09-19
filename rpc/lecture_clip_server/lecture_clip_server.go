package lecture_clip_server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	pb "github.com/DominicWuest/Alphie/rpc/lecture_clip_server/lecture_clip_pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/lib/pq"

	"github.com/robfig/cron"
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
	// All aliases of the clipper
	aliases []string
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

// Calendar weeks of semester start / end
const (
	springSemesterStart = 8
	springSemesterEnd   = 22

	fallSemesterStart = 38
	fallSemesterEnd   = 51
)

var (
	// Active clippers currently tracking active lectures, addressed by their ID
	activeClippersByID map[string]*lectureClipper
	// Active clippers currently tracking active lectures, addressed by their alias
	activeClippersByAlias map[string]*lectureClipper
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
	// Domain of the frontend
	wwwDomain string
	// Which protocol to use when accessing outside facing HTTP services
	httpProto string
)

var cronScheduler *cron.Cron

// Registers the lecture clip server and initialises needed variables
func Register(srv *grpc.Server) {
	lectureClipBaseUrl = os.Getenv("LECTURE_CLIP_BASE_URL")
	if lectureClipBaseUrl == "" {
		panic("LECTURE_CLIP_BASE_URL environment variable not set")
	}

	hostname := os.Getenv("CDN_HOSTNAME")
	port := os.Getenv("CDN_REST_PORT")
	domain := os.Getenv("WWW_DOMAIN")
	proto := os.Getenv("HTTP_PROTO")
	if len(hostname)*len(port)*len(domain)*len(proto) == 0 {
		panic("No CDN_HOSTNAME, CDN_REST_PORT, WWW_DOMAIN or HTTP_PROTO set")
	}
	cdnHostname = hostname
	cdnPort = port
	cdnConnString = "http://" + cdnHostname + ":" + cdnPort + cdnURL

	wwwDomain = domain

	httpProto = proto

	activeClippersByID = make(map[string]*lectureClipper)
	activeClippersByAlias = make(map[string]*lectureClipper)
	clippersMutex = &sync.Mutex{}

	connString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOSTNAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_USER"),
	)
	// Check if DB connection works
	db, err := sql.Open("postgres", connString)
	if err != nil {
		panic(fmt.Sprintln("Error connecting to the database: ", err))
	}
	if !checkDBConnection(db) {
		panic("Couldn't connect to the database, pings timed out")
	}
	if err = initLectureClipperSchedules(db); err != nil {
		panic(fmt.Sprintln("Failed to initialise lecture clipper schedules: ", err))
	}

	pb.RegisterLectureClipServer(srv, &LectureClipServer{})
}

func (s *LectureClipServer) Clip(ctx context.Context, in *pb.ClipRequest) (*pb.ClipResponse, error) {
	// Make sure the clippers are consistent during the clipping
	clippersMutex.Lock()

	lectureId := strings.ToLower(in.GetLectureId())

	// Clip specific lecture
	var clipper *lectureClipper
	// If an index was supplied
	if index, err := strconv.Atoi(lectureId); err == nil {
		clipper = activeClippers[index]
	} else {
		// Try ID
		tmp, found := activeClippersByID[lectureId]
		clipper = tmp
		if !found {
			// Try alias
			tmp, found = activeClippersByAlias[lectureId]
			clipper = tmp
			if !found {
				clippersMutex.Unlock()
				return nil, status.Error(codes.InvalidArgument, "invalid lecture ID")
			}
		}
	}
	clip, err := clipper.clip()
	if err != nil {
		clippersMutex.Unlock()
		return nil, err
	}

	clippersMutex.Unlock()

	// Start posting the new clips to the CDN
	url, err := postClip(clip)
	if err != nil {
		return nil, err
	}
	response := &pb.ClipResponse{
		Id:         &clipper.clipperId,
		ContentUrl: url,
	}

	return response, nil
}

func (s *LectureClipServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListResponse, error) {
	clippersMutex.Lock()
	defer clippersMutex.Unlock()

	res := pb.ListResponse{}

	for i, clipper := range activeClippers {
		res.Ids = append(res.Ids, &pb.ClipperID{
			Index: int32(i),
			Id:    clipper.clipperId,
			Alias: clipper.aliases,
		})
	}

	return &res, nil
}

func createAndStartClipper(clipperId string, aliases []string, roomUrl string, recordingDuration time.Duration) error {
	clipper, err := createClipper(clipperId, aliases, lectureClipBaseUrl+"/"+roomUrl)
	if err != nil {
		return err
	}

	if err := clipper.start(); err != nil {
		return err
	}

	// Insert the clipper into the list of active clippers
	clippersMutex.Lock()

	activeClippers = append(activeClippers, clipper)
	activeClippersByID[strings.ToLower(clipperId)] = clipper

	for _, alias := range aliases {
		activeClippersByAlias[strings.ToLower(alias)] = clipper
	}

	clippersMutex.Unlock()

	// Shut down the clipper after the requested duration
	time.AfterFunc(recordingDuration, func() {
		shutdownClipper(clipper)
	})

	log.Printf("Successfully started the clipper: %s\n", clipper.clipperId)
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

	for _, alias := range clipper.aliases {
		delete(activeClippersByAlias, alias)
	}

	clippersMutex.Unlock()

	log.Printf("Shutting down the clipper: %s\n", clipper.clipperId)
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

	clipUrl := "/clip" + strings.TrimSuffix(strings.TrimPrefix(response.Filename, cdnURL), ".mp4")

	url := httpProto + "://" + wwwDomain + clipUrl

	return url, nil
}

// Checks that the DB is up before querying the schedules
func checkDBConnection(db *sql.DB) bool {
	// Check that DB is up, panic if not
	success := false
	var err error = nil
	// Try to ping 30, retrying every 2 seconds, wait for the DB to boot up first
	for i := 0; i < 30; i++ {
		if err = db.Ping(); err == nil {
			success = true
			break
		}
		time.Sleep(2 * time.Second)
	}
	return success
}

// Queries the schedules form the DB and inits cronjobs for starting the clippers
func initLectureClipperSchedules(db *sql.DB) error {
	const dbTimeout time.Duration = 5 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	cronScheduler = cron.New()

	// Get all the schedules
	rows, err := db.QueryContext(ctx,
		`SELECT * FROM lecture_clippers.clippers
		JOIN lecture_clippers.schedule USING(id)
	`)
	if err != nil {
		return err
	}

	// Init the schedule for every row
	for rows.Next() {
		var (
			id              string
			semester        string
			room_url        string
			schedule        string
			aliases         []string
			durationMinutes int
		)
		if err := rows.Scan(&id, &semester, &room_url, &schedule, &durationMinutes); err != nil {
			return err
		}

		aliasRows, err := db.QueryContext(ctx,
			`SELECT aliases FROM lecture_clippers.lecture_alias
		WHERE id=$1
		`, id)
		if err != nil {
			return err
		}

		aliasRows.Next()
		if err = aliasRows.Scan(pq.Array(&aliases)); err != nil {
			return err
		}
		aliasRows.Close()

		if err := initSchedule(id, semester, room_url, schedule, aliases, durationMinutes); err != nil {
			return err
		}
	}

	cronScheduler.Start()

	return nil
}

// Initialises the scheduled clipper using a cronjob
func initSchedule(id, semester, roomUrl, schedule string, aliases []string, durationMinutes int) error {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	sched, err := parser.Parse(schedule)
	if err != nil {
		return err
	}

	cronScheduler.Schedule(sched, cron.FuncJob(func() {
		// Ensure we're in the right semester
		_, calendarWeek := time.Now().ISOWeek()
		if semester == "F" &&
			(calendarWeek < springSemesterStart || calendarWeek > springSemesterEnd) {
			return
		}
		if semester == "H" &&
			(calendarWeek < fallSemesterStart || calendarWeek > fallSemesterEnd) {
			return
		}
		if semester == "B" &&
			(calendarWeek < springSemesterStart || calendarWeek > springSemesterEnd) &&
			(calendarWeek < fallSemesterStart || calendarWeek > fallSemesterEnd) {
			return
		}

		// Start the clipper
		if err := createAndStartClipper(id, aliases, roomUrl, time.Duration(durationMinutes)*time.Minute); err != nil {
			log.Printf("Failed to start clipper %s with url %s using schedule %s: %v\n", id, roomUrl, schedule, err)
		}
	}))

	return nil
}
