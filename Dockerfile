# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang as builder
WORKDIR /app

COPY go.mod .
RUN go mod download

COPY . .
RUN <<EOF
  set -ex
  tag=$(git describe --tags --always --abbrev=8 --dirty)
  if echo "$tag" | grep -q dirty; then exit 1; fi
  file_prefix=$(go env GOMODCACHE)/cache/download/github.com/zhsj/wghttp/@v/$tag
  mkdir -p $(dirname "$file_prefix")
  echo "{\"Version\": \"$tag\"}" > "$file_prefix".info
  cp go.mod "$file_prefix".mod
  git archive --prefix=github.com/zhsj/wghttp@"$tag"/ -o "$file_prefix".zip HEAD
EOF

ARG TARGETARCH
RUN <<EOF
  set -ex
  export CGO_ENABLED=0
  export GOARCH=$TARGETARCH
  export GOPROXY=file://$(go env GOMODCACHE)/cache/download
  export GOSUMDB=off
  tag=$(git describe --tags --abbrev=8 --always)
  go install -trimpath -ldflags="-w -s" github.com/zhsj/wghttp@"$tag"
  cross_bin=/go/bin/$(go env GOOS)_$(go env GOARCH)/wghttp
  if [ -e "$cross_bin" ]; then mv "$cross_bin" /go/bin/wghttp; fi
EOF

FROM scratch
COPY --from=builder /go/bin/wghttp /
ENTRYPOINT ["/wghttp"]
