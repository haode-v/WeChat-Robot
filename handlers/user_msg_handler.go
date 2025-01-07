package handlers

import (
	"fmt"
	"log"
	"time"

	"github.com/eatmoreapple/openwechat"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // SQLite 驱动
)

var _ MessageHandlerInterface = (*UserMessageHandler)(nil)

type UserMessageHandler struct {
	db *sqlx.DB
}

// NewUserMessageHandler 创建私聊处理器
func NewUserMessageHandler(db *sqlx.DB) MessageHandlerInterface {
	return &UserMessageHandler{db: db}
}

// handle 处理消息
func (g *UserMessageHandler) handle(msg *openwechat.Message) error {
	if msg.IsText() {
		content := msg.Content
		if content == "开始" {
			// 用户发送了“开始”，启动定时任务
			go g.startTweetUpdateTask(msg)
			return nil
		}
		return g.ReplyText(msg)
	}
	return nil
}

// ReplyText 发送文本消息
func (g *UserMessageHandler) ReplyText(msg *openwechat.Message) error {
	sender, err := msg.Sender()
	log.Printf("Received User %v Text Msg : %v", sender.NickName, msg.Content)

	// 直接回复预设的消息
	replyText := "你好！请输入‘开始’以启动数据查询功能。"
	_, err = msg.ReplyText(replyText)
	if err != nil {
		log.Printf("response user error: %v \n", err)
	}
	return err
}

// startTweetUpdateTask 启动定时任务，每隔 5 分钟查询数据库中的最新 Tweets 数据
func (g *UserMessageHandler) startTweetUpdateTask(msg *openwechat.Message) {
	sender, err := msg.Sender()
	if err != nil {
		log.Printf("获取消息发送者失败：%v", err)
		return
	}
	log.Printf("User %v started the task.", sender.NickName)

	// 定时任务，每隔 5 分钟查询一次
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 查询最新的 Tweet 数据
			latestTweet, err := g.getLatestTweetFromDB()
			if err != nil {
				log.Printf("获取最新 Tweets 数据失败: %v", err)
				continue
			}

			// 回复最新的 Tweet 数据给用户
			replyText := fmt.Sprintf("最新 Tweet:\n%s", latestTweet)
			_, err = msg.ReplyText(replyText)
			if err != nil {
				log.Printf("回复用户失败：%v", err)
			}
		}
	}
}

// getLatestTweetFromDB 从数据库中获取最新的 Tweet
func (g *UserMessageHandler) getLatestTweetFromDB() (string, error) {
	var tweet string
	query := "SELECT tweet_content FROM Tweets ORDER BY created_at DESC LIMIT 1"
	err := g.db.Get(&tweet, query)
	if err != nil {
		log.Printf("查询数据库失败: %v", err)
		return "", err
	}
	return tweet, nil
}
