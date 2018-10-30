package logging

import (
	"github.com/op/go-logging"
	"os"
	"strings"
	"errors"
	"fmt"
)

var (
	Log *logging.Logger

	format = logging.MustStringFormatter(
		`%{time:15:04:05.000} %{shortfunc} %{level:.4s} %{message}`,
	)
)

func InitLogger(module, logLevel string) error {
	Log = logging.MustGetLogger(module)

	backend := logging.NewLogBackend(os.Stdout, "", 0)
	formatter := logging.NewBackendFormatter(backend, format)

	leveled := logging.AddModuleLevel(formatter)

	switch strings.ToLower(logLevel) {
	case "critical":
		leveled.SetLevel(logging.CRITICAL, module)
	case "error":
		leveled.SetLevel(logging.ERROR, module)
	case "warning":
		leveled.SetLevel(logging.WARNING, module)
	case "notice":
		leveled.SetLevel(logging.NOTICE, module)
	case "info":
		leveled.SetLevel(logging.INFO, module)
	case "debug":
		leveled.SetLevel(logging.DEBUG, module)
	default:
		return errors.New(fmt.Sprintf("unexpected log leve: %s", logLevel))
	}

	Log.SetBackend(leveled)
	return nil
}