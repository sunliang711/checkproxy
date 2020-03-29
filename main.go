package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/sunliang711/goutils/config"
)

const (
	OK    = 1
	Error = 0
)

var (
	lastStatus = OK

	proxy    string
	timeout  int
	dest     string
	interval int

	apiServer string
	receiver  string
	authKey   string
)

func main() {
	err := config.InitConfigLogger()
	if err != nil {
		logrus.Fatalf("Init config logger error: %s", err.Error())
	}

	proxy = viper.GetString("proxy")
	timeout = viper.GetInt("timeout")
	dest = viper.GetString("dest")
	interval = viper.GetInt("interval")

	logrus.Infof("proxy: %s", proxy)
	logrus.Infof("timeout: %d ms", timeout)
	logrus.Infof("dest: %s", dest)
	logrus.Infof("interval: %d s", interval)

	apiServer = viper.GetString("email.apiServer")
	receiver = viper.GetString("email.receiver")
	authKey = viper.GetString("email.auth_key")

	logrus.Infof("email.apiServer: %s", apiServer)
	logrus.Infof("email.receiver: %s", receiver)
	logrus.Infof("email.authKey: %s", authKey)

	proxyURL, err := url.Parse(proxy)
	if err != nil {
		logrus.Fatalf("Parse proxy url error: %s", err.Error())

	}

	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
		Timeout:   time.Millisecond * time.Duration(timeout),
	}

	logrus.Debug("Enter check")
	check(client)

}

func check(client *http.Client) {
	tick := time.Tick(time.Second * time.Duration(interval))
	for {
	INNER:
		select {
		case <-tick:
			logrus.Debugf("clien.Get(%s)", dest)
			resp, err := client.Get(dest)
			if err != nil {
				logrus.Errorf("Get %s error: %s", dest, err.Error())
				notify(Error)
				break INNER
			}
			resp.Body.Close()
			logrus.Infof("Get %s OK", dest)
			notify(OK)

		}
	}
}

func notify(status int) {
	subject := fmt.Sprintf("Proxy: %s", proxy)
	if status == OK {
		subject += " work"
	} else {
		subject += " not work"
	}
	logrus.Debugf(subject)

	//send mail on when status changs
	if status != lastStatus {
		reqBody := struct {
			To      string `json:"to"`
			Subject string `json:"subject"`
			Body    string `json:"body"`
			AuthKey string `json:"auth_key"`
		}{
			To:      receiver,
			Subject: subject,
			Body:    subject,
			AuthKey: authKey,
		}
		bs, err := json.Marshal(&reqBody)
		if err != nil {
			logrus.Errorf("Marshal reqBody error: %s", err.Error())
			return
		}

		_, err = http.Post(apiServer, "application/json", bytes.NewReader(bs))
		if err != nil {
			logrus.Error("Send mail error: %s", err)
			return
		}
		logrus.Info("Email sent")

	}
	lastStatus = status
}
