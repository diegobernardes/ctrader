# cTrader Go SDK
This repository contains a Go SDK to consume [cTrader OpenAPI](https://help.ctrader.com/open-api).
<img src="docs/gopher.svg" align="right" height="204px" width="200px" />

## Requirements
- Go 1.21 or higher.
- Earthly (optional).

## Usage
Check the `_test.go` files.

## Testing
```shell
# Set the following environment variables:
# - CTRADER_CLIENT_ID
# - CTRADER_SECRET
# - CTRADER_ACCOUNT_ID
# - CTRADER_TOKEN,

# Execute the tests directly with Go:
go test -tags integration -race ./...

# Or you can execute using Earthly:
earthly --secret CTRADER_CLIENT_ID="$CTRADER_CLIENT_ID" \
  --secret CTRADER_SECRET="$CTRADER_SECRET" \
  --secret CTRADER_ACCOUNT_ID="$CTRADER_ACCOUNT_ID" \
  --secret CTRADER_TOKEN="$CTRADER_TOKEN" \
  +go-test
```

## FAQ
### How to register an application?
Follow [this](https://help.ctrader.com/open-api/creating-new-app/#register-your-application) instructions.

## How to get an access ID and secret?
The easiest way is to use the
[playground](https://help.ctrader.com/open-api/account-authentication/#using-the-playground).

### How can I upgrade cTrader OpenAPI protobuf files?
- Open the [Earthfile](https://github.com/diegobernardes/ctrader/blob/main/Earthfile.md) and edit the 
`+compile-proto` target.
- Execute the target `earthly +compile-proto`.
- Sync the dependencies `go mod tidy`.
- Ensure the package still compiles `go build ./...`.
- Open a pull request.

## Documentation
- [Protobuf](./docs/protobuf.md)
- [Testing](./docs/testing.md)

## Acknowledgments
* [ty2/ctrader-go](https://github.com/ty2/ctrader-go)
* [MariaLetta/free-gophers-pack](https://github.com/MariaLetta/free-gophers-pack)
