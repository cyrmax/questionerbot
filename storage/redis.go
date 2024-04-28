package storage

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

type RedisStorage struct {
	client *redis.Client
	ctx    context.Context
}

func (r *RedisStorage) Get(chatID int64, msgID int) (int64, int, error) {
	query := fmt.Sprintf("qbot:%d:%d", chatID, msgID)
	result, err := r.client.Get(r.ctx, query).Result()
	if err != nil {
		return 0, 0, err
	}
	parts := strings.Split(result, ":")
	resultChatID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	resultMsgID, err := strconv.ParseInt(parts[1], 10, 64)
	return resultChatID, int(resultMsgID), err
}

func (r *RedisStorage) Set(chatID int64, msgID int, chatID2 int64, msgID2 int) error {
	query := fmt.Sprintf("qbot:%d:%d", chatID, msgID)
	return r.client.Set(r.ctx, query, fmt.Sprintf("%d:%d", chatID2, msgID2), 0).Err()
}
