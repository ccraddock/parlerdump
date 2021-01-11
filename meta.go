package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strconv"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/mattetti/filebuffer"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

func main() {
	// Create context to enable cancellation.
	ctx, cancel := context.WithCancel(context.Background())
	// Start goroutine to capture user requesting early shutdown (CTRL+C).
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		// Tell all goroutines that their context has been cancelled.
		cancel()
		// Give some time to clean up gracefully.
		time.Sleep(time.Second * 20)
	}()
	// Determine concurrency level.
	concurrency := os.Getenv("PARLER_CONCURRENCY")
	if concurrency == "" {
		log.Fatal("env PARLER_CONCURRENCY must be specified")
	}
	maxRequests, err := strconv.Atoi(concurrency)
	if err != nil {
		log.Fatal(err)
	}
	// Determine destination bucket.
	bucket := os.Getenv("PARLER_BUCKET")
	if bucket == "" {
		log.Fatal("env PARLER_BUCKET must be specified")
	}
	// Initiate session with aws using standard aws sdk environment variables.
	sess, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
	s3 := s3.New(sess)
	if err != nil {
		log.Fatal(err)
	}
	// Read all URLs from stdin
	var urls []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		urls = append(urls, scanner.Text())
	}
	// Create streaming uploader.
	manager := s3manager.NewUploader(sess, func(u *s3manager.Uploader) {
		u.BufferProvider = s3manager.NewBufferedReadSeekerWriteToPool(25 * 1024 * 1024)
	})
	fmt.Printf("scanned %d urls, extracting metadata to s3 with concurrency of %s\n", len(urls), concurrency)
	// Concurrently process them.
	sem := semaphore.NewWeighted(int64(maxRequests))
	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		for _, url := range urls {
			url := url // https://golang.org/doc/faq#closures_and_goroutines
			if err := sem.Acquire(egCtx, 1); err != nil {
				return err
			}
			eg.Go(func() error {
				defer sem.Release(1)
				// don't die on failures
				if err := meta(egCtx, bucket, url, s3, manager); err != nil {
					fmt.Printf("failure: %s", err)
				}
				return nil
			})
		}
		return nil
	})
	if err := eg.Wait(); err != nil {
		log.Fatal(err)
	}
}

func meta(
	ctx context.Context,
	bucket string,
	url string,
	S3 *s3.S3,
	manager *s3manager.Uploader,
) error {
	srcFile := path.Base(url)
	destFile := "meta-" + srcFile + ".json"
	// prevent uploading the same metadata twice
	_, err := S3.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(destFile),
	})
	if err == nil {
		fmt.Printf("skipping %s\n", url)
		return nil
	}
	object, err := S3.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(srcFile),
	})
	if err != nil {
		// try to archive missed files for second pass
		_ = archive(ctx, bucket, srcFile, S3, manager)
		return err
	}
	defer object.Body.Close()
	var out bytes.Buffer
	cmd := exec.Command("exiftool", "-j", "-")
	cmd.Stdin = object.Body
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	fmt.Printf("recording meta for %s\n", srcFile)
	_, err = S3.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(destFile),
		Body:   filebuffer.New(out.Bytes()),
	})
	return nil
}

func archive(
	ctx context.Context,
	bucket string,
	url string,
	S3 *s3.S3,
	manager *s3manager.Uploader,
) error {
	client := retryablehttp.NewClient()
	client.Logger = log.New(ioutil.Discard, "", 0)
	request, _ := retryablehttp.NewRequest("GET", url, nil)
	req, err := client.Do(request.WithContext(ctx))
	if err != nil {
		return err
	}
	destFile := path.Base(url)
	// prevent uploading the same object twice
	// no checking on size here so it's possible this will leave partial
	// transfers failed.
	_, err = S3.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(destFile),
	})
	if err == nil {
		fmt.Printf("skipping %s\n", url)
		return nil
	}
	fmt.Printf("archiving %s\n", url)
	if _, err := manager.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(destFile),
		Body:   req.Body,
	}); err != nil {
		return err
	}
	return nil
}
