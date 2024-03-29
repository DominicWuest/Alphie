package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
)

// Allowed content types and their respective file extensions
var contentTypes = map[string]string{
	"image/gif":  "gif",
	"image/jpeg": "jpg",
	"image/png":  "png",
	"video/MP2T": "mp4", // Will be a .ts file at first, but then converted to mp4
}

var cdn_path = os.Getenv("CDN_ROOT")

func main() {
	if len(cdn_path) == 0 {
		panic("No CDN_ROOT specified")
	}

	// Create folders where content gets stored if they don't exist
	folders := []string{"bounce"}
	for _, folder := range folders {
		folderPath := path.Join(cdn_path, folder)
		_, err := os.Stat(folderPath)
		if os.IsNotExist(err) {
			errDir := os.MkdirAll(folderPath, 0755)
			if errDir != nil {
				panic(err)
			}
		}
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Handle accepted methods
		if r.Method == http.MethodPost {
			handlePost(w, r)
		} else if r.Method == http.MethodDelete {
			handleDelete(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	port := os.Getenv("CDN_REST_PORT")
	if len(port) == 0 {
		panic("No CDN_REST_PORT specified")
	}
	http.ListenAndServe(":"+port, nil)
}

func handlePost(w http.ResponseWriter, r *http.Request) {
	// Check if request is to a folder (i.e. /images, /lib etc)
	folder := regexp.MustCompile("^/(.+)/?$").FindString(r.URL.EscapedPath())
	if len(folder) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Check if requested folder exists
	if _, err := os.Stat(cdn_path + folder); os.IsNotExist(err) {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Check that content-type is set
	contentType := r.Header["Content-Type"]
	if len(contentType) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check for valid content type
	file_extension, found := contentTypes[contentType[0]]
	if !found {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Read data
	postData, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Create file with random name
	file, err := os.CreateTemp(path.Join(cdn_path, folder), "*."+file_extension)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	file.Chmod(0644)

	// Convert transport stream to mp4
	if contentType[0] == "video/MP2T" {
		// Write ts to temp file
		tsOut, err := os.CreateTemp("", "*.ts")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		tsOut.Write(postData)

		// Convert to mp4 and write to temporary file already created
		in := tsOut.Name()
		out := file.Name()

		cmd := exec.Command("ffmpeg",
			"-i", in,
			"-c:v", "libx264",
			"-c:a", "aac",
			"-y",
			"-preset", "ultrafast",
			out,
		)

		if err := cmd.Run(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else if _, err = file.Write(postData); err != nil { // Write to file
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Send back 200 OK with the path of the file
	w.WriteHeader(200)
	w.Write([]byte(fmt.Sprintf(`{"filename":"%s"}`, strings.TrimPrefix(file.Name(), cdn_path))))
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	// Check if request is to a file in a folder (i.e. /images/213123.gif, /lib/0913875.jpg etc)
	file := regexp.MustCompile("^/(.+/.+)/?$").FindString(r.URL.EscapedPath())
	if len(file) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Check if requested file exists
	if _, err := os.Stat(cdn_path + file); os.IsNotExist(err) {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Delete file
	if err := os.Remove(cdn_path + file); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
