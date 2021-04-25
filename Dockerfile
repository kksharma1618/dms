FROM golang

RUN \
  apt-get update && \
  DEBIAN_FRONTEND=noninteractive \
    apt-get install -y --no-install-recommends \
      ffmpeg \
      ffmpegthumbnailer \
  && \
  apt-get clean && \
  rm -rf /var/lib/apt/lists/*

COPY . /go/src/github.com/kksharma1618/dms/
WORKDIR /go/src/github.com/kksharma1618/dms/
RUN \
  go build -v .

ENTRYPOINT [ "./dms" ]
