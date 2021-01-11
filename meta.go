package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
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
				_ = meta(egCtx, bucket, url, s3)
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
) error {
	srcFile := path.Base(url)
	destFile := srcFile + ".json"
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
		fmt.Fprintf(os.Stdout, "fuck")
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
