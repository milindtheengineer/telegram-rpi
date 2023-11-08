package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/stianeikeland/go-rpio/v4"
)

const (
	botToken = ""
	apiURL   = "https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=%d&allowed_updates=%s"
)

func main() {

	err := rpio.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer rpio.Close()

	// Use GPIO pin 17 (BCM numbering)
	pin := rpio.Pin(17)

	// Set pin as an output
	pin.Output()

	offset := 0
	timeout := 60
	allowedUpdates := "message"

	for {
		updates, err := getUpdates(offset, timeout, allowedUpdates)
		if err != nil {
			log.Fatal(err)
		}

		for _, update := range updates {
			offset = update.ID + 1

			if update.Message != nil && update.Message.Text != "" {
				if err := processMessage(update.Message, pin); err != nil {
					log.Printf("Error :%v\n", err)
				}
			}
		}

		time.Sleep(5 * time.Second) // Add some delay to avoid making too many requests
	}
}

func getUpdates(offset int, timeout int, allowedUpdates string) ([]Update, error) {
	url := fmt.Sprintf(apiURL, botToken, offset, timeout, allowedUpdates)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result UpdateResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return result.Result, nil
}

func processMessage(message *Message, pin rpio.Pin) error {
	chatID := message.Chat.ID
	text := message.Text

	pattern := `^water the plants for (\d+) seconds$`

	re := regexp.MustCompile(pattern)

	match := re.FindStringSubmatch(text)

	// Check if there was a match and extract the seconds value
	if len(match) >= 2 {
		secondsStr := match[1]
		seconds, err := strconv.Atoi(secondsStr)
		if err != nil {
			return fmt.Errorf("processMessage: %v", err)
		}
		pin.High()
		time.Sleep(time.Duration(seconds) * time.Second)
		pin.Low()
		return sendMessage(chatID, fmt.Sprintf("Watered the plants for %d seconds.\n", seconds))
	} else {
		return sendMessage(chatID, "Wrong input")
	}
}

func sendMessage(chatID int64, text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	payload := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("sendMessage: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("sendMessage: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sendMessage: Error sending message. Status code: %d", resp.StatusCode)
	}
	return nil
}

type UpdateResponse struct {
	Result []Update `json:"result"`
}

type Update struct {
	ID      int      `json:"update_id"`
	Message *Message `json:"message"`
}

type Message struct {
	MessageID int    `json:"message_id"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
}

type Chat struct {
	ID int64 `json:"id"`
}
