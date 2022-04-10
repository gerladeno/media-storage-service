package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gerladeno/media-storage-service/internal/rest"
	"github.com/gerladeno/media-storage-service/internal/storage"
	"github.com/gerladeno/media-storage-service/pkg/common"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

const httpPort = 3000

//go:embed public.pub
var publicSigningKey []byte

var version = `0.0.0`

func main() {
	log := GetLogger(true)
	log.Infof("starting authorization service version %s", version)
	if err := godotenv.Load(); err != nil {
		if common.RunsInContainer() {
			log.Warn(err)
		} else {
			log.Panic(err)
		}
	}

	var (
		minioEndpoint  = os.Getenv("MINIO_ENDPOINT")
		minioAccessKey = os.Getenv("MINIO_ACCESS_KEY")
		minioSecretKey = os.Getenv("MINIO_SECRET_KEY")
		host           = "localhost"
	)
	ctx := context.Background()
	fileStorage, err := storage.New(log, minioEndpoint, minioAccessKey, minioSecretKey)
	if err != nil {
		log.Panic(err)
	}
	fileService, err := storage.NewService(log, fileStorage)
	if err != nil {
		log.Panic(err)
	}
	router := rest.NewRouter(log, fileService, mustGetPrivateKey(publicSigningKey), host, version)
	if err = startServer(ctx, router, log); err != nil {
		log.Panic(err)
	}
}

func startServer(ctx context.Context, router http.Handler, log *logrus.Logger) error {
	log.Infof("starting server on port %d", httpPort)
	s := &http.Server{
		Addr:              fmt.Sprintf(":%d", httpPort),
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		Handler:           router,
	}
	errCh := make(chan error)
	go func() {
		if err := s.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT)
	select {
	case err := <-errCh:
		return err
	case <-sigCh:
	}
	log.Info("terminating...")
	gfCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return s.Shutdown(gfCtx)
}

func GetLogger(verbose bool) *logrus.Logger {
	log := logrus.StandardLogger()
	log.SetFormatter(&logrus.JSONFormatter{})
	if verbose {
		log.SetLevel(logrus.DebugLevel)
		log.Debug("log level set to debug")
	}
	return log
}

func mustGetPrivateKey(keyBytes []byte) *rsa.PublicKey {
	if len(keyBytes) == 0 {
		panic("neither file private_rsa_key.pem nor env PUBLIC_SIGNING_KEY are set")
	}
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		panic("unable to decode private key to blocks")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		panic(err)
	}
	return key.(*rsa.PublicKey)
}
