package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/O-C-R/auth/session"

	"github.com/aws/aws-sdk-go/aws"
	awsSession "github.com/aws/aws-sdk-go/aws/session"

	"github.com/O-C-R/floodwatch/floodwatch-server/backend"
	"github.com/O-C-R/floodwatch/floodwatch-server/webserver"
)

var (
	config struct {
		addr, staticPath, backendURL, sessionStoreAddr, sessionStorePassword, s3Bucket, sqsClassifierInputQueueURL, sqsClassifierOutputQueueURL, twofishesHost, redirectAddr string
		insecure                                                                                                                                                             bool
	}
)

func init() {
	flag.StringVar(&config.backendURL, "backend-url", "postgres://localhost/floodwatch?sslmode=disable", "postgres backend URL")
	flag.StringVar(&config.sessionStoreAddr, "session-store-address", "localhost:6379", "redis session store address")
	flag.StringVar(&config.sessionStorePassword, "session-store-password", "", "redis session store password")
	flag.StringVar(&config.addr, "a", "127.0.0.1:8080", "address to listen on")
	flag.StringVar(&config.s3Bucket, "bucket", "floodwatch-ads", "S3 bucket")
	flag.StringVar(&config.sqsClassifierInputQueueURL, "input-queue-url", "https://sqs.us-east-1.amazonaws.com/963245043784/classifier-input", "S3 bucket")
	flag.StringVar(&config.sqsClassifierOutputQueueURL, "output-queue-url", "https://sqs.us-east-1.amazonaws.com/963245043784/classifier-output", "S3 bucket")
	flag.StringVar(&config.staticPath, "static", "", "static path")
	flag.StringVar(&config.twofishesHost, "twofishes-host", "http://twofishes.floodwatch.me", "host for twofishes server")
	flag.StringVar(&config.redirectAddr, "redirect-addr", "127.0.0.1:8081", "address to redirect to https")
	flag.BoolVar(&config.insecure, "i", false, "insecure (no user authentication)")
}

func main() {
	flag.Parse()

	if backendURL := os.Getenv("BACKEND_URL"); backendURL != "" {
		config.backendURL = backendURL
	}

	if sessionStoreAddr := os.Getenv("SESSION_STORE_ADDRESS"); sessionStoreAddr != "" {
		config.sessionStoreAddr = sessionStoreAddr
	}

	if sessionStorePassword := os.Getenv("SESSION_STORE_PASSWORD"); sessionStorePassword != "" {
		config.sessionStorePassword = sessionStorePassword
	}

	if bucket := os.Getenv("BUCKET"); bucket != "" {
		config.s3Bucket = bucket
	}

	if inputQueueURL := os.Getenv("INPUT_QUEUE_URL"); inputQueueURL != "" {
		config.sqsClassifierInputQueueURL = inputQueueURL
	}

	if outputQueueURL := os.Getenv("OUTPUT_QUEUE_URL"); outputQueueURL != "" {
		config.sqsClassifierOutputQueueURL = outputQueueURL
	}

	b, err := backend.New(config.backendURL)
	if err != nil {
		log.Fatal(err)
	}

	sessionStore, err := session.NewSessionStore(session.SessionStoreOptions{
		Addr:            config.sessionStoreAddr,
		Password:        config.sessionStorePassword,
		SessionDuration: time.Hour * 24 * 365,
		MaxSessions:     100,
	})
	if err != nil {
		log.Fatal(err)
	}

	awsSession, err := awsSession.NewSessionWithOptions(awsSession.Options{
		Profile: "floodwatch",
		Config: aws.Config{
			Region: aws.String("us-east-1"),
			CredentialsChainVerboseErrors: aws.Bool(true),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	options := &webserver.Options{
		Addr:                        config.addr,
		RedirectAddr:                config.redirectAddr,
		Backend:                     b,
		SessionStore:                sessionStore,
		AWSSession:                  awsSession,
		S3Bucket:                    config.s3Bucket,
		SQSClassifierInputQueueURL:  config.sqsClassifierInputQueueURL,
		SQSClassifierOutputQueueURL: config.sqsClassifierOutputQueueURL,
		Insecure:                    config.insecure,
		StaticPath:                  config.staticPath,
		TwofishesHost:               config.twofishesHost,
	}

	server, err := webserver.New(options)
	if err != nil {
		log.Fatal(err)
	}

	redirectServer := webserver.NewRedirectServer(options)

	errs := make(chan error)
	go func() {
		errs <- server.ListenAndServe()
	}()
	go func() {
		errs <- redirectServer.ListenAndServe()
	}()

	log.Fatal(<-errs)
}
