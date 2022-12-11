package main

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/PullRequestInc/go-gpt3"
	"github.com/eatmoreapple/openwechat"
)

var (
	keyword = flag.String("keyword", "/gpt", "触发聊天的关键词(前缀)")
	apiKey  = flag.String("api_key", "xxxx", "chatgpt api key")
)

func chat(ctx context.Context, client gpt3.Client, quesiton string) (string, error) {
	resp, err := client.CompletionWithEngine(ctx, gpt3.TextDavinci003Engine, gpt3.CompletionRequest{
		Prompt: []string{
			quesiton,
		},
		MaxTokens:   gpt3.IntPtr(3000),
		Temperature: gpt3.Float32Ptr(0),
	})
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Text, nil
}

func checkKeyword(keyword, content string) (string, bool) {
	content = strings.Trim(strings.TrimSpace(content), "\n")
	if !strings.HasPrefix(content, keyword) {
		return "", false
	}
	content = strings.TrimPrefix(content, keyword)
	return strings.Trim(strings.TrimSpace(content), "\n"), true
}

func main() {
	flag.Parse()
	client := gpt3.NewClient(*apiKey)

	//bot := openwechat.DefaultBot()
	bot := openwechat.DefaultBot(openwechat.Desktop) // 桌面模式，上面登录不上的可以尝试切换这种模式

	var myself *openwechat.Self

	// 注册消息处理函数
	bot.MessageHandler = func(msg *openwechat.Message) {
		defer func() {
			if e := recover(); e != nil {
				fmt.Println(e)
			}
		}()

		sender, err := msg.Sender()
		if err != nil {
			fmt.Println(err)
			return
		}

		content, ok := checkKeyword(*keyword, msg.Content)
		if !ok {
			return
		}
		fmt.Println(sender.NickName, ":", content)

		var sendFunc func(text string) error
		if msg.IsSendByGroup() {
			groupSender, err := msg.SenderInGroup()
			if err != nil {
				fmt.Println("group sender", err)
				return
			}
			group := &openwechat.Group{User: sender}
			if msg.IsSendBySelf() { // FIXME 自己发送的
				if g, err := msg.Receiver(); err != nil {
					fmt.Println("group erorr", err)
					return
				} else {
					group = &openwechat.Group{User: g}
				}
			}
			sendFunc = func(text string) error {
				_, err = myself.SendTextToGroup(group, "@"+groupSender.NickName+" "+text)
				return err
			}
		} else if msg.IsSendByFriend() {
			friend := &openwechat.Friend{User: sender}
			sendFunc = func(text string) error {
				_, err = myself.SendTextToFriend(friend, "@"+sender.NickName+" "+text)
				return err
			}
		}
		if sendFunc == nil {
			return
		}

		if len(content) == 0 {
			sendFunc("请说出你的问题")
			return
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			replyText, err := chat(ctx, client, content)
			if err != nil {
				replyText = err.Error()
			}
			sendFunc(replyText)
		}()
	}
	// 注册登陆二维码回调
	bot.UUIDCallback = openwechat.PrintlnQrcodeUrl

	// 登陆
	if err := bot.Login(); err != nil {
		fmt.Println(err)
		return
	}

	// 获取登陆的用户
	self, err := bot.GetCurrentUser()
	if err != nil {
		fmt.Println(err)
		return
	}
	self.FileHelper()
	myself = self

	// 获取所有的好友
	friends, err := self.Friends()
	fmt.Println(friends, err)

	// 获取所有的群组
	groups, err := self.Groups()
	fmt.Println(groups, err)

	// 阻塞主goroutine, 直到发生异常或者用户主动退出
	bot.Block()
}
