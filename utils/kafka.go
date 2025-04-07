package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
)

// ----------------------------------------------------------------------
// Globals & Redis initialization
// ----------------------------------------------------------------------
var RedisClient *redis.Client

func InitRedisClient() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}

// ----------------------------------------------------------------------
// WebSocket connections: map userID -> websocket.Conn
// ----------------------------------------------------------------------
var (
	WebSocketClients = make(map[string]*websocket.Conn)
	ClientsMutex     = sync.RWMutex{}
)

// ----------------------------------------------------------------------
// Redis key prefixes
// ----------------------------------------------------------------------
const (
	UserOnlinePrefix     = "user:online:"
	UserLastActivePrefix = "user:lastactive:"
	UserUnreadPrefix     = "user:unread:"
)

// ----------------------------------------------------------------------
// Message: unified struct for storing/publishing messages
// ----------------------------------------------------------------------
type Message struct {
	SenderID   string    `json:"sender_id"`
	Sender     string    `json:"sender"` // e.g., display name
	ReceiverID string    `json:"receiver_id"`
	Content    string    `json:"content"`
	Timestamp  time.Time `json:"timestamp"`
	MsgType    string    `json:"msg_type"` // e.g., text, image, file
}

// ----------------------------------------------------------------------
// Kafka Writers / Consumers
// ----------------------------------------------------------------------

// SendMessageToKafka writes a Message into Kafka with (senderID+receiverID) as the Key
func SendMessageToKafka(topic string, msg Message) error {
	w := kafka.Writer{
		Addr:         kafka.TCP("8.152.221.3:9091"),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: 10 * time.Second,
	}
	defer w.Close()

	value, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	keyStr := msg.SenderID + msg.ReceiverID // Or build the room key however you like
	err = w.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte(keyStr),
		Value: value,
		Time:  time.Now(),
	})
	if err == nil {
		// On success, store in Redis for quick access
		saveMessageToRedis(topic, msg)
	}
	return err
}

// Save message to Redis (LPUSH + LTRIM)
func saveMessageToRedis(topic string, msg Message) {
	chatKey := fmt.Sprintf("chat:%s:%s", msg.SenderID, msg.ReceiverID)
	msgData, _ := json.Marshal(msg)

	// Insert message at head of the list
	err := RedisClient.LPush(context.Background(), chatKey, string(msgData)).Err()
	if err != nil {
		log.Printf("保存消息到Redis失败: %v", err)
		return
	}

	// Keep only 100 messages
	RedisClient.LTrim(context.Background(), chatKey, 0, 99)

	// If receiver is offline, increment unread
	if !IsUserOnline(msg.ReceiverID) {
		RedisClient.HIncrBy(context.Background(), UserUnreadPrefix+msg.ReceiverID, msg.SenderID, 1)
	}
}

// LoadAllMessages scans the entire Kafka topic from earliest offset
// for the given room1 or room2 keys. In practice, you’d want a better approach.
func LoadAllMessages(room1, room2 string) ([]Message, error) {
	var allMsgs []Message

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{"8.152.221.3:9092"},
		Topic:       "test",
		GroupID:     "", // empty => no offset commit
		StartOffset: kafka.FirstOffset,
		MinBytes:    1,
		MaxBytes:    10e6,
	})
	defer r.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			// likely finished or timed out
			break
		}
		if string(m.Key) == room1 || string(m.Key) == room2 {
			var msg Message
			if jErr := json.Unmarshal(m.Value, &msg); jErr == nil {
				allMsgs = append(allMsgs, msg)
			}
		}
	}
	return allMsgs, nil
}

// ----------------------------------------------------------------------
// Redis + WebSocket Helpers
// ----------------------------------------------------------------------

// IsUserOnline checks Redis to see if the user is marked online
func IsUserOnline(userID string) bool {
	val, err := RedisClient.Get(context.Background(), UserOnlinePrefix+userID).Result()
	if err != nil || val == "" {
		return false
	}
	return true
}

// SetUserOnline sets user’s online status in Redis
func SetUserOnline(userID string) error {
	return RedisClient.Set(context.Background(), UserOnlinePrefix+userID, "1", time.Hour).Err()
}

// SetUserOffline removes user’s online status
func SetUserOffline(userID string) error {
	return RedisClient.Del(context.Background(), UserOnlinePrefix+userID).Err()
}

// UpdateUserActivity updates user’s last-active timestamp
func UpdateUserActivity(userID string) error {
	return RedisClient.Set(context.Background(), UserLastActivePrefix+userID, time.Now().Unix(), time.Hour*24).Err()
}

// SendMessageViaWebSocket tries to deliver JSON to an online user
func SendMessageViaWebSocket(receiverID string, message []byte) bool {
	ClientsMutex.RLock()
	conn, exists := WebSocketClients[receiverID]
	ClientsMutex.RUnlock()

	if exists {
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("发送WebSocket消息失败: %v", err)
			return false
		}
		return true
	}
	return false
}

// ----------------------------------------------------------------------
// Chat history from Redis
// ----------------------------------------------------------------------
func GetChatHistory(senderID, receiverID string, limit int64) ([]Message, error) {
	key1 := fmt.Sprintf("chat:%s:%s", senderID, receiverID)
	key2 := fmt.Sprintf("chat:%s:%s", receiverID, senderID)

	// Try key1 first
	msgStrings, err := RedisClient.LRange(context.Background(), key1, 0, limit-1).Result()
	if err != nil || len(msgStrings) == 0 {
		// fallback key2
		msgStrings, err = RedisClient.LRange(context.Background(), key2, 0, limit-1).Result()
		if err != nil {
			return nil, err
		}
	}

	var result []Message
	for _, s := range msgStrings {
		var m Message
		if err := json.Unmarshal([]byte(s), &m); err == nil {
			result = append(result, m)
		}
	}
	return result, nil
}

// WirteMessagesToRedis writes a slice of messages to Redis for caching
func WirteMessagesToRedis(senderID, receiverID string, messages []Message) {
	key := fmt.Sprintf("chat:%s:%s", senderID, receiverID)
	for _, msg := range messages {
		msgData, _ := json.Marshal(msg)
		RedisClient.RPush(context.Background(), key, msgData)
	}
}

// ----------------------------------------------------------------------
// Unread Counters
// ----------------------------------------------------------------------

// ClearUnreadMessages - remove senderID’s entry in userID’s unread map
func ClearUnreadMessages(userID, senderID string) error {
	return RedisClient.HDel(context.Background(), UserUnreadPrefix+userID, senderID).Err()
}

// IncrementUnreadCount - if a user is offline, we can track the unread increment
func IncrementUnreadCount(receiverID, senderID string) error {
	return RedisClient.HIncrBy(context.Background(), UserUnreadPrefix+receiverID, senderID, 1).Err()
}

// GetUnreadMessageCount returns all senders->count from user’s unread map
func GetUnreadMessageCount(userID string) (map[string]int64, error) {
	result, err := RedisClient.HGetAll(context.Background(), UserUnreadPrefix+userID).Result()
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64)
	for senderID, v := range result {
		var count int64
		fmt.Sscanf(v, "%d", &count)
		counts[senderID] = count
	}
	return counts, nil
}
