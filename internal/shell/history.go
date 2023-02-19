package shell

type history struct {
	list  []string
	index int
}

func newHistory() history {
	return history{}
}

func (h *history) add(item string) {
	h.list = append(h.list, item)
	h.index = len(h.list)
}

func (h *history) previous() string {
	if h.index > 0 {
		h.index--
		return h.list[h.index]
	}
	return ""
}

func (h *history) next() (string, bool) {
	if len(h.list)-1 > h.index {
		h.index++
		return h.list[h.index], true
	} else if len(h.list) > h.index {
		h.index++
		return "", true
	}
	return "", false
}
