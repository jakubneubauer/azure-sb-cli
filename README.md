## Compile & install
- Install go ( https://golang.org )
- Get source code and build
```
cd $GOPATH
go get -u github.com/jakubneubauer/azure-sb-cli
cd github.com/jakubneubauer/azure-sb-cli
go build
```
- Try it:
```
./azure-sb-cli -h
Usage: ./azure-sb-cli <command> <options>

Commands:
  send - Sends messages to a queue. Reads standard input, sending each line as message, all in same session.
  receive - Receives messages, outputting them to standard output, message per line.

Common options:
  -c   Connection string
  -q   Queue name
  -s   Session ID. 
       If the queue is not session-enabled, do not set this option.
       If the queue is session-enabled, must be specified for receive. The 'send' works without it.
       If set to empty string for receive, will receive message from any session.
  -h   (flag) Show this help
  -d   (flag) Log debug info

Receive options:
  -p   Prefix every message with session id, separated with ':'
```
