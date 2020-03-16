package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

var extensions = map[string]string{
	".css":  "text/css; charset=utf-8",
	".gif":  "image/gif",
	".htm":  "text/html; charset=utf-8",
	".html": "text/html; charset=utf-8",
	".jpeg": "image/jpeg",
	".jpg":  "image/jpeg",
	".js":   "application/javascript",
	".mjs":  "application/javascript",
	".pdf":  "application/pdf",
	".png":  "image/png",
	".svg":  "image/svg+xml",
	".wasm": "application/wasm",
	".webp": "image/webp",
	".xml":  "text/xml; charset=utf-8",
}

var assetsDir = "/assets/"

// ErrMediaPathNil ...
var ErrMediaPathNil = errors.New("media path can't be nil")

// SvcFiles ...
type SvcFiles struct {
	mediaPath string
}

// MediaPath ...
type MediaPath string

// MyFile ...
type MyFile struct {
	FileName string
	Source   io.Reader
}

// FilePath ...
type FilePath struct {
	Path []string `json:"path"`
}

// NewMediaPath ...
func NewMediaPath() MediaPath {
	return "/assets/"
}

// NewFilesSvc ...
func NewFilesSvc(mediaPath string) *SvcFiles {
	if mediaPath == "" {
		panic(ErrMediaPathNil)
	}

	return &SvcFiles{mediaPath: mediaPath}
}

func main() {
	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "9898"
	}

	dir := http.Dir("./assets/")

	http.Handle("/assets", http.StripPrefix("/assets/", http.FileServer(dir)))

	// uploading images ...
	http.HandleFunc("/api/files", uploadImages)

	// redirect to assets dir ...
	http.HandleFunc("/", redirect)

	http.HandleFunc(assetsDir, serveFiles)

	log.Println("FileServer starting on localhost:9898: ...")
	panic(http.ListenAndServe(":"+port, nil))
}

// redirecting response
func redirect(res http.ResponseWriter, req *http.Request) {
	http.Redirect(res, req, assetsDir, 301)
}

// serveFiles ...
func serveFiles(res http.ResponseWriter, req *http.Request) {
	http.ServeFile(res, req, req.URL.Path[1:])
}

// uploadImages ..
func uploadImages(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		log.Println("Invalid HTTP Method: ", req.Method)
		res.WriteHeader(http.StatusMethodNotAllowed)
	}

	err := req.ParseMultipartForm(10 << 20)
	if err != nil {
		log.Println("ParseMultipartForm Error: ", err)
		res.WriteHeader(http.StatusInternalServerError)
	}

	var files = make([]*multipart.FileHeader, 0)

	if req.MultipartForm != nil {
		files = req.MultipartForm.File["files"]
	}

	svc := NewFilesSvc("/assets/")

	imgs, err := svc.CreateDir(files)
	if err != nil {
		log.Println("CreateDir error: ", err)
		res.WriteHeader(http.StatusInternalServerError)
	}

	filePath := &FilePath{
		Path: imgs,
	}

	path, err := json.Marshal(filePath)
	if err != nil {
		log.Println("Path marshal error: ", err)
		res.WriteHeader(http.StatusInternalServerError)
	}

	res.Header().Set("Content-Type", "application/json")
	res.Write(path)
}

// CreateDir create empty directory by name and add images
func (receiver *SvcFiles) CreateDir(images []*multipart.FileHeader) ([]string, error) {
	filePath := make([]string, 0)

	for _, value := range images {
		file, _ := value.Open()

		extension := strings.Split(value.Filename, ".")
		uuidV4 := uuid.New().String()

		paths := filepath.Join("."+assetsDir, uuidV4+"."+extension[1])

		tmp, err := os.Create(paths)
		if err != nil {
			log.Println("CreateDir os.Create error: ", err)
			return []string{}, err
		}

		filePath = append(filePath, assetsDir+uuidV4+"."+extension[1])

		_, err = io.Copy(tmp, file)
		if err != nil {
			log.Println("CreateDir io.Copy error: ", err)
			return []string{}, err
		}

		defer func() {
			err = tmp.Close()
			if err != nil {
				log.Println("CreateDir tempFile close Error: ", err)
			}
		}()
	}

	return filePath, nil
}
