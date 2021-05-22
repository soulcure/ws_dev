package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"idreamsky.com/fanbook/config"
)

var index int = 0

func main() {
	f := config.NewLogFile()
	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetOutput(f)

	//logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
	})
	logrus.SetLevel(logrus.DebugLevel)
	defer func() {
		if err := f.Close(); err != nil {
			logrus.Printf("close log file error: %s", err)
		}
	}()

	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: config.WsConfig.WsInfo.Scheme, Host: config.WsConfig.WsInfo.Host}
	logrus.Debug("connecting to=", u.String())

	headerMap := map[string]string{
		"platform":     config.HeaderCfg.HeaderInfo.Platform,
		"version":      config.HeaderCfg.HeaderInfo.Version,
		"channel":      config.HeaderCfg.HeaderInfo.Channel,
		"device_id":    config.HeaderCfg.HeaderInfo.Device_id,
		"build_number": config.HeaderCfg.HeaderInfo.Build_number,
	}

	jsonStr, err := json.Marshal(headerMap)

	logrus.Debug("WebSocketDebug jsonString=:", jsonStr)

	if err != nil {
		logrus.Error("json.Marshal failed:", err)
		return
	}
	bs := base64.StdEncoding.EncodeToString(jsonStr)
	logrus.Debug("WebSocketDebug base64=:", bs)

	var p = url.Values{}
	p.Add("id", config.Account.Account.Token)
	p.Add("dId", config.HeaderCfg.HeaderInfo.Device_id)
	p.Add("v", config.HeaderCfg.HeaderInfo.Version)
	p.Add("x-super-properties", bs)

	uri := u.String() + "?" + p.Encode()
	logrus.Debug("WebSocketDebug uri=", uri)

	c, _, err := websocket.DefaultDialer.Dial(uri, nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			msgType, message, err := c.ReadMessage()
			if err != nil {
				logrus.Error("read:", err)
				return
			}

			if msgType == websocket.TextMessage {
				newStr := string(message)
				logrus.Debug("recv: %s", newStr)
			} else if msgType == websocket.BinaryMessage {
				newStr := UGZipBytes(message)
				logrus.Debug("recv: %s", newStr)
			}

		}
	}()

	ticker := time.NewTicker(time.Millisecond * 100) //100毫秒发送一条消息
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if index > config.RuleCfg.Times {
				c.Close()
				return
			}

			err := c.WriteMessage(websocket.TextMessage, getSendMessage(done))
			if err != nil {
				log.Println("write:", err)
				return
			}
			//log.Println("ticker.C")
		case <-interrupt:
			log.Println("interrupt")
			c.Close()
			return
			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			// err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			// if err != nil {
			// 	log.Println("write close:", err)
			// 	return
			// }
		}
	}
}

func getSendMessage(done chan struct{}) []byte {
	str := fmt.Sprintf("第%d条消息：%s", index, config.RuleCfg.Content)
	index++
	return []byte(str)
}

// //压缩
// func GZipBytes(data []byte) []byte {
// 	var input bytes.Buffer
// 	g := gzip.NewWriter(&in) //面向api编程调用压缩算法的一个api
// 	//参数就是指向某个数据缓冲区默认压缩等级是DefaultCompression 在这里还有另一个api可以调用调整压缩级别
// 	//gzip.NewWirterLevel(&in,gzip.BestCompression) NoCompression（对应的int 0）、
// 	//BestSpeed（1）、DefaultCompression（-1）、HuffmanOnly（-2）BestCompression（9）这几个级别也可以
// 	//这样写gzip.NewWirterLevel(&in,0)
// 	//这里的异常返回最好还是处理下，我这里纯属省事
// 	g.Write(data)
// 	g.Close()
// 	return input.Bytes()
// }

//解压
func UGZipBytes(data []byte) string {

	buf := bytes.NewBuffer(data)

	zr, err := gzip.NewReader(buf)
	if err != nil {
		return string(data)
	}

	res := new(strings.Builder)

	if _, err := io.Copy(res, zr); err != nil {
		log.Fatal(err)
	}

	if err := zr.Close(); err != nil {
		log.Fatal(err)
	}

	return res.String()
}
