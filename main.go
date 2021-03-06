package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"idreamsky.com/fanbook/config"
)

var seq int = 0

func main() {
	f := config.NewLogFile()
	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetOutput(f)

	//logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableColors:   true,
		TimestampFormat: time.StampMilli,
	})
	logrus.SetLevel(logrus.DebugLevel)
	defer func() {
		if err := f.Close(); err != nil {
			logrus.Error("close log file error: %s", err)
		}
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: config.WsConfig.WsInfo.Scheme, Host: config.WsConfig.WsInfo.Host, Path: config.WsConfig.WsInfo.Path}
	fmt.Println("connecting to=", u.String())

	headerMap := map[string]string{
		"platform":     config.HeaderCfg.HeaderInfo.Platform,
		"version":      config.HeaderCfg.HeaderInfo.Version,
		"channel":      config.HeaderCfg.HeaderInfo.Channel,
		"device_id":    config.HeaderCfg.HeaderInfo.Device_id,
		"build_number": config.HeaderCfg.HeaderInfo.Build_number,
	}

	jsonStr, err := json.Marshal(headerMap)

	if err != nil {
		logrus.Error("json.Marshal failed:", err)
		return
	}
	bs := base64.StdEncoding.EncodeToString(jsonStr)

	var p = url.Values{}
	p.Add("id", config.Account.Account.Token)
	p.Add("dId", config.HeaderCfg.HeaderInfo.Device_id)
	p.Add("v", config.HeaderCfg.HeaderInfo.Version)
	p.Add("x-super-properties", bs)

	uri := u.String() + "?" + p.Encode()
	fmt.Println("WebSocketDebug uri=", uri)

	c, _, err := websocket.DefaultDialer.Dial(uri, nil)
	if err != nil {
		logrus.Error("dial:", err)
	}

	fmt.Println("connect websocket success", u.String())
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			msgType, message, err := c.ReadMessage()
			if err != nil {
				fmt.Println("read:", err)
				return
			}

			if msgType == websocket.TextMessage {
				newStr := string(message)
				logrus.Debug("recv string =", newStr)
			} else if msgType == websocket.BinaryMessage {
				newStr := UGZipBytes(message)
				logrus.Debug("recv binary =", newStr)
			}

		}
	}()

	interval := config.RuleCfg.Interval
	ticker := time.NewTicker(time.Duration(interval) * time.Millisecond) //100????????????????????????

	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if seq > config.RuleCfg.Times-1 {
				//Cleanly close the connection by sending a close message and then
				//waiting (with timeout) for the server to close the connection.
				err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				if err != nil {
					logrus.Error("write close:", err)
				}
				return
			}
			b := getSendMessage(done)

			err := c.WriteMessage(websocket.TextMessage, b)

			if err != nil {
				logrus.Error("write:", err)
				return
			} else {
				logrus.Debug("send message =", string(b))
				seq++
			}
		case <-interrupt:
			fmt.Println("interrupt")
			c.Close()
			return
		}
	}
}

func getSendMessage(done chan struct{}) []byte {
	text := fmt.Sprintf("???%d????????????%s", seq, config.RuleCfg.Content)
	msgContent := map[string]interface{}{
		"type":        "text",
		"text":        text,
		"contentType": 0,
	}

	msgBytes, err := json.Marshal(msgContent)
	if err != nil {
		fmt.Println("json.Marshal failed:", err)
		return nil
	}

	node, err := snowflake.NewNode(1)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	// Generate a snowflake ID.
	snowflakeId := int64(node.Generate())
	nonce := strconv.FormatInt(snowflakeId, 10)

	message := map[string]interface{}{
		"action":      "send",
		"channel_id":  config.RuleCfg.ChannelId,
		"seq":         seq,
		"quote_l1":    nil,
		"quote_l2":    nil,
		"guild_id":    config.RuleCfg.GuildId,
		"content":     string(msgBytes),
		"ctype":       0,
		"nonce":       nonce,
		"app_version": "1.6.2",
	}

	b, err := json.Marshal(message)
	if err != nil {
		fmt.Println("json.Marshal failed:", err)
		return nil
	}

	return b
}

// //??????
// func GZipBytes(data []byte) []byte {
// 	var input bytes.Buffer
// 	g := gzip.NewWriter(&in) //??????api?????????????????????????????????api
// 	//????????????????????????????????????????????????????????????DefaultCompression ????????????????????????api??????????????????????????????
// 	//gzip.NewWirterLevel(&in,gzip.BestCompression) NoCompression????????????int 0??????
// 	//BestSpeed???1??????DefaultCompression???-1??????HuffmanOnly???-2???BestCompression???9???????????????????????????
// 	//?????????gzip.NewWirterLevel(&in,0)
// 	//??????????????????????????????????????????????????????????????????
// 	g.Write(data)
// 	g.Close()
// 	return input.Bytes()
// }

//??????
func UGZipBytes(data []byte) string {

	buf := bytes.NewBuffer(data)

	zr, err := gzip.NewReader(buf)
	if err != nil {
		return string(data)
	}

	res := new(strings.Builder)

	if _, err := io.Copy(res, zr); err != nil {
		logrus.Error(err)
	}

	if err := zr.Close(); err != nil {
		logrus.Error(err)
	}

	return res.String()
}
