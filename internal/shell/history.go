package shell

import (
	"encoding/json"
	"fmt"
)

type historyItem struct {
	Command string `json:"command"`
}

type history struct {
	list     []historyItem
	index    int
	config   *config
	fileName string
	maxSize  int
}

func newHistory(c *config) history {
	return history{
		config:   c,
		fileName: "history.json",
		maxSize:  1000,
	}
}

func (h *history) loadFile() error {
	fileData, err := h.config.readFile(h.fileName)
	if err != nil {
		return fmt.Errorf("loadFile error: %w", err)
	}
	if len(fileData) == 0 {
		return nil
	}

	return json.Unmarshal(fileData, &h.list)
}

func (h history) saveFile() error {
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

func (h *history) add(command string) {
	h.list = append(h.list, historyItem{
		Command: command,
	})
	h.index = len(h.list)
}

func (h *history) previous() string {
	if h.index > 0 {
		h.index--
		return h.list[h.index].Command
	}
	return ""
}

func (h *history) next() (string, bool) {
	if len(h.list)-1 > h.index {
		h.index++
		return h.list[h.index].Command, true
	} else if len(h.list) > h.index {
		h.index++
		return "", true
	}
	return "", false
}
