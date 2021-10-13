package tg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"oikotie/config"

	"github.com/pkg/errors"
)

func SendMessage(cfg *config.Reader, msg string) error {
	return send(cfg.TgBotToken(), cfg.TgChatID(), msg)
}

type sendRequest struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

func send(token string, chatId string, msg string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	req := sendRequest{
		ChatID: chatId,
		Text:   msg,
	}

	r, err := json.Marshal(req)
	if err != nil {
		return errors.WithStack(err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(r))
	if err != nil {
		return errors.WithStack(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(string(body))
	}

	return nil
}
