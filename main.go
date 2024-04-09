package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Debug bool
	Host  string
	Port  int
}

type Server struct {
	Config Config
	engine *gin.Engine
}

var log = logrus.New()

func main() {
	var (
		host  string
		port  int
		debug bool
	)

	flag.StringVar(&host, "host", "", "host to connect")
	flag.IntVar(&port, "port", 0, "port to connect")
	flag.BoolVar(&debug, "debug", false, "enable debug")

	flag.Parse()

	if host == "" {
		log.Warnf("host is empty, set default %s", "127.0.0.1")
		host = "127.0.0.1"
	}
	if port == 0 {
		log.Warnf("port is empty, set default %d", 1515)
		port = 1515
	}

	var config Config
	config.Host = host
	config.Port = port
	config.Debug = debug

	ctx := context.Background()

	s := New(ctx, config)
	srv := s.Start()

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Error server shutdown:", err)
	}
}

func New(ctx context.Context, conf Config) *Server {
	s := &Server{
		Config: conf,
	}

	if conf.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()

	ips := engine.Group("/get")
	{
		ips.GET("/body", s.getBody(ctx))
	}

	s.engine = engine
	return s
}

func (s *Server) Start() *http.Server {
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.Config.Host, s.Config.Port),
		Handler:      s.engine,
		ReadTimeout:  300 * time.Second,
		IdleTimeout:  300 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	return srv
}

func (s *Server) getBody(ctx context.Context) func(c *gin.Context) {
	return func(c *gin.Context) {
		file, err := os.Open("body.txt")
		if err != nil {
			log.Errorf("error opening file %s", "body.txt")
			c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		}

		bodyFile, err := io.ReadAll(file)
		if err != nil {
			log.Errorf("error reading file %s", "body.txt")
			c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		}
		_ = file.Close()

		bodyText := string(bodyFile)
		bodyText = strings.ReplaceAll(bodyText, "\n", "")
		bodyText = strings.ReplaceAll(bodyText, "\r", "")

		var data json.RawMessage
		err = json.Unmarshal([]byte(bodyText), &data)
		if err != nil {
			log.Errorf("error unmarshal data %s", "body.txt")
			c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		}
		c.JSON(http.StatusOK, data)
	}
}
