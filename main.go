package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/jinzhu/configor"
	"github.com/wq1019/ding_talk"
	"io"
	"log"
	"sync"
	"time"
)

// config
type Config struct {
	NotifyUrls []string `required:"true"`
	Filters    *Filters
}
type Filters struct {
	Name  []string `json:"name"`
	Event []string `json:"event"`
	Type  []string `json:"type"`
}

const (
	Unhealthy = "health_status: unhealthy"
	Healthy   = "health_status: healthy"
)

func main() {
	var (
		config  Config
		err     error
		ctx     context.Context
		timeout time.Duration
	)
	timeout = time.Second * 3

	ctx = context.Background()

	cli, err := client.NewEnvClient()
	checkErr(err)

	err = configor.Load(&config, "config.yml")
	checkErr(err)

	// ding ding clients
	dingClients := ding_talk.NewClients(config.NotifyUrls)

	messages, errs := cli.Events(ctx, types.EventsOptions{
		Filters: buildFilters(config.Filters),
	})
	checkErr(err)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case err := <-errs:
				if err != nil && err != io.EOF {
					panic(err)
				}
			case e := <-messages:
				log.Printf("接收到新的 Docker 事件：%+v\n", e)

				markdown := ding_talk.MarkdownMessage{
					MsgType: ding_talk.Markdown,
					At: &ding_talk.At{
						IsAtAll: true,
					},
				}
				var id string
				if len(e.ID) > 8 {
					id = e.ID[:8]
				} else {
					id = e.ID
				}
				switch e.Status {
				case Unhealthy:
					markdown.Markdown.Title = "程序爆炸啦"
					markdown.Markdown.Text = fmt.Sprintf("#### 服务爆炸啦\n"+
						"> ID：%s\n\n"+
						"> 名称：%s\n\n"+
						"> 服务状态：unhealthy\n\n"+
						"> ###### %s发布 [来自优品单通知](https://ypdan.com)\n", id, e.Actor.Attributes["name"], timeFormat(e.Time))
					// 重启服务
					checkErr(cli.ContainerRestart(ctx, e.ID, &timeout))
				case Healthy:
					markdown.Markdown.Title = "程序恢复正常"
					markdown.Markdown.Text = fmt.Sprintf("#### 程序已经恢复正常啦\n"+
						"> ID：%s\n\n"+
						"> 名称：%s\n\n"+
						"> 服务状态：healthy\n\n"+
						"> ###### %s发布 [来自优品单通知](https://ypdan.com)\n", id, e.Actor.Attributes["name"], timeFormat(e.Time))
				default:

					//continue
					markdown.Markdown.Title = "其他通知"
					markdown.Markdown.Text = fmt.Sprintf("#### 其他通知\n"+
						"> ID：%s\n\n"+
						"> 名称：%s\n\n"+
						"> 服务状态：%s\n\n"+
						"> ###### %s发布 [来自优品单通知](https://ypdan.com)\n", id, e.Actor.Attributes["name"], e.Status, timeFormat(e.Time))
				}
				for _, c := range dingClients {
					go func(client ding_talk.DingTalkClient) {
						resp, _ := client.Execute(markdown)
						if resp.ErrCode != 0 {
							checkInfo(errors.New(fmt.Sprintf("发送通知失败 err: %s\n", resp.ErrMsg)))
						}
					}(c)
				}
			}
		}
	}()
	log.Println("Robot is running...")
	log.Println("Waiting for docker event...")
	wg.Wait()
}

func timeFormat(timeInt int64) string {
	t := time.Unix(timeInt, 0)
	return fmt.Sprintf("%d月%d日%d时%d分%d秒", t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}

func buildFilters(config *Filters) filters.Args {
	body, err := json.Marshal(config)
	checkErr(err)
	args, err := filters.FromParam(string(body))
	checkErr(err)
	return args
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func checkInfo(err error) {
	if err != nil {
		log.Print(err)
	}
}
