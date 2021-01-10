# parlerdump
> use work from https://github.com/d0nk to upload all videos from parler to s3 fast

## Setup
1. Clone this repo.
2. Ensure you have golang 1.15+ installed.
3. Create s3 bucket.
4. Run `archive.sh` configured to use an awscli profile that has access to the
   bucket you created. Use the maximum concurrency your network connection can
   sustain. All files will be streamed to s3.
   ```
   AWS_PROFILE=parler \
   PARLER_BUCKET=parlerdump \
   PARLER_CONCURRENCY=20 \
    ./archive.sh
```