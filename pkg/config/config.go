package config

import (
	"context"
	"errors"
	"os"

	"github.com/sirupsen/logrus"
)

type Config struct {
	ApplicationMetrics *ApplicationMetrics
	Local              bool
	Host               string
	Port               string
}

func New(ctx context.Context, log *logrus.Logger) (*Config, func(context.Context) error, error) {
	applicationMetrics, shutdownLogs, err := ConfigureOpenTelemetry(ctx, log)
	if err != nil {
		return nil, nil, err
	}

	local := os.Getenv("LOCAL") == "true"

	host, err := func() (string, error) {
		if local {
			return "http://localhost", nil
		}

		host := os.Getenv("HOST")

		if host == "" {
			return "", errors.New("HostNotSet")
		}

		return host, nil
	}()
	if err != nil {
		return nil, nil, err
	}

	port := func() string {
		port := os.Getenv("PORT")
		if port == "" {
			return "8080"
		}

		return port
	}()

	return &Config{
		ApplicationMetrics: applicationMetrics,
		Local:              local,
		Host:               host,
		Port:               port,
	}, shutdownLogs, nil
}
