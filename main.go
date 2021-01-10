package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path"
	"strconv"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/hashicorp/go-retryablehttp"
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
	if err != nil {
		log.Fatal(err)
	}
	// Create streaming uploader.
	manager := s3manager.NewUploader(sess, func(u *s3manager.Uploader) {
		u.BufferProvider = s3manager.NewBufferedReadSeekerWriteToPool(25 * 1024 * 1024)
	})
	// Read all URLs from stdin and upload them to s3.
	var urls []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		urls = append(urls, scanner.Text())
	}
	fmt.Printf("scanned %d urls, uploading to s3 with concurrency of %s\n", len(urls), concurrency)
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
				return archive(egCtx, bucket, url, manager)
			})
		}
		return nil
	})
	if err := eg.Wait(); err != nil {
		log.Fatal(err)
	}
}

func archive(
	ctx context.Context,
	bucket string,
	url string,
	manager *s3manager.Uploader,
) error {
	client := retryablehttp.NewClient()
	client.Logger = log.New(ioutil.Discard, "", 0)
	request, _ := retryablehttp.NewRequest("GET", url, nil)
	req, err := client.Do(request.WithContext(ctx))
	if err != nil {
		return err
	}
	fmt.Printf("archiving %s\n", url)
	if _, err := manager.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path.Base(url)),
		Body:   req.Body,
	}); err != nil {
		return err
	}
	return nil
}
