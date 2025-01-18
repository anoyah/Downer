package core

import "go.uber.org/zap"

type logger struct {
	*zap.SugaredLogger
}

// TODO optimize logger instance
func newLogger() (*logger, error) {
	log, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}

	return &logger{
		SugaredLogger: log.Sugar(),
	}, nil
}
