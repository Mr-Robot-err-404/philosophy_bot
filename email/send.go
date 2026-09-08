package email

import (
	"gopkg.in/gomail.v2"
)

type Payload struct {
	To       string
	Msg      string
	Subject  string
	Username string
	Pwd      string
}

type Log struct {
	Msg string
	Err error
}

func Send(payload Payload) error {
	m := gomail.NewMessage()
	m.SetHeader("From", payload.Username)
	m.SetHeader("To", payload.To)
	m.SetHeader("Subject", payload.Subject)
	m.SetBody("text/html", payload.Msg)

	d := gomail.NewDialer("smtp.gmail.com", 587, payload.Username, payload.Pwd)
	return d.DialAndSend(m)
}
