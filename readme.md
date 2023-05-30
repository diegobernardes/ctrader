# cTrader Go SDK
This repository contains a Go SDK to consume [cTrader OpenAPI](https://help.ctrader.com/open-api).

## Requirements
- Go 1.20 or higher.
- Earthly (optional).

## Usage
```go
func main() {
  // Information needed to use the SDK.
  var (
    ctraderClientID  = ""
    ctraderSecret    = ""
    ctraderAccountID = 0
    ctraderToken     = ""
  )

  // Handler for asynchronous events.
  fn := func(msg proto.Message) {
		fmt.Println(msg)
	}

  // Create the transport.
  transport := NewTransportTCP(time.Second)

  // Config and start the client.
	c := Client[*TransportTCP]{
    Transport:    transport,
    Live:         false,
    ClientID:     ctraderClientID,
    Secret:       ctraderSecret,
    HandlerEvent: fn,
  }
  if err := c.Start(); err != nil {
    panic(err)
  }

  // Authenticate the account.
  if _, err = c.AccountAuth(context.Background(), ctraderAccountID, ctraderToken); err != nil {
		panic(err)
	}

  // From this point on you can use the API.
  resp, err := c.SymbolList(context.Background(), ctraderAccountID)
	if err != nil {
		panic(err)
	}
  fmt.Println(resp)
}
```

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

## Acknowledgments
* [ty2/ctrader-go](https://github.com/ty2/ctrader-go)
