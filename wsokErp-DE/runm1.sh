docker build . -t werpm1:latest --no-cache --platform linux/arm64
docker run -ti --rm -p 8080:8080 werpm1
