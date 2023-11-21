VERSION 0.7

ARG --global BASE_IMAGE=golang:1.21-bookworm
FROM $BASE_IMAGE
WORKDIR /opt/ctrader

configure:
  LOCALLY
  RUN git config pull.rebase true \
    && git config remote.origin.prune true \
    && git config branch.main.mergeoptions "--ff-only"

go-base:
  COPY --dir openapi .
  COPY go.mod go.sum *.go .
  RUN go mod download

go-build:
  FROM +go-base
  RUN go build -trimpath ./...

go-test:
  ARG INTEGRATION_TEST="false"
  FROM +go-base
  RUN go install github.com/mfridman/tparse@v0.13.1
  IF [ "$INTEGRATION_TEST" = "true" ]
    RUN --secret CTRADER_CLIENT_ID --secret CTRADER_SECRET --secret CTRADER_ACCOUNT_ID --secret CTRADER_TOKEN \
      go test --tags integration -trimpath -race -cover -covermode=atomic -json ./... | tparse -all -smallscreen
  ELSE
    RUN go test -trimpath -race -cover -covermode=atomic -json ./... | tparse -all -smallscreen
  END

go-linter:
  FROM +go-base
  RUN go install golang.org/x/vuln/cmd/govulncheck@v1.0.1 \
    && go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.0
  COPY .golangci.yaml .
  RUN govulncheck ./... \
    && golangci-lint run ./...

go-mod-linter:
  FROM +go-base
  COPY . .
  RUN git checkout Earthfile \
    && go mod tidy \
    && git add . \
    && git diff --cached --exit-code

update-pkg-go-dev:
  RUN curl https://proxy.golang.org/github.com/diegobernardes/ctrader/@v/main.info

# compile-proto is used to compile cTrader Open API protobuf files. In case the 'protoc-gen-go' version changes, it's
# recommended to run 'go mod tidy'.
compile-proto:
  LOCALLY 
  RUN rm -rf openapi
  FROM $BASE_IMAGE
  RUN apt-get update \
    && apt-get install --yes --no-install-recommends protobuf-compiler=3.* \
    && rm -rf /var/lib/apt/lists/* \
    && go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0
  GIT CLONE --branch 88 https://github.com/spotware/openapi-proto-messages.git openapi-proto-messages
  RUN cd openapi-proto-messages \
    && protoc --go_out=. --go_opt=paths=source_relative *.proto \
    && find . ! \( -name '*.go' \) -delete
  SAVE ARTIFACT openapi-proto-messages AS LOCAL openapi