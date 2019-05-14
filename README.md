rocketrus
========

[RocketChat](https://rocket.chat/) hook for [Logrus](https://github.com/sirupsen/logrus). 

## Use

```go
package main

import (
	"log"
	"time"

	"github.com/miraclesu/rocketrus"
	"github.com/sirupsen/logrus"
)

func main() {
	hook := &rocketrus.RocketrusHook{
		HookURL: "http://localhost:3000",
		Channel: "general",

		NotifyUsers: []string{"miracle", "yuhcwl"},
		AcceptedLevels: rocketrus.LevelThreshold(logrus.DebugLevel),

		Email:    "suchuangji@gmail.com",
		Password: "gopher",

		Duration: -1,
		Batch:    1,
	}
	if err := hook.Run(); err != nil {
		log.Fatalln(err.Error())
	}

	logrus.AddHook(hook)
	logrus.SetLevel(logrus.DebugLevel)

	logrus.Warn("warn")
	logrus.Info("info")
	logrus.Debug("debug")
	time.Sleep(1 * time.Second)
}
```

## Parameters

#### Required
  * HookURL
  * Channel
  * UserID & Token or Email & Password

#### Optional
  * AcceptedLevels
  * Disabled
  * Title
  * Alias
  * Emoji
  * Avatar
  * NotifyUsers
  * Duration
  * Batch
## Installation

    go get github.com/miraclesu/rocketrus

## Credits

[slackrus](https://github.com/johntdyer/slackrus)