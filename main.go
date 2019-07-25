package main

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/joho/godotenv"
	_ "github.com/joho/godotenv/autoload"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Config struct {
	NotifyUrl     string
	ContainerName string
}

const (
	Unhealthy = "health_status: unhealthy" // Unhealthy indicates that the container has a problem
)

func main() {
	log.Println("robot is starting...")

	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}
	config := getNotifyConfig()

	eventFilters := filters.NewArgs()
	eventFilters.Add("event", "health_status")
	eventFilters.Add("container", config.ContainerName)
	messages, errs := cli.Events(context.Background(), types.EventsOptions{
		Filters: eventFilters,
	})

loop:
	for {
		select {
		case err := <-errs:
			if err != nil && err != io.EOF {
				log.Fatal(err)
			}
			break loop
		case e := <-messages:
			if e.Status == Unhealthy {
				log.Printf("接收到容器不健康事件; message: %+v\n", e)
				postNotify(e, config.NotifyUrl)
			}
		}
	}
}

func postNotify(msg events.Message, url string) {
	body := strings.NewReader(fmt.Sprintf(`{"msgtype":"text","text":{"content":"@所有人 程序爆炸啦! Id: %s, Name: %s, Time: %s"}}`, msg.ID[0:6],
		msg.Actor.Attributes["name"], time.Unix(msg.Time, 0).Format("2006-01-02 15:04:05")))
	request := &http.Request{}
	request, err := http.NewRequest(http.MethodPost, url, body)
	if request == nil {
		log.Fatal("request is nil")
	}
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	resp, err := http.DefaultClient.Do(request)
	checkErr(err)
	respData, err := ioutil.ReadAll(resp.Body)
	checkErr(err)
	log.Printf("叮叮消息发送结果 %s\n", string(respData))
	_ = resp.Body.Close()
}

func getNotifyConfig() Config {
	var (
		err error
	)
	err = godotenv.Load()
	checkErr(err)
	config := Config{}
	config.NotifyUrl = os.Getenv("NOTIFY_URL")
	config.ContainerName = os.Getenv("CONTAINER_NAME")
	return config
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
