# Protobuf
## How can I update it?
1. Update the tag version at the git clone command under the `compile-proto` target at the [Earthfile](../Earthfile).
2. In case new messages are added, update the corresponding `mapping*` functions at [ctrader.go](../ctrader.go).
3. Run the tests with `earthly +go-test`, ideally with
[integration tests](./testing.md#how-to-execute-integration-tests).
4. Push the changes.
