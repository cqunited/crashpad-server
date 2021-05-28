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

func handler(resp http.ResponseWriter, req *http.Request) {

	compressed := strings.ToLower(req.Header.Get("Content-Encoding")) == "gzip"

	if compressed {
		body, err := gzip.NewReader(req.Body)
		if err != nil {
			resp.WriteHeader(400)
			resp.Write([]byte(err.Error()))
			log.Println(err)
			return
		}
		defer body.Close()

		mediaType, params, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
		if err != nil {
			resp.WriteHeader(400)
			resp.Write([]byte(err.Error()))
			log.Println(err)
			return
		}

		// date - guid - uuid
		current_time := time.Now()
		time_str := fmt.Sprintf("%d-%d-%d-%d:%d:%d",
			current_time.Year(), current_time.Month(), current_time.Day(),
			current_time.Hour(), current_time.Minute(), current_time.Second())
		id := uuid.New()
		guid := ""
		if val, ok := req.URL.Query()["guid"]; ok {
			if len(val) > 0 {
				guid = val[0]
			}
		}
		if guid == "" {
			msg := "guid not present"
			resp.WriteHeader(400)
			resp.Write([]byte(msg))
			log.Println(msg)
			return
		}
		prefix := time_str + "-" + guid + "-" + id.String()

		metadata := make(map[string]string)
		if strings.HasPrefix(mediaType, "multipart/") {
			mr := multipart.NewReader(body, params["boundary"])
			for {
				p, err := mr.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					resp.WriteHeader(400)
					resp.Write([]byte(err.Error()))
					log.Println(err)
					return
				}
				slurp, err := io.ReadAll(p)
				if err != nil {
					resp.WriteHeader(400)
					resp.Write([]byte(err.Error()))
					log.Println(err)
					return
				}

				_, params, err := mime.ParseMediaType(p.Header.Get("Content-Disposition"))
				if err != nil {
					resp.WriteHeader(400)
					resp.Write([]byte(err.Error()))
					log.Println(err)
					return
				}

				if val, ok := params["name"]; ok {
					if val == "upload_file_minidump" {
						// dmp
						getClient().PutObject(context.Background(), getOrDefault("S3_BK", "crashpad"), prefix+".dmp", bytes.NewReader(slurp), -1, minio.PutObjectOptions{})
					} else {
						// meta
						metadata[val] = string(slurp)
					}
				} else {
					msg := "name not present"
					resp.WriteHeader(400)
					resp.Write([]byte(msg))
					log.Println(err)
					return
				}
			}
		}

		metadata_json, _ := json.MarshalIndent(metadata, "", "\t")
		getClient().PutObject(context.Background(), getOrDefault("S3_BK", "crashpad"), prefix+".json", bytes.NewReader(metadata_json), -1, minio.PutObjectOptions{})

		resp.Write([]byte(prefix))
		log.Println(prefix)
		return
	}

	msg := "Invalid content format"
	resp.WriteHeader(400)
	resp.Write([]byte(msg))
	log.Println(msg)
}

func getOrDefault(key, def string) string {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	return val
}

func getClient() *minio.Client {
	endpoint := getOrDefault("S3_ENDPOINT", "play.min.io")
	accessKeyID := getOrDefault("S3_AK", "Q3AM3UQ867SPQQA43P2F")
	secretAccessKey := getOrDefault("S3_SK", "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG")
	useSSL := getOrDefault("S3_SSL", "true") == "true"

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
	fmt.Println("Server started at port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
