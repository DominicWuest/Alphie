package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
)

// Allowed content types and their respective file extensions
var contentTypes = map[string]string{
	"image/gif":  "gif",
	"image/jpeg": "jpg",
	"image/png":  "png",
}

func main() {
	const CDN_PATH = "/usr/share/nginx/cdn/"

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Check if method is POST
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Check if request is to a folder (i.e. /images, /lib etc)
		folder := regexp.MustCompile("^/(.+)/?$").FindString(r.URL.EscapedPath())
		if len(folder) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Check if requested folder exists
		if _, err := os.Stat(CDN_PATH + folder); os.IsNotExist(err) {
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
		file_extension, found := contentTypes[r.Header["Content-Type"][0]]
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
		file, err := os.CreateTemp(CDN_PATH+folder, "*."+file_extension)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		file.Chmod(0644)

		// Write to file
		if _, err = file.Write(postData); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Send back 200 OK with the path of the file
		w.WriteHeader(200)
		w.Write([]byte(fmt.Sprintf("{filename:%s}", strings.TrimPrefix(file.Name(), CDN_PATH))))
	})

	port := os.Getenv("CDN_POST_PORT")
	if len(port) == 0 {
		panic("No CDN_POST_PORT specified")
	}
	http.ListenAndServe(":"+port, nil)
}
