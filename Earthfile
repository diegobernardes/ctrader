VERSION 0.6

ARG GO_IMAGE=golang:1.20-bullseye
ARG WORKDIR=/opt/ctrader

configure:
  LOCALLY
  RUN git config pull.rebase true \
    && git config remote.origin.prune true \
    && git config branch.main.mergeoptions "--ff-only"

go-test:
  ARG INTEGRATION_TEST="true"
  FROM +go-base
  RUN go install github.com/mfridman/tparse@v0.12.2
  IF [ "$INTEGRATION_TEST" = "true" ]
    RUN --secret CTRADER_CLIENT_ID --secret CTRADER_SECRET --secret CTRADER_ACCOUNT_ID --secret CTRADER_TOKEN \
      go test --tags integration -trimpath -race -cover -covermode=atomic -json ./... | tparse -all -smallscreen
  ELSE
    RUN go test -trimpath -race -cover -covermode=atomic -json ./... | tparse -all
  END

go-build:
  FROM +go-base
  RUN go build -trimpath ./...

go-linter:
  FROM +go-base
  WORKDIR $WORKDIR
  RUN go install golang.org/x/vuln/cmd/govulncheck@latest \
    && go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.52.2
  COPY .golangci.yaml .
  RUN govulncheck ./...
  RUN golangci-lint run ./...

go-mod-linter:
  FROM $GO_IMAGE
  WORKDIR $WORKDIR
  COPY . .
  RUN git checkout Earthfile \
    && go mod tidy \
    && git add . \
    && git diff --cached --exit-code

go-base:
  FROM $GO_IMAGE
  WORKDIR $WORKDIR
  COPY --dir openapi .
  COPY go.mod go.sum *.go .
  RUN go mod download

update-pkg-go-dev:
  FROM $GO_IMAGE
  RUN curl https://proxy.golang.org/github.com/diegobernardes/ctrader/@v/main.info

# compile-proto is used to compile cTrader Open API protobuf files. In case the 'protoc-gen-go' version changes, it's
# recommended to run 'go mod tidy'.
compile-proto:
  LOCALLY 
  RUN rm -rf openapi
  FROM $GO_IMAGE
  RUN apt-get update \
    && apt-get install --yes --no-install-recommends protobuf-compiler=3.* \
    && rm -rf /var/lib/apt/lists/* \
    && go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.30.0
  RUN git clone --depth 1 --branch 86 https://github.com/spotware/openapi-proto-messages.git \
    && cd openapi-proto-messages \
    && protoc --go_out=. --go_opt=paths=source_relative *.proto \
    && find . ! \( -name '*.go' \) -delete
  SAVE ARTIFACT openapi-proto-messages AS LOCAL openapi