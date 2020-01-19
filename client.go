package main

import (
	"encoding/json"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/gorilla/websocket"
	"github.com/joshbetz/config"
)

type Callback struct {
	Type string
}

func main() {
	log.SetFlags(0)

	var conf_redis, conf_token, conf_addr string

	conf := config.New("config.json")

	conf.Get("redis", &conf_redis)
	conf.Get("token", &conf_token)
	conf.Get("server", &conf_addr)

	if conf_redis == "" || conf_token == "" || conf_addr == "" {
		log.Fatal("配置文件不正确，请在当前目录创建 configs.json 文件" + `
{
    "token":"xxooxxoo",
    "server":"127.0.0.1:8080",
    "redis":"127.0.0.1:6379"
}`)
	}

	//尝试建立 Redis 连接
	redisClient := redisInit(conf_redis)
	info, _ := redisClient.LLen("callback").Result()
	log.Printf("队列中有 %d 条记录\n", info)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: conf_addr, Path: "/token"}
	log.Printf("连接服务器 %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("连接服务器失败:", err)
	}
	defer c.Close()

	//建立连接后声明身份
	conf_token = `{"token":"` + conf_token + `"}`
	err = c.WriteMessage(websocket.TextMessage, []byte(conf_token))
	if err != nil {
		log.Fatal("消息发送失败:", err)
		return
	} else {
		log.Printf("上报消息: %s", conf_token)
	}

	done := make(chan struct{})

	go func() {
		defer close(done)

		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("读取服务器响应出错:", err)
				return
			}

			log.Printf("服务器消息: %s", message)

			var callback Callback
			err = json.Unmarshal(message, &callback)
			//是任务数据写入队列
			if err == nil {
				redisClient.LPush("callback", message)
			}
		}
	}()

	//定时器保活
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			err := c.WriteMessage(websocket.TextMessage, []byte(conf_token))
			if err != nil {
				log.Println("write:", err)
				return
			}
		case <-interrupt:
			log.Println("检测到中断操作")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("请求断开出现异常:", err)
				return
			} else {
				log.Println("已断开与服务器的连接")
			}

			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}

func redisInit(addr string) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	_, err := client.Ping().Result()

	if err != nil {
		log.Fatal("redis 连接出错:" + err.Error())
	}

	return client
}