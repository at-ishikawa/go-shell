package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"go.uber.org/zap"
)

type HistoryItem struct {
	Command string `json:"command"`
	// @deprecated
	Status int `json:"status,omitempty"`

	// @deprecated
	RunAt time.Time `json:"run_at,omitempty"`

	LastSucceededAt time.Time         `json:"last_succeeded_at"`
	LastFailedAt    time.Time         `json:"last_failed_at,omitempty"`
	Count           int               `json:"count"`
	Context         map[string]string `json:"context,omitempty"`
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

	if err := json.Unmarshal(fileData, &h.list); err != nil {
		return err
	}

	h.index = len(h.list)
	return nil
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

func (h *History) Sync(
	command string,
	status int,
	currentContext map[string]string,
	logger *zap.Logger,
) chan struct{} {
	ch := make(chan struct{})
	go func() {
		if err := h.LoadFile(); err != nil {
			logger.Error("Failed to load a history file",
				zap.Error(err),
			)
			return
		}
		var currentTime time.Time
		if !h.currentTime.Equal(time.Time{}) {
			currentTime = h.currentTime
		} else {
			currentTime = time.Now()
		}

		h.Add(command, status, currentContext, currentTime)
		if err := h.saveFile(); err != nil {
			logger.Error("Failed to save a history file",
				zap.Error(err),
			)
			return
		}
		ch <- struct{}{}
	}()

	return ch
}

func (h *History) Add(command string, status int, currentContext map[string]string, currentTime time.Time) {
	var lastSucceededAt time.Time
	var lastFailedAt time.Time
	count := 1

	for index, item := range h.list {
		if command == item.Command && reflect.DeepEqual(item.Context, currentContext) {
			// remove this element
			h.list = append(h.list[:index], h.list[index+1:]...)

			lastSucceededAt = item.LastSucceededAt
			lastFailedAt = item.LastFailedAt
			count = item.Count + 1
			break
		}
	}

	if status == 0 {
		lastSucceededAt = currentTime
	} else {
		lastFailedAt = currentTime
	}

	h.list = append(h.list, HistoryItem{
		Command:         command,
		LastSucceededAt: lastSucceededAt,
		LastFailedAt:    lastFailedAt,
		Context:         currentContext,
		Count:           count,
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
