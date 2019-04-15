package gobrake

import (
	"log"
	"os"

	"github.com/jonboulle/clockwork"
)

var logger *log.Logger
var clock = clockwork.NewRealClock()

func init() {
	SetLogger(log.New(os.Stderr, "gobrake: ", log.LstdFlags))
}

func SetLogger(l *log.Logger) {
	logger = l
}
