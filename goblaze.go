package goblaze

import (
	"github.com/erikdubbelboer/fasthttp"
	"github.com/sirupsen/logrus"
)

type GoBlaze struct {
	server *fasthttp.Server
	// router // our router
	log *logrus.Logger
}
