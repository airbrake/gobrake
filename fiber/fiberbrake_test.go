package fiber

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/airbrake/gobrake/v5"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
)

var notifier = gobrake.NewNotifierWithOptions(&gobrake.NotifierOptions{
	ProjectId:   999999,
	ProjectKey:  "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
	Environment: "production",
})

// go test -run Test_Fiberbrake
func Test_Fiberbrake(t *testing.T) {
	app := fiber.New()
	app.Use(New(notifier))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Status(http.StatusOK).SendString("Hello")
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
	utils.AssertEqual(t, nil, err)
	utils.AssertEqual(t, fiber.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	utils.AssertEqual(t, "Hello", string(body))
}

// go test -run Test_Fiberbrake_Next
func Test_Fiberbrake_Next(t *testing.T) {
	app := fiber.New()
	app.Use(New(nil))
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Status(http.StatusOK).SendString("Hello")
	})
	resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
	utils.AssertEqual(t, nil, err)
	utils.AssertEqual(t, fiber.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	utils.AssertEqual(t, "Hello", string(body))
}
