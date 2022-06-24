package relay_test

import (
	"io/ioutil"

	"github.com/sirupsen/logrus"
)

func noopLogger() *logrus.Entry {
	logger := logrus.New()
	logger.SetOutput(ioutil.Discard)

	return logger.WithFields(logrus.Fields{})
}
