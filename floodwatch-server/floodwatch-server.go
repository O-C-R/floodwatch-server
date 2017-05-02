package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"

	"github.com/O-C-R/auth/session"

	"github.com/aws/aws-sdk-go/aws"
	awsSession "github.com/aws/aws-sdk-go/aws/session"

	"github.com/O-C-R/floodwatch/floodwatch-server/backend"
	"github.com/O-C-R/floodwatch/floodwatch-server/webserver"
)

type Config struct {
	Addr       string `default:"127.0.0.1:8080"`
	StaticPath string `split_words:"true"`
	BackendURL string `default:"postgres://localhost/floodwatch?sslmode=disable" split_words:"true"`

	SessionStoreAddr     string `default:"localhost:6379" split_words:"true"`
	SessionStorePassword string `split_words:"true"`

	AWSProfile                  string `default:"floodwatch" envconfig:"AWS_PROFILE"`
	AWSRegion                   string `default:"us-east-1" envconfig:"AWS_REGION"`
	S3Bucket                    string `envconfig:"S3_BUCKET"`
	SQSClassifierInputQueueURL  string `envconfig:"SQS_CLASSIFIER_INPUT_QUEUE_URL"`
	SQSClassifierOutputQueueURL string `envconfig:"SQS_CLASSIFIER_OUTPUT_QUEUE_URL"`

	TwofishesHost string `split_words:"true"`
	RedirectAddr  string `default:"127.0.0.1:8081" split_words:"true"`
	Insecure      bool   `default:"false"`
}

var help bool

func init() {
	flag.BoolVar(&help, "h", false, "print usage")
}

func main() {
	var config Config
	if help {
		envconfig.Usage("fw", &config)
		os.Exit(0)
	}

	if err := envconfig.Process("fw", &config); err != nil {
		panic(err)
	}

	b, err := backend.New(config.BackendURL)
	if err != nil {
		log.Fatal(err)
	}

	sessionStore, err := session.NewSessionStore(session.SessionStoreOptions{
		Addr:            config.SessionStoreAddr,
		Password:        config.SessionStorePassword,
		SessionDuration: time.Hour * 24 * 365,
		MaxSessions:     100,
	})
	if err != nil {
		log.Fatal(err)
	}

	awsSession, err := awsSession.NewSessionWithOptions(awsSession.Options{
		Profile: config.AWSProfile,
		Config: aws.Config{
			Region: aws.String(config.AWSRegion),
			CredentialsChainVerboseErrors: aws.Bool(true),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	options := &webserver.Options{
		Addr:                        config.Addr,
		RedirectAddr:                config.RedirectAddr,
		Backend:                     b,
		SessionStore:                sessionStore,
		AWSSession:                  awsSession,
		S3Bucket:                    config.S3Bucket,
		SQSClassifierInputQueueURL:  config.SQSClassifierInputQueueURL,
		SQSClassifierOutputQueueURL: config.SQSClassifierOutputQueueURL,
		Insecure:                    config.Insecure,
		StaticPath:                  config.StaticPath,
		TwofishesHost:               config.TwofishesHost,
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
