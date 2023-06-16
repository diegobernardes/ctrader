# Testing
## How to execute integration tests?
```shell
# Be careful with the secrets being logged at the history file. Other ways to pass the secret can be found at 
# https://docs.earthly.dev/docs/guides/build-args#setting-secret-values.
export CTRADER_CLIENT_ID=''
export CTRADER_SECRET=''
export CTRADER_ACCOUNT_ID=''
export CTRADER_TOKEN=''

earthly --secret CTRADER_CLIENT_ID --secret CTRADER_SECRET --secret CTRADER_ACCOUNT_ID \
--secret CTRADER_TOKEN +go-tests --INTEGRATION_TEST=true
```
