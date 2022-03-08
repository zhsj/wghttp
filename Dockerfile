FROM --platform=$BUILDPLATFORM golang as builder
WORKDIR /app
COPY . .
RUN go mod vendor
ARG TARGETARCH
RUN CGO_ENABLED=0 GOARCH=$TARGETARCH go build -v -trimpath -ldflags="-w -s" .

FROM scratch
COPY --from=builder /app/wghttp /
ENTRYPOINT ["/wghttp"]
