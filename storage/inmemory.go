package storage

// In memory implementation is written by AI in a hurry

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

type InMemoryStorage struct {
	data  map[string]string
	mutex sync.Mutex
}

func (i *InMemoryStorage) Get(chatID int64, msgID int) (int64, int, error) {
	query := fmt.Sprintf("qbot:%d:%d", chatID, msgID)
	i.mutex.Lock()
	defer i.mutex.Unlock()
	result, ok := i.data[query]
	if !ok {
		return 0, 0, fmt.Errorf("not found")
	}
	parts := strings.Split(result, ":")
	resultChatID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	resultMsgID, err := strconv.ParseInt(parts[1], 10, 64)
	return resultChatID, int(resultMsgID), err
}

func (i *InMemoryStorage) Set(chatID int64, msgID int, chatID2 int64, msgID2 int) error {
	query := fmt.Sprintf("qbot:%d:%d", chatID, msgID)
	value := fmt.Sprintf("%d:%d", chatID2, msgID2)
	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.data[query] = value
	return nil
}

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		data:  make(map[string]string),
		mutex: sync.Mutex{},
	}
}
