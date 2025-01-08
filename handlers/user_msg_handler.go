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
	db             *sqlx.DB
	stopUpdateTask chan bool         // 控制任务停止的通道
	lastTweets     map[string]string // 记录每个用户的上一次发送的 Tweet 内容
}

// NewUserMessageHandler 创建私聊处理器
func NewUserMessageHandler(db *sqlx.DB) MessageHandlerInterface {
	return &UserMessageHandler{
		db:             db,
		stopUpdateTask: make(chan bool), // 初始化停止任务的通道
		lastTweets:     make(map[string]string),
	}
}

// handle 处理消息
func (g *UserMessageHandler) handle(msg *openwechat.Message) error {
	if msg.IsText() {
		content := msg.Content
		if content == "开始" {
			// 用户发送了“开始”，启动定时任务
			go g.startTweetUpdateTask(msg)
			return nil
		} else if content == "停止" {
			// 用户发送了“停止”，停止定时任务
			g.stopUpdateTask <- true
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
	replyText := "你好！"
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

	// 定时任务，每隔 3 分钟查询一次
	ticker := time.NewTicker(3 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 查询所有用户的最新 Tweet 数据
			latestTweets, err := g.getAllLatestTweetsFromDB()
			if err != nil {
				log.Printf("获取最新 Tweets 数据失败: %v", err)
				continue
			}

			for userID, tweetContent := range latestTweets {
				// 判断是否与上一次发送的推文相同，避免重复发送
				if g.lastTweets[userID] == tweetContent {
					continue
				}

				// 回复最新的 Tweet 数据给用户
				replyText := fmt.Sprintf("用户 %s 最新 Tweet:\n%s", userID, tweetContent)
				_, err = msg.ReplyText(replyText)
				if err != nil {
					log.Printf("回复用户失败：%v", err)
				} else {
					// 更新 lastTweets，保存当前发送的推文内容
					g.lastTweets[userID] = tweetContent
				}
			}
		case <-g.stopUpdateTask:
			// 收到停止任务的信号，退出循环
			log.Printf("User %v stopped the task.", sender.NickName)
			return
		}
	}
}

// getAllLatestTweetsFromDB 从数据库中获取每个用户的最新 Tweet
func (g *UserMessageHandler) getAllLatestTweetsFromDB() (map[string]string, error) {
	latestTweets := make(map[string]string)
	query := `
		SELECT user_id, tweet_content 
		FROM Tweets 
		WHERE id IN (
			SELECT MAX(id) 
			FROM Tweets 
			GROUP BY user_id
		);
	`
	rows, err := g.db.Queryx(query)
	if err != nil {
		log.Printf("查询数据库失败: %v", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var userID, tweetContent string
		if err := rows.Scan(&userID, &tweetContent); err != nil {
			log.Printf("扫描数据库行失败: %v", err)
			continue
		}
		latestTweets[userID] = tweetContent
	}
	return latestTweets, nil
}
