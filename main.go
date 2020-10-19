package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/Azure/azure-service-bus-go"
	"github.com/Azure/go-amqp"
	"log"
	"os"
)

var buildVersion = "unknown"
var buildDate = "unknown"

const NullStr = "\xff"

var logDebug = false
var prefixMsgWithSessionId = false
var correlationId = ""

type MySessionHandler struct {
	messageSession *servicebus.MessageSession
	received       bool
}

// Start is called when a new session is started
func (sh *MySessionHandler) Start(ms *servicebus.MessageSession) error {
	sh.messageSession = ms
	debug("Begin message session:", strPtroToString(ms.SessionID()))
	return nil
}

// Handle is called when a new session message is received
func (sh *MySessionHandler) Handle(ctx context.Context, msg *servicebus.Message) error {
	if msg.SessionID != nil && prefixMsgWithSessionId {
		fmt.Print(*msg.SessionID + ":")
	}
	fmt.Println(string(msg.Data))

	// If we wouldn't close the message session here, the call of queueSession.receiveOne would be blocked
	// and we would receive another message during it. It would be different way of using the servicebus API.
	if sh.messageSession != nil {
		debug("Closing message session: " + strPtroToString(sh.messageSession.SessionID()))
		sh.messageSession.Close()
	}
	sh.received = true
	return msg.Complete(ctx)
}

// End is called when the message session is closed. Service Bus will not automatically end your message session. Be
// sure to know when to terminate your own session.
func (sh *MySessionHandler) End() {
	debug("End message session:", strPtroToString(sh.messageSession.SessionID()))
}

func send(ctx context.Context, q *servicebus.Queue, sessionId *string) {
	if *sessionId == NullStr {
		sendNoSession(ctx, q)
	} else {
		if *sessionId == "" {
			sessionId = nil
		}
		sendSession(ctx, q, sessionId)
	}
}

func sendNoSession(ctx context.Context, q *servicebus.Queue) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		err := q.Send(ctx, createMsgFromString(scanner.Text()))
		if err != nil {
			fatal("Cannot send message:", err)
		}
	}
	if err := scanner.Err(); err != nil {
		fatal("Cannot read standard input:", err)
	}
}

func sendSession(ctx context.Context, q *servicebus.Queue, sessionId *string) {
	debug("Opening session", strPtroToString(sessionId))
	session := q.NewSession(sessionId)
	defer func() {
		debug("Closing session:", strPtroToString(sessionId))
		if err := session.Close(ctx); err != nil {
			debug("Cannot close session:", err)
		}
	}()
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		err := session.Send(ctx, createMsgFromString(scanner.Text()))
		if err != nil {
			fatal("Cannot send message:", err)
		}
	}
	if err := scanner.Err(); err != nil {
		fatal("Cannot read standard input:", err)
	}
}

func receive(ctx context.Context, q *servicebus.Queue, sessionId *string, count int) {
	if sessionId != nil && *sessionId != NullStr && *sessionId != "" {
		receiveMoreSession(ctx, q, sessionId, count)
	} else {
		for i := 0; (count < 0 || i < count) && receiveOne(ctx, q, sessionId); i++ {
		}
	}
}

func receiveOne(ctx context.Context, q *servicebus.Queue, sessionId *string) bool {
	if *sessionId == NullStr {
		return receiveOneNoSession(ctx, q)
	} else {
		if *sessionId == "" {
			sessionId = nil
		}
		return receiveOneSession(ctx, q, sessionId)
	}
}

func receiveOneNoSession(ctx context.Context, q *servicebus.Queue) bool {
	sh := new(MySessionHandler)
	err := q.ReceiveOne(ctx, sh)
	if err != nil {
		if amqpErr, ok := err.(*amqp.Error); ok {
			if amqpErr.Condition == "com.microsoft:timeout" {
				debug("Timeout receiving message", err)
				return true
			}
		}
		fatal("Cannot receive:", err)
	}
	return sh.received
}

func receiveOneSession(ctx context.Context, q *servicebus.Queue, sessionId *string) bool {
	debug("Opening queue session", strPtroToString(sessionId))
	queueSession := q.NewSession(sessionId)
	defer func() {
		debug("Closing queue session", strPtroToString(sessionId))
		if err := queueSession.Close(ctx); err != nil {
			debug("Cannot close queue session:", err)
		}
	}()
	sh := new(MySessionHandler)
	err := queueSession.ReceiveOne(ctx, sh)
	if err != nil {
		if amqpErr, ok := err.(*amqp.Error); ok {
			if amqpErr.Condition == "com.microsoft:timeout" {
				debug("Timeout receiving message", err)
				return true
			}
		}
		fatal("Cannot receive:", err)
	}
	return sh.received
}

func receiveMoreSession(ctx context.Context, q *servicebus.Queue, sessionId *string, count int) bool {
	debug("Opening queue session", strPtroToString(sessionId))
	queueSession := q.NewSession(sessionId)
	defer func() {
		debug("Closing queue session", strPtroToString(sessionId))
		if err := queueSession.Close(ctx); err != nil {
			debug("Cannot close queue session:", err)
		}
	}()
	sh := new(MySessionHandler)
	for i := 0; count < 0 || i < count; i++ {
		err := queueSession.ReceiveOne(ctx, sh)
		if err != nil {
			if amqpErr, ok := err.(*amqp.Error); ok {
				if amqpErr.Condition == "com.microsoft:timeout" {
					debug("Timeout receiving message", err)
					return true
				}
			}
			fatal("Cannot receive:", err)
		}
	}
	return sh.received
}

func peek(ctx context.Context, q *servicebus.Queue, count int) {
	subject, err := q.Peek(ctx)
	if err != nil {
		fatal("Cannot peek:", err)
	}
	for i := 0; count < 0 || i < count; i++ {
		cursor, err := subject.Next(ctx)
		if err != nil {
			if _, ok := err.(servicebus.ErrNoMessages); ok {
				return
			}
			fatal("Cannot iterate cursor:", err)
		}
		fmt.Println(string(cursor.Data))
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
       If the queue is session-enabled, must be specified for receive. The 'send' works without it.
       If set to empty string for receive, will receive message from any session.

Receive options:
  -p   Prefix every message with session id, separated with ':'
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
		commonFlags.BoolVar(&prefixMsgWithSessionId, "p", false, "Prefix received messages with session id, separated with ':'")
	case "send":
		commonFlags.StringVar(&correlationId, "i", "", "Correlation ID")
	}

	commonFlags.Parse(os.Args[2:])

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

	ns, err := servicebus.NewNamespace(servicebus.NamespaceWithConnectionString(*connStrPtr))
	if err != nil {
		fatal("Cannot connect to servicebus:", err)
	}

	// Create a client to communicate with the queue.
	q, err := ns.NewQueue(*queueNamePtr)
	if err != nil {
		fatal("Cannot connect to queue:", err)
	}

	switch os.Args[1] {
	case "send":
		send(ctx, q, sessionIdPtr)
	case "receive":
		receive(ctx, q, sessionIdPtr, *msgCountPtr)
	case "peek":
		peek(ctx, q, *msgCountPtr)
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

func createMsgFromString(s string) *servicebus.Message {
	msg := servicebus.NewMessageFromString(s)
	if correlationId != "" {
		msg.CorrelationID = correlationId
	}
	return msg
}
