## Description
This is a commandline utility to send and receive messages to Azure servicebus queues.
Supports connection only via connection string.

## Download
Download a Linux/Windows binary here: https://github.com/jakubneubauer/azure-sb-cli/releases

## Usage
```
./azure-sb-cli -h
Usage: ./azure-sb-cli <command> <options>

Commands:
  send    - Sends messages to a queue. Reads standard input, sending each line as message, all in same session.
  receive - Receives messages, outputting them to standard output, message per line.
  peek    - Peeks one or more messages, outpus them to standard output.
  -v      - Prints version info.

Common options:
  -c   Connection string
  -d   (flag) Log debug info
  -h   (flag) Show this help
  -n   number of received or peeked messages. Defaults to one.
  -q   Queue name
  -s   Session ID.
       If the queue is not session-enabled, do not set this option.
       If the queue is session-enabled, must be specified for receive. The 'send' works without it.
       If set to empty string for receive, will receive message from any session.

Receive options:
  -p   Prefix every message with session id, separated with ':'
```
