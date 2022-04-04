package gin

import (
	"context"
	"fmt"

	"github.com/airbrake/gobrake/v5"

	"github.com/charmbracelet/lipgloss"
	"github.com/gin-gonic/gin"
)

// New returns a function that satisfies gin.HandlerFunc interface
func New(notifier *gobrake.Notifier) gin.HandlerFunc {
	style := lipgloss.NewStyle().
		Width(80).
		Bold(true).
		Padding(1).
		Foreground(lipgloss.Color("#ab1ef3")).
		Background(lipgloss.Color("#850AC2")).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#ab1ef3")).
		BorderBackground(lipgloss.Color("850AC2"))

	fmt.Println(style.Render(
		fmt.Sprintf(
			"Hello Local Gobrake Dev! Here's your current notifier config:\n\n%s",
			notifier.Options(),
		),
	))

	return func(c *gin.Context) {
		_, metric := gobrake.NewRouteMetric(context.TODO(), c.Request.Method, c.FullPath())

		c.Next()

		metric.StatusCode = c.Writer.Status()
		_ = notifier.Routes.Notify(context.TODO(), metric)
	}
}

// This function is deprecated. Please use New() function instead
func NewMiddleware(engine *gin.Engine, notifier *gobrake.Notifier) func(c *gin.Context) {
	return New(notifier)
}
