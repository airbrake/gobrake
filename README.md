# Airbrake Golang Notifier [![Build Status](https://circleci.com/gh/airbrake/gobrake.png?circle-token=4cbcbf1a58fa8275217247351a2db7250c1ef976)](https://circleci.com/gh/airbrake/gobrake)

<img src="http://f.cl.ly/items/3J3h1L05222X3o1w2l2L/golang.jpg" width=800px>

Example
---

```go
import "gopkg.in/airbrake/gobrake.v1"

airbrake = gobrake.NewNotifier(projectId, apiKey)
airbrake.SetContext("environment", "production")

if err := processRequest(req); err != nil {
   go airbrake.Notify(err, req)
}
```
