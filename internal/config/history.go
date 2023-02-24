package config

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type HistoryItem struct {
	Command string    `json:"command"`
	Status  int       `json:"status"`
	RunAt   time.Time `json:"run_at"`
}

type History struct {
	list        []HistoryItem
	index       int
	config      *Config
	fileName    string
	maxSize     int
	currentTime time.Time
}

func NewHistory(c *Config) History {
	return History{
		config:   c,
		fileName: "history.json",
		maxSize:  1000,
	}
}

func (h *History) Get() []HistoryItem {
	return h.list
}

func (h *History) LoadFile() error {
	fileData, err := h.config.readFile(h.fileName)
	if err != nil {
		return fmt.Errorf("LoadFile error: %w", err)
	}
	if len(fileData) == 0 {
		return nil
	}

	return json.Unmarshal(fileData, &h.list)
}

func (h History) saveFile() error {
	if len(h.list) > h.maxSize {
		firstIndex := len(h.list) - h.maxSize
		h.list = h.list[firstIndex:]
	}
	marshaledJson, err := json.Marshal(h.list)
	if err != nil {
		return err
	}
	return h.config.writeFile(h.fileName, marshaledJson)
}

func (h History) StartWith(inputCommand string, status int) string {
	for i := len(h.list) - 1; i >= 0; i-- {
		item := h.list[i]
		if strings.HasPrefix(item.Command, inputCommand) && item.Status == status {
			return item.Command
		}
	}
	return ""
}

func (h *History) Sync(command string, status int) chan struct{} {
	ch := make(chan struct{})
	go func() {
		if err := h.LoadFile(); err != nil {
			fmt.Println(err)
			return
		}
		h.Add(command, status)
		if err := h.saveFile(); err != nil {
			fmt.Println(err)
			return
		}
		ch <- struct{}{}
	}()

	return ch
}

func (h *History) Add(command string, status int) {
	var currentTime time.Time
	if !h.currentTime.Equal(time.Time{}) {
		currentTime = h.currentTime
	} else {
		currentTime = time.Now()
	}

	h.list = append(h.list, HistoryItem{
		Status:  status,
		Command: command,
		RunAt:   currentTime,
	})
	h.index = len(h.list)
}

func (h *History) Previous() string {
	if h.index > 0 {
		h.index--
		return h.list[h.index].Command
	}
	return ""
}

func (h *History) Next() (string, bool) {
	if len(h.list)-1 > h.index {
		h.index++
		return h.list[h.index].Command, true
	} else if len(h.list) > h.index {
		h.index++
		return "", true
	}
	return "", false
}
