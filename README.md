# Testground test plans for libp2p


Run ping plan

```console
foo@bar:~$ testground run single --plan=test-plans/ping --testcase=ping --runner=local:docker --builder=docker:go --instances=2
```
