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
	"github.com/wtsi-hgi/softpack-frontend/ldap"
	"github.com/wtsi-hgi/softpack-frontend/server"
	"github.com/wtsi-hgi/softpack-frontend/spack"
	"github.com/wtsi-hgi/softpack-frontend/users"
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
		Cache           string `yaml:"Cache"`
	} `yaml:"Spack"`
	Artefacts struct {
		Repo     string `yaml:"Repo"`
		Username string `yaml:"Username"`
		Password string `yaml:"Password"`
		Cache    string `yaml:"Cache"`
	} `yaml:"Artefacts"`
	Server struct {
		IP   string `yaml:"IP"`
		Port string `yaml:"Port"`
		Path string `yaml:"Path"`
	} `yaml:"Server"`
	LDAP struct {
		Server string `yaml:"Server"`
		Base   string `yaml:"Base"`
		Filter string `yaml:"Filter"`
		Attr   string `yaml:"Attr"`
	} `yaml:"LDAP"`
}

func run() error {
	var configFile string

	flag.StringVar(&configFile, "c", "", "config file")
	flag.Parse()

	c, err := parseConfig(configFile)
	if err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	var u http.Handler

	if c.LDAP.Server != "" {
		slog.Debug("connecting to ldap server", "url", c.LDAP.Server)

		l, err := ldap.New(c.LDAP.Server, c.LDAP.Base, c.LDAP.Filter, c.LDAP.Attr)
		if err != nil {
			return fmt.Errorf("error connecting to ldap server: %w", err)
		}

		u = l
	} else {
		u = users.New()
	}

	var (
		spackOptions []spack.Option
		spackDebug   = []any{"version", c.Spack.Version}
	)

	if c.Spack.CustomRepo != "" {
		spackOptions = append(spackOptions, spack.Remote(c.Spack.CustomRepo, time.Duration(c.Spack.UpdateFrequency)*time.Second))
		spackDebug = append(spackDebug, "remote", c.Spack.CustomRepo)
	}

	if c.Spack.Cache != "" {
		spackOptions = append(spackOptions, spack.CacheDir(c.Spack.Cache))
		spackDebug = append(spackDebug, "cache", c.Spack.Cache)
	}

	slog.Debug("loading spack repo", spackDebug...)

	s, err := spack.New(plumbing.NewTagReferenceName(c.Spack.Version), spackOptions...)
	if err != nil {
		return fmt.Errorf("error loading spack repo: %w", err)
	}

	artefactOptions := []artefacts.Option{artefacts.Remote(c.Artefacts.Repo)}
	artefactsDebug := []any{"repo", c.Artefacts.Repo}

	if c.Artefacts.Cache != "" {
		artefactOptions = append(artefactOptions, artefacts.FS(c.Artefacts.Cache))
		artefactsDebug = append(artefactsDebug, "cache", c.Artefacts.Cache)
	}

	slog.Debug("loading artefacts", artefactsDebug...)

	a, err := artefacts.New(artefactOptions...)
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

		h = server.New(s, e, u)
	} else {
		h = server.NewDev(s, e, u, dev)
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