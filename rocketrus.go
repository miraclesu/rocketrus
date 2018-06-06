// Package rocketrus provides a RocketChat hook for the logrus loggin package.
package rocketrus

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/RocketChat/Rocket.Chat.Go.SDK/models"
	"github.com/RocketChat/Rocket.Chat.Go.SDK/rest"
	"github.com/sirupsen/logrus"
)

var (
	NotRunningErr = fmt.Errorf("RocketrusHook doesn't running, please call Run function first")
)

// RocketrusHook is a logrus Hook for dispatching messages to the specified
// channel on RocketChat.
type RocketrusHook struct {
	HookURL string
	Channel string
	// If UserID and Token are present, will use UserID and Token auth rocket.chat API
	// otherwise Email and the Password are mandatory.
	UserID   string
	Token    string
	Email    string
	Password string

	// Messages with a log level not contained in this array
	// will not be dispatched. If nil, all messages will be dispatched.
	AcceptedLevels []logrus.Level
	Disabled       bool
	// Title name for log
	Title  string
	Alias  string
	Emoji  string
	Avatar string
	// Notify users with @user in RocketChat.
	NotifyUsers []string
	// batch send message duration, uion/second, default is 10 seconds
	// if duration is negative, RocketrusHook will block ticker message send
	Duration int64
	// batch send message, default is 8
	Batch int

	running bool
	msg     *models.PostMessage
	msgChan chan *models.Attachment

	*models.UserCredentials
	*rest.Client
}

// LevelThreshold - Returns every logging level above and including the given parameter.
func LevelThreshold(l logrus.Level) []logrus.Level {
	return logrus.AllLevels[:l+1]
}

// Run start RocketrusHook message processor
func (rh *RocketrusHook) Run() error {
	index := strings.Index(rh.HookURL, "://")
	serverUrl := &url.URL{
		Scheme: "http",
	}
	if index > 0 {
		serverUrl.Host = rh.HookURL[index+len("://"):]
		if strings.HasPrefix(rh.HookURL, "https") {
			serverUrl.Scheme = "https"
		}
	} else {
		serverUrl.Host = rh.HookURL
	}

	rh.Client = rest.NewClient(serverUrl, false)
	rh.UserCredentials = &models.UserCredentials{
		ID:       rh.UserID,
		Token:    rh.Token,
		Email:    rh.Email,
		Password: rh.Password,
	}
	if err := rh.Client.Login(rh.UserCredentials); err != nil {
		return err
	}
	rh.msgChan = make(chan *models.Attachment, 16)
	if rh.Duration == 0 {
		rh.Duration = 10
	}
	if rh.Batch <= 0 {
		rh.Batch = 8
	}

	var atUsers string
	if len(rh.NotifyUsers) > 0 {
		atUsers = strings.Join(rh.NotifyUsers, " @")
		atUsers = "@" + atUsers
	}

	rh.msg = &models.PostMessage{
		Channel: rh.Channel,
		Alias:   rh.Alias,
		Emoji:   rh.Emoji,
		Avatar:  rh.Avatar,
		Text:    fmt.Sprintf("%s\n*%s logs*", atUsers, rh.Title),
	}

	go rh.send()
	rh.running = true
	return nil
}

func (rh *RocketrusHook) send() {
	var (
		timer    *time.Timer
		duration time.Duration
	)
	if rh.Duration < 0 {
		timer = time.NewTimer(0)
		timer.C = nil
	} else {
		duration = time.Duration(rh.Duration) * time.Second
		timer = time.NewTimer(duration)
	}

	for {
		select {
		case msg := <-rh.msgChan:
			rh.msg.Attachments = append(rh.msg.Attachments, *msg)
			if len(rh.msg.Attachments) >= rh.Batch {
				rh.postMessage()
				timer.Reset(duration)
			}
		case <-timer.C:
			if len(rh.msg.Attachments) == 0 {
				timer.Reset(duration)
				continue
			}

			rh.postMessage()
			timer.Reset(duration)
		}
	}
}

func (rh *RocketrusHook) postMessage() {
	rh.Client.PostMessage(rh.msg)
	rh.msg.Attachments = rh.msg.Attachments[:0]
	if cap(rh.msg.Attachments) > 1024 {
		rh.msg.Attachments = make([]models.Attachment, 0, 16)
	}
}

// Levels sets which levels to sent to RocketChat
func (rh *RocketrusHook) Levels() []logrus.Level {
	if len(rh.AcceptedLevels) == 0 {
		return logrus.AllLevels
	}
	return rh.AcceptedLevels
}

// Fire -  Sent event to RocketChat
func (rh *RocketrusHook) Fire(e *logrus.Entry) error {
	if rh.Disabled {
		return nil
	}
	if !rh.running {
		return NotRunningErr
	}

	color := ""
	switch e.Level {
	case logrus.DebugLevel:
		color = "purple"
	case logrus.InfoLevel:
		color = "green"
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		color = "red"
	default:
		color = "yellow"
	}

	msg := &models.Attachment{
		Color: color,
		Title: e.Level.String() + " log",
		Ts:    e.Time.String(),
		Text:  e.Message,
	}

	if len(e.Data) > 0 {
		msg.Fields = make([]models.AttachmentField, len(e.Data))
		i := 0
		for k, v := range e.Data {
			msg.Fields[i] = models.AttachmentField{
				Title: k,
				Value: fmt.Sprint(v),
			}
			// If the field is <= 20 then we'll set it to short
			if len(msg.Fields[i].Value) <= 20 {
				msg.Fields[i].Short = true
			}
			i++
		}
	}

	rh.msgChan <- msg
	return nil
}
