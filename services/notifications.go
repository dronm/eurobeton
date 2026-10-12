package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dronm/gobizapp/notif"

	"github.com/dronm/ds/v4"
)

// Здесь собраны функции отправки различных сообщений
// специфичные для проекта
var eurobetonNotifier *notif.Notifier

func InitNotifier(host, appName, pwd string) {
	eurobetonNotifier = notif.NewNotifier(host, appName, pwd)
}

func NotifSend(batch []*notif.NotifMessage) error {
	if eurobetonNotifier == nil {
		return fmt.Errorf("eurobetonNotifier is not initialized")
	}
	resp, err := eurobetonNotifier.Send(batch)
	if err != nil {
		return err
	}

	var e strings.Builder
	for _, r := range resp {
		if r.Error != "" {
			if e.Len() > 0 {
				e.WriteString(", ")
			}
			e.WriteString(fmt.Sprintf("Error sending message ID: %d, text: %s", r.ID, r.Error))
		}
	}
	if e.Len() > 0 {
		return errors.New(e.String())
	}
	return nil
}

func RecoverPasswordEmail(ctx context.Context, conn ds.Conn, userId int, url string, newPassword string) error {
	msg := &notif.EmailMessage{}
	err := conn.QueryRow(ctx,
		`SELECT from_addr, from_name, reply_name, sender_addr, to_addr, to_name, body, subject
		FROM email_recover_pwd($1, $2, $3)`,
		userId, url, newPassword,
	).Scan(&msg.FromAddr, &msg.FromName, &msg.ReplyName, &msg.SenderAddr,
		&msg.ToAddr, &msg.ToName, &msg.Body, &msg.Subject)
	if errors.Is(err, ds.ErrNoRows) {
		return errors.New(notif.ER_TEMPLATE_NOT_FOUND)
	}
	if err != nil {
		return fmt.Errorf("email_recover_pwd query failed: %w", err)
	}
	if msg.ToAddr == "" {
		return errors.New(notif.ER_EMAIL_NO_TO_ADDR)
	}

	notifMsg := notif.NewNotifMessage("confirm", notif.PROV_EMAIL)
	notifMsg.AddEmailMessage(msg)
	return NotifSend([]*notif.NotifMessage{notifMsg})
}

func CorrectTelForSMS(tel *string) {
	if len(*tel) == 10 {
		*tel = "7" + *tel
	}
}
