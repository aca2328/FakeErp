docker build . -t werp86:latest --no-cache --platform linux/amd64
docker run -ti --rm -p 8080:8080 werp86
