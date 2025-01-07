package handlers

import (
	"log"

	"github.com/869413421/wechatbot/config"
	"github.com/eatmoreapple/openwechat"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // SQLite 驱动
)

// MessageHandlerInterface 消息处理接口
type MessageHandlerInterface interface {
	handle(*openwechat.Message) error
	ReplyText(*openwechat.Message) error
}

type HandlerType string

const (
	UserHandler = "user"
)

// handlers 所有消息类型类型的处理器
var handlers map[HandlerType]MessageHandlerInterface
var db *sqlx.DB // 全局数据库连接

// init 函数：初始化数据库连接并传递给处理器
func init() {
	// 打开 SQLite 数据库
	var err error
	db, err = sqlx.Open("sqlite3", "/home/xonedev/project/tweet_monitoring.db")
	if err != nil {
		log.Fatalf("打开数据库失败: %v", err)
		return
	}

	// 初始化消息处理器
	handlers = make(map[HandlerType]MessageHandlerInterface)
	handlers[UserHandler] = NewUserMessageHandler(db)
}

// Handler 全局处理入口
func Handler(msg *openwechat.Message) {
	log.Printf("Received msg: %v", msg.Content)

	// 好友申请
	if msg.IsFriendAdd() {
		if config.LoadConfig().AutoPass {
			_, err := msg.Agree("你好，我是一个微信机器人，欢迎添加我为好友！")
			if err != nil {
				log.Fatalf("add friend agree error : %v", err)
				return
			}
		}
	}

	// 私聊
	handlers[UserHandler].handle(msg)
}
