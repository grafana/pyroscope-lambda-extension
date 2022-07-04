package relay_test

import (
	"io/ioutil"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func noopLogger() *logrus.Entry {
	logger := logrus.New()
	logger.SetOutput(ioutil.Discard)

	return logger.WithFields(logrus.Fields{})
}

func readTestdataFile(t *testing.T, name string) []byte {
	f, err := ioutil.ReadFile(name)
	assert.NoError(t, err)
	return f
}
