package core

import (
	"go.uber.org/zap"
)

type logger struct {
	*zap.SugaredLogger
}

// TODO optimize logger instance
func newLogger(debug bool) (*logger, error) {
	var (
		log *zap.Logger
		err error
	)
	if debug {
		log, err = zap.NewDevelopment()
	} else {
		log, err = zap.NewProduction()
	}
	if err != nil {
		return nil, err
	}

	return &logger{
		SugaredLogger: log.Sugar(),
	}, nil
}
