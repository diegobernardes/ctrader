# Protobuf
## When it should be updated?
Every time the Docker image or the `protoc-gen-go` version change at the `compile-proto` target in the
[Earthfile](../Earthfile).

## How can I update it?
1. Update the tag version at the git clone command at the `compile-proto` target in the [Earthfile](../Earthfile).
2. In case new messages are added, update the `mapping*` functions at [ctrader.go](../ctrader.go).
3. Run the tests. Ideally the [integration](./testing.md#how-to-execute-integration-tests) ones.
4. Push the changes.
