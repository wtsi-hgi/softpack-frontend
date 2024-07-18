package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/wtsi-hgi/softpack-frontend/artefacts"
	"github.com/wtsi-hgi/softpack-frontend/environments"
	"github.com/wtsi-hgi/softpack-frontend/server"
	"github.com/wtsi-hgi/softpack-frontend/spack"
	"gopkg.in/yaml.v3"
)

func main() {
	if os.Getenv("DEV") != "" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	if err := run(); err != nil {
		slog.Error(err.Error())

		os.Exit(1)
	}
}

type Config struct {
	Spack struct {
		Version         string `yaml:"Version"`
		CustomRepo      string `yaml:"CustomRepo"`
		UpdateFrequency int    `yaml:"UpdateFrequency"`
	} `yaml:"Spack"`
	Artefacts struct {
		Repo     string `yaml:"Repo"`
		Username string `yaml:"Username"`
		Password string `yaml:"Password"`
	} `yaml:"Artefacts"`
	Server struct {
		IP   string `yaml:"IP"`
		Port string `yaml:"Port"`
		Path string `yaml:"Path"`
	} `yaml:"Server"`
}

func run() error {
	var configFile string

	flag.StringVar(&configFile, "c", "", "config file")
	flag.Parse()

	c, err := parseConfig(configFile)
	if err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	slog.Debug("loading spack repo", "version", c.Spack.Version)

	s, err := spack.New(plumbing.NewTagReferenceName(c.Spack.Version))
	if err != nil {
		return fmt.Errorf("error loading spack repo: %w", err)
	}

	if c.Spack.CustomRepo != "" {
		slog.Debug("watching spack custom repo", "repo", c.Spack.CustomRepo)

		if err = s.WatchRemote(c.Spack.CustomRepo, time.Duration(c.Spack.UpdateFrequency)*time.Second); err != nil {
			return fmt.Errorf("error watching custom repo: %w", err)
		}
	}

	slog.Debug("loading artefacts", "repo", c.Artefacts.Repo)

	a, err := artefacts.New(artefacts.Remote(c.Artefacts.Repo))
	if err != nil {
		return fmt.Errorf("error loading artefacts: %w", err)
	}

	slog.Debug("loading environments")

	e, err := environments.New(a)
	if err != nil {
		return fmt.Errorf("error loading environments: %w", err)
	}

	var h http.Handler

	if dev := os.Getenv("DEV"); dev == "" {
		slog.Debug("creating dev server", "path", dev)

		h = server.New(s, e)
	} else {
		h = server.NewDev(s, e, dev)
	}

	return startServer(c, h)
}

func startServer(c *Config, h http.Handler) error {
	if c.Server.Path != "" {
		h = http.StripPrefix(c.Server.Path, h)
	}

	if c.Server.Port == "" {
		c.Server.Port = "8080"
	}

	addr := fmt.Sprintf("%s:%s", c.Server.IP, c.Server.Port)

	slog.Info("running server on", "addr", addr)
	http.ListenAndServe(addr, h)

	return nil
}

func parseConfig(configFile string) (*Config, error) {
	var config Config

	f, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	if err = yaml.NewDecoder(f).Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
