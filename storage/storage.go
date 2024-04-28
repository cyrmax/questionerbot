package storage

type Storage interface {
	Get(chatID int64, msgID int) (int64, int, error)
	Set(chatID int64, msgID int, chatID2 int64, msgID2 int) error
}
