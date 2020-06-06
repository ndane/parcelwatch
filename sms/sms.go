package sms

import (
	"github.com/kevinburke/twilio-go"
)

type Sender interface {
	Send(to, message string) error
}

type sender struct {
	tw     *twilio.Client
	sid    string
	token  string
	number string
}

func NewTwilioSender(token, sid, number string) Sender {
	return &sender{
		tw:     twilio.NewClient(sid, token, nil),
		sid:    sid,
		token:  token,
		number: number,
	}
}

func (s sender) Send(to, message string) error {
	_, err := s.tw.Messages.SendMessage(s.number, to, message, nil)
	return err
}
