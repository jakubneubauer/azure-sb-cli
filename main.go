package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus"
	"log"
	"os"
	"strings"
)

var buildVersion = "unknown"
var buildDate = "unknown"

const NullStr = "\xff"

var logDebug = false
var prefixMsgWithSessionId = false
var prefixMsgWithMessageId = false
var correlationId = ""

func send(ctx context.Context, client *azservicebus.Client, sessionId *string, queueName *string) {
	if *sessionId == "" {
		sessionId = nil
	}
	debug("Opening sender")
	sender, err := client.NewSender(*queueName, nil)
	if err != nil {
		fatal("Cannot create sender:", err)
	}
	defer func() {
		debug("Closing sender")
		sender.Close(ctx)
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		msg := createMsgFromString(scanner.Text())
		if sessionId != nil && *sessionId != NullStr {
			msg.SessionID = sessionId
		}
		debug("Sending message")
		err := sender.SendMessage(ctx, msg, nil)
		if err != nil {
			fatal("Cannot send message:", err)
		}
	}
	if err := scanner.Err(); err != nil {
		fatal("Cannot read standard input:", err)
	}
}

func receive(ctx context.Context, client *azservicebus.Client, sessionId *string, count int, queueName *string) {
	if sessionId != nil && *sessionId != NullStr && *sessionId != "" {
		receiveMoreSession(ctx, client, sessionId, queueName, count)
	} else {
		for i := 0; (count < 0 || i < count) && receiveOne(ctx, client, sessionId, queueName); i++ {
		}
	}
}

func receiveOne(ctx context.Context, client *azservicebus.Client, sessionId *string, queueName *string) bool {
	if *sessionId == NullStr {
		return receiveOneNoSession(ctx, client, queueName)
	} else {
		if *sessionId == "" {
			sessionId = nil
		}
		return receiveMoreSession(ctx, client, sessionId, queueName, 1)
	}
}

func receiveOneNoSession(ctx context.Context, client *azservicebus.Client, queueName *string) bool {
	debug("Opening receiver")
	receiver, err := client.NewReceiverForQueue(
		*queueName,
		nil,
	)
	defer func() {
		debug("Closing receiver")
		receiver.Close(ctx)
	}()

	if err != nil {
		fatal("Cannot create receiver:", err)
	}

	debug("Calling receive", 1)
	messages, err := receiver.ReceiveMessages(ctx, 1, nil)
	if err != nil {
		fatal("Cannot receive message:", err)
	}

	debug("Received messages", len(messages))

	for _, msg := range messages {
		if msg.SessionID != nil && prefixMsgWithSessionId {
			fmt.Print(*msg.SessionID + ":")
		}
		if msg.SessionID != nil && prefixMsgWithMessageId {
			fmt.Print(msg.MessageID + ":")
		}
		fmt.Println(string(msg.Body))

		err = receiver.CompleteMessage(ctx, msg, nil)

		if err != nil {
			var sbErr *azservicebus.Error

			if errors.As(err, &sbErr) && sbErr.Code == azservicebus.CodeLockLost {
				// The message lock has expired. This isn't fatal for the client, but it does mean
				// that this message can be received by another Receiver (or potentially this one!).
				debug("Message lock expired\n")
				continue
			}

			fatal("Cannot receive message", err)
		}
		return true
	}
	return false
}

func receiveMoreSession(ctx context.Context, client *azservicebus.Client, sessionId *string, queueName *string, count int) bool {
	var receiver *azservicebus.SessionReceiver
	var err error

	for true {
		if sessionId != nil && *sessionId != "" {
			debug("Opening queue session", strPtroToString(sessionId))
			receiver, err = client.AcceptSessionForQueue(ctx, *queueName, *sessionId, nil)
		} else {
			debug("Opening next available session")
			receiver, err = client.AcceptNextSessionForQueue(ctx, *queueName, nil)
		}
		if err != nil {
			// After version 1.1.0 this should be possible (see https://github.com/Azure/azure-sdk-for-go/issues/19039)
			//var sbErr *azservicebus.Error
			//if errors.As(err, &sbErr) && sbErr.Code == azservicebus.CodeTimeout {
			//	// there are no sessions available. This isn't fatal - we can use the client and try
			// 	// to AcceptNextSessionForQueue() again.
			//	debug("No session available")
			//	continue
			//}
			if strings.Contains(err.Error(), "com.microsoft:timeout") {
				// there are no sessions available. This isn't fatal - we can use the client and try
				// to AcceptNextSessionForQueue() again.
				debug("No session available, trying again")
				continue
			}
			fatal("Cannot open session:", err)
		} else {
			break
		}
	}
	defer func() {
		debug("Closing queue session", strPtroToString(sessionId))
		receiver.Close(ctx)
	}()

	for i := 0; count < 0 || i < count; {
		remains := 100
		if count > 0 {
			remains = count - i
		}
		debug("Calling receive", remains)
		messages, err := receiver.ReceiveMessages(ctx, remains, nil)
		fatal("Cannot receive messages:", err)

		for _, msg := range messages {
			if msg.SessionID != nil && prefixMsgWithSessionId {
				fmt.Print(*msg.SessionID + ":")
			}
			if msg.SessionID != nil && prefixMsgWithMessageId {
				fmt.Print(msg.MessageID + ":")
			}
			fmt.Println(string(msg.Body))

			err = receiver.CompleteMessage(ctx, msg, nil)

			if err != nil {
				var sbErr *azservicebus.Error

				if errors.As(err, &sbErr) && sbErr.Code == azservicebus.CodeLockLost {
					// The message lock has expired. This isn't fatal for the client, but it does mean
					// that this message can be received by another Receiver (or potentially this one!).
					debug("Message lock expired\n")
					continue
				}

				fatal("Cannot receive message", err)
			}
			i++
		}
	}
	return true
}

func peek(ctx context.Context, client *azservicebus.Client, queueName *string, count int) {
	receiver, err := client.NewReceiverForQueue(
		*queueName,
		nil,
	)
	if err != nil {
		fatal("Cannot create receiver:", err)
	}
	defer receiver.Close(ctx)

	for i := 0; count < 0 || i < count; {
		messages, err := receiver.PeekMessages(ctx, 1, nil)
		if err != nil {
			fatal("Cannot peek message:", err)
		}

		for _, msg := range messages {
			if msg.SessionID != nil && prefixMsgWithSessionId {
				fmt.Print(*msg.SessionID + ":")
			}
			if msg.SessionID != nil && prefixMsgWithMessageId {
				fmt.Print(msg.MessageID + ":")
			}
			fmt.Println(string(msg.Body))
			i++
		}
	}
}

func usage() {
	fmt.Println(`Usage: ` + os.Args[0] + ` <command> <options>

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
       If the queue is session-enabled, must be specified.
       If set to empty string for receive, will receive message from any session.

Receive options:
  -ps   Prefix every message with session id, separated with ':'. Useful if receiving all sessions messages.
  -pm   Prefix every message with message id, separated with ':'.
Send option:
  -i   Correlation ID for sent messages
`)
}

func printVersion() {
	fmt.Println(os.Args[0] + " " + buildVersion + " (built " + buildDate + ")")
}

func main() {
	commonFlags := flag.NewFlagSet("common flags", flag.ExitOnError)
	connStrPtr := commonFlags.String("c", "", "Connection string")
	queueNamePtr := commonFlags.String("q", "", "Queue name")
	sessionIdPtr := commonFlags.String("s", NullStr, "Session ID")
	msgCountPtr := commonFlags.Int("n", 1, "Number of received/peeked messages")
	helpPtr := commonFlags.Bool("h", false, "Show help")
	commonFlags.BoolVar(&logDebug, "d", false, "Log debug info")

	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "-h":
		usage()
		return
	case "-v":
		printVersion()
		return
	case "receive":
		commonFlags.BoolVar(&prefixMsgWithSessionId, "ps", false, "Prefix received messages with session id, separated with ':'")
		commonFlags.BoolVar(&prefixMsgWithMessageId, "pm", false, "Prefix received messages with message id, separated with ':'")
	case "send":
		commonFlags.StringVar(&correlationId, "i", "", "Correlation ID")
	}

	err := commonFlags.Parse(os.Args[2:])
	if err != nil {
		usage()
		os.Exit(2)
	}

	if *helpPtr {
		usage()
		os.Exit(2)
	}

	if *connStrPtr == "" {
		fatal("connection string is required")
	}
	if *queueNamePtr == "" {
		fatal("queue name is required")
	}

	//ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	//defer cancel()
	ctx := context.Background()

	client, err := azservicebus.NewClientFromConnectionString(*connStrPtr, nil)

	if err != nil {
		fatal("Cannot connect to servicebus:", err)
	}

	switch os.Args[1] {
	case "send":
		send(ctx, client, sessionIdPtr, queueNamePtr)
	case "receive":
		receive(ctx, client, sessionIdPtr, *msgCountPtr, queueNamePtr)
	case "peek":
		peek(ctx, client, queueNamePtr, *msgCountPtr)
	}
}

func strPtroToString(s *string) string {
	if s == nil {
		return "<nil>"
	} else {
		return *s
	}
}

func debug(msg string, args ...interface{}) {
	if logDebug {
		log.Println(append([]interface{}{"DEBUG: " + msg}, args...)...)
	}
}

func fatal(msg string, args ...interface{}) {
	log.Fatal(append([]interface{}{"ERROR: " + msg}, args...)...)
}

func createMsgFromString(s string) *azservicebus.Message {
	msg := &azservicebus.Message{
		Body: []byte(s),
	}
	if correlationId != "" {
		msg.CorrelationID = &correlationId
	}
	return msg
}
