package lecture_clip_server

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/quangngotan95/go-m3u8/m3u8"
)

const (
	// How many video fragments should be cached, decides lectureClipper buffer size
	clipCacheSize int = 180
)

func createClipper(clipperId string, aliases []string, endpoint string) (*lectureClipper, error) {
	return &lectureClipper{
		clipperId:      clipperId,
		aliases:        aliases,
		streamEndpoint: endpoint,
		cache:          make([][]byte, clipCacheSize),
	}, nil
}

// Starts the clipper
func (s *lectureClipper) start() error {
	s.recording = true

	go func() {
		// Main loop
		for s.recording {
			sleepDuration, err := s.getNewFragments()
			if err != nil {
				s.recording = false
				s.stopped = true
				log.Printf("Error while getting new fragments for %s, stopping clipper: %v\n", s.clipperId, err)
				s.stop()
				return
			}
			time.Sleep(sleepDuration)
		}
		// Confirm we stopped
		s.stopped = true
	}()

	return nil
}

// Shuts down the clipper
func (s *lectureClipper) stop() error {
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
		return fmt.Errorf("failed to stop recording of %s (timed out)", s.streamEndpoint)
	}

	return nil
}

// Creates the clip and returns the url where it was stored
func (s *lectureClipper) clip() ([]byte, error) {
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
			return nil, err
		}
	}

	s.Unlock()

	return clip.Bytes(), nil
}

// Gets the new fragments and returns how long to wait until calling the function again
func (s *lectureClipper) getNewFragments() (time.Duration, error) {
	// TODO: Use url.JoinPath in go v1.19
	// Get the playlist
	playlistUrl := s.streamEndpoint + "/index.m3u8"
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
	itemUrl := s.streamEndpoint + "/" + url

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
