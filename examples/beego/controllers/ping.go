package controllers

import (
	"time"

	"github.com/astaxie/beego"
)

type PingController struct {
	beego.Controller
}

func (pc *PingController) Get() {
	time.Sleep(time.Second)

	pc.Data["json"] = "Pong"
	pc.ServeJSON()
}
