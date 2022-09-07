## Description
Simple commandline utility to send and receive messages to/from
[Azure servicebus](https://docs.microsoft.com/en-us/azure/service-bus-messaging/service-bus-messaging-overview)
queues.

Limitations:
 - Only queues are supported.
 - Authentication is supported only with connection string.

## Download
Download a Linux/Windows binary here: https://github.com/jakubneubauer/azure-sb-cli/releases


## Example

### Using a queue without sessions support
```shell

# Connection string and session-less queue name - variables used in the
# rest of the example
$ CONN_STR="*** YOUR CONNECTION STRING ***"
$ QUEUE_NAME = "your queue name"

# Send a message
$ echo "Hello world!" | ./azure-sb-cli send -c "$CONN_STR" -q "$QUEUE_NAME"

# Receive message
$ azure-sb-cli receive -c "$CONN_STR" -q $QUEUE_NAME
Hello world!
```

### Communicating with queue using sessions
```shell

# Connection string and session-enabled queue name - variables used in the
# rest of the example
$ CONN_STR="*** YOUR CONNECTION STRING ***"
$ QUEUE_NAME = "your queue name"

# Send a message
$ echo "Hello world!" | ./azure-sb-cli send -c "$CONN_STR" -q "$QUEUE_NAME" -s "session-12345"

# Receive message:
# - using empty string as a session ID will receive message from any session. 
# - The message is prefixed with session ID in the output
$ azure-sb-cli receive -c "$CONN_STR" -q $QUEUE_NAME -p -s ""
session-12345:Hello world!
```

## Documentation
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
Send option:
  -i   Correlation ID for sent messages
```

# Further development
Ideas to implement:
- Support Correlation ID for `receive` operation.
- Support for servicebus topics.
- Batch send/receive for better performance.
