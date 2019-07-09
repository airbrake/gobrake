package gobrake

import (
	"log"
	"os"

	"github.com/jonboulle/clockwork"
)

var logger = log.New(os.Stderr, "gobrake: ", log.LstdFlags|log.Lshortfile)
var clock = clockwork.NewRealClock()

func SetLogger(l *log.Logger) {
	logger = l
}
