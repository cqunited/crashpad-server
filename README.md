# crashpad server

> crashpad server for chromium

## usage

+ build docker image
```bash
docker build -t reg.example.com/chromium/crashpad:latest .
```
+ run
```bash
docker run -tid \
  --name="crashpad-go" \
  -p 8080:8080 \
  -e S3_ENDPOINT="play.min.io" \
  -e S3_ACCESSKEY="Q3AM3UQ867SPQQA43P2F" \
  -e S3_SECRETKEY="zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG" \
  -e S3_BUCKETNAME="crashpad" \
  -e S3_IS_SSL="true" \
  -e LISTEN_ADDR=":8080" \
  reg.example.com/chromium/crashpad:latest
```
