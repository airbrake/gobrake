package controllers

import (
	"fmt"

	"github.com/astaxie/beego"
)

type HelloController struct {
	beego.Controller
}

func (pc *HelloController) Get() {
	name := pc.Ctx.Input.Param(":name")
	pc.Data["json"] = fmt.Sprintf("Hello %s", name)
	pc.ServeJSON()
}
