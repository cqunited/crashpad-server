package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// http
const (
	CONTENT_TYPE        = "Content-Type"
	CONTENT_ENCODING    = "Content-Encoding"
	CONTENT_DISPOSITION = "Content-Disposition"
	GZIP                = "gzip"
	MULTIPART_PREFIX    = "multipart/"
	MULTIPART_BOUNDARY  = "boundary"
)

// env name
const (
	LISTEN_ADDR   = "LISTEN_ADDR"
	S3_ENDPOINT   = "S3_ENDPOINT"
	S3_ACCESSKEY  = "S3_ACCESSKEY"
	S3_SECRETKEY  = "S3_SECRETKEY"
	S3_BUCKETNAME = "S3_BUCKETNAME"
	S3_IS_SSL     = "S3_IS_SSL"
)

// biz
const (
	DMP_FILE_KEY = "upload_file_minidump"
	DMP          = ".dmp"
	JSON         = ".json"
)

func getObjectNamePrefix(guid string) string {
	// date - guid - uuid
	current_time := time.Now()
	time_str := fmt.Sprintf("%d-%02d-%02d-%02d-%02d-%02d",
		current_time.Year(), current_time.Month(), current_time.Day(),
		current_time.Hour(), current_time.Minute(), current_time.Second())
	id := uuid.New()
	return fmt.Sprintf("%s-%s-%s", time_str, guid, id.String())
}

func werror(resp http.ResponseWriter, code int, err string) {
	resp.WriteHeader(code)
	resp.Write([]byte(err))
	log.Println(err)
}

func handler(resp http.ResponseWriter, req *http.Request) {

	compressed := strings.ToLower(req.Header.Get(CONTENT_ENCODING)) == GZIP

	if compressed {
		body, err := gzip.NewReader(req.Body)
		if err != nil {
			werror(resp, http.StatusBadRequest, err.Error())
			return
		}
		defer body.Close()

		mediaType, params, err := mime.ParseMediaType(req.Header.Get(CONTENT_TYPE))
		if err != nil {
			werror(resp, http.StatusBadRequest, err.Error())
			return
		}

		guid := ""
		if val, ok := req.URL.Query()["guid"]; ok {
			if len(val) > 0 {
				guid = val[0]
			}
		}
		if guid == "" {
			werror(resp, http.StatusBadRequest, "guid not present")
			return
		}
		prefix := getObjectNamePrefix(guid)

		metadata := make(map[string]string)
		if strings.HasPrefix(mediaType, MULTIPART_PREFIX) {
			mr := multipart.NewReader(body, params[MULTIPART_BOUNDARY])
			for {
				p, err := mr.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					werror(resp, http.StatusInternalServerError, err.Error())
					return
				}
				slurp, err := io.ReadAll(p)
				if err != nil {
					werror(resp, http.StatusInternalServerError, err.Error())
					return
				}

				_, params, err := mime.ParseMediaType(p.Header.Get(CONTENT_DISPOSITION))
				if err != nil {
					werror(resp, http.StatusBadRequest, "Could not parse meta from content disposition")
					return
				}

				if val, ok := params["name"]; ok {
					if val == DMP_FILE_KEY {
						// dmp
						getClient().PutObject(context.Background(), getOrDefault(S3_BUCKETNAME, "crashpad"), prefix+DMP, bytes.NewReader(slurp), -1, minio.PutObjectOptions{})
					} else {
						// meta
						metadata[val] = string(slurp)
					}
				} else {
					werror(resp, http.StatusBadRequest, "name not present")
					return
				}
			}
		}

		metadata_json, _ := json.MarshalIndent(metadata, "", "\t")
		getClient().PutObject(context.Background(), getOrDefault(S3_BUCKETNAME, "crashpad"), prefix+JSON, bytes.NewReader(metadata_json), -1, minio.PutObjectOptions{})

		resp.Write([]byte(prefix))
		log.Println(prefix)
		return
	}

	werror(resp, http.StatusBadRequest, "Invalid content format")
}

func getOrDefault(key, def string) string {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	return val
}

func getClient() *minio.Client {
	endpoint := getOrDefault(S3_ENDPOINT, "play.min.io")
	accessKeyID := getOrDefault(S3_ACCESSKEY, "Q3AM3UQ867SPQQA43P2F")
	secretAccessKey := getOrDefault(S3_SECRETKEY, "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG")
	useSSL := getOrDefault(S3_IS_SSL, "true") == "true"

	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln(err)
	}
	return minioClient
}

func main() {
	http.HandleFunc("/", handler)
	addr := getOrDefault(LISTEN_ADDR, ":8080")
	log.Printf("Server started at port %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
