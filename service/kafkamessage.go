package service

import (
	"e-commerse/utils"
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// ----------------------------------------------------------------------
// 1) MessageRequest: unified request struct for front-end JSON payloads
// ----------------------------------------------------------------------
type MessageRequest struct {
	SenderID   string `json:"sender_id"`
	Sender     string `json:"sender"`
	ReceiverID string `json:"receiver_id"`
	Content    string `json:"content"`
	MsgType    string `json:"msg_type"`
}

// ----------------------------------------------------------------------
// WebSocket Upgrader
// ----------------------------------------------------------------------
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// For simplicity, allow all. Adjust as needed for production.
		return true
	},
}

// ----------------------------------------------------------------------
// SendMsg - HTTP POST handler to send a message to Kafka & WebSocket
// ----------------------------------------------------------------------
func SendMsg(c *gin.Context) {
	var req MessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误",
			"error":   err.Error(),
		})
		return
	}

	// Build our message struct for Kafka
	msg := utils.Message{
		SenderID:   req.SenderID,
		Sender:     req.Sender,
		ReceiverID: req.ReceiverID,
		Content:    req.Content,
		Timestamp:  time.Now(),
		MsgType:    req.MsgType,
	}

	// Send message to Kafka
	err := utils.SendMessageToKafka("test", msg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "发送消息失败",
			"error":   err.Error(),
		})
		return
	}

	// Check if receiver is online
	if utils.IsUserOnline(req.ReceiverID) {
		// Marshal message to JSON
		msgData, _ := json.Marshal(msg)

		// Try to send via WebSocket
		sent := utils.SendMessageViaWebSocket(req.ReceiverID, msgData)
		if sent {
			log.Printf("消息通过WebSocket发送成功: %s -> %s", req.SenderID, req.ReceiverID)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "发送消息成功",
	})
}

// ----------------------------------------------------------------------
// GetMsg - HTTP GET handler to fetch history
// ----------------------------------------------------------------------
func GetMsg(c *gin.Context) {
	senderID := c.Query("sender_id")
	receiverID := c.Query("receiver_id")
	limitStr := c.DefaultQuery("limit", "20")

	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil {
		limit = 20
	}

	if senderID == "" || receiverID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误：需要sender_id和receiver_id",
		})
		return
	}

	// 先尝试从Redis获取聊天历史
	messages, err := utils.GetChatHistory(senderID, receiverID, limit)
	if err != nil || len(messages) == 0 {
		// 如果Redis获取失败，或拿到为空，就去Kafka拉
		room1 := senderID + receiverID
		room2 := receiverID + senderID

		messages, err = utils.LoadAllMessages(room1, room2)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "获取消息失败",
				"error":   err.Error(),
			})
			return
		}

		// 回填Redis
		utils.WirteMessagesToRedis(senderID, receiverID, messages)
	}

	// ============= 新增：对 messages 做时间戳排序 =============
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp.Before(messages[j].Timestamp)
	})

	// 清除未读
	utils.ClearUnreadMessages(senderID, receiverID)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取消息成功",
		"data":    messages,
	})
}

// ----------------------------------------------------------------------
// GetUnreadMessages - HTTP GET handler to fetch unread counts
// ----------------------------------------------------------------------
func GetUnreadMessages(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误：需要user_id",
		})
		return
	}

	counts, err := utils.GetUnreadMessageCount(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取未读消息失败",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取未读消息成功",
		"data":    counts,
	})
}

// ----------------------------------------------------------------------
// WebSocketHandler - Upgrades GET /ws to a WebSocket connection
// ----------------------------------------------------------------------
func WebSocketHandler(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少user_id参数",
		})
		return
	}

	// Upgrade HTTP -> WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket升级失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "WebSocket升级失败",
			"error":   err.Error(),
		})
		return
	}

	// Store connection & set user online
	utils.ClientsMutex.Lock()
	utils.WebSocketClients[userID] = conn
	utils.ClientsMutex.Unlock()

	utils.SetUserOnline(userID)
	utils.UpdateUserActivity(userID)
	log.Printf("用户 %s 已连接", userID)

	// Handle read loop in separate function
	handleWebSocketMessages(userID, conn)
}

// ----------------------------------------------------------------------
// handleWebSocketMessages - read loop for a given user + WebSocket conn
// ----------------------------------------------------------------------
func handleWebSocketMessages(userID string, conn *websocket.Conn) {
	defer func() {
		conn.Close()
		utils.ClientsMutex.Lock()
		delete(utils.WebSocketClients, userID)
		utils.ClientsMutex.Unlock()

		utils.SetUserOffline(userID)
		log.Printf("用户 %s 已断开连接", userID)
	}()

	// Periodically update user activity
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			utils.UpdateUserActivity(userID)
		}
	}()

	// Read messages from this WebSocket
	for {
		_, messageBytes, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket错误: %v", err)
			}
			break
		}

		// Unmarshal into MessageRequest (the same we use for HTTP)
		var msgReq MessageRequest
		if err := json.Unmarshal(messageBytes, &msgReq); err != nil {
			log.Printf("解析WebSocket消息失败: %v", err)
			continue
		}

		// Prevent forging sender_id
		if msgReq.SenderID == "" {
			msgReq.SenderID = userID
		} else if msgReq.SenderID != userID {
			log.Printf("警告: 尝试伪造sender_id: %s vs %s", msgReq.SenderID, userID)
			msgReq.SenderID = userID
		}

		// If valid, forward to Kafka + potential WebSocket
		if msgReq.ReceiverID != "" && msgReq.Content != "" {
			outMsg := utils.Message{
				SenderID:   msgReq.SenderID,
				Sender:     msgReq.Sender,
				ReceiverID: msgReq.ReceiverID,
				Content:    msgReq.Content,
				Timestamp:  time.Now(),
				MsgType:    msgReq.MsgType,
			}

			if err := utils.SendMessageToKafka("test", outMsg); err != nil {
				log.Printf("发送消息到Kafka失败: %v", err)
			}

			// If receiver is online, push via WebSocket
			if utils.IsUserOnline(msgReq.ReceiverID) {
				msgData, _ := json.Marshal(outMsg)
				utils.SendMessageViaWebSocket(msgReq.ReceiverID, msgData)
			} else {
				// If offline, increment unread
				utils.IncrementUnreadCount(msgReq.ReceiverID, msgReq.SenderID)
			}
		}

		// Update last activity
		utils.UpdateUserActivity(userID)
	}
}
