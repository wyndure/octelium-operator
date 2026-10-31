# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang:1.27.3 AS build

WORKDIR /src

COPY go.mod go.sum ./

RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -buildvcs=false -mod=readonly -trimpath -ldflags="-s -w" -o /manager ./cmd

FROM build AS test

RUN make test-race

FROM gcr.io/distroless/static:nonroot

COPY --from=build --chown=65532:65532 /manager /manager

COPY LICENSE-AGPL-3.0 LICENSE-APACHE /usr/share/licenses/octelium-operator/

USER 65532:65532

ENTRYPOINT ["/manager"]
