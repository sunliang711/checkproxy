package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/sunliang711/goutils/config"
)

const (
	OK = iota
	Error
)

var (
	proxiesStatus map[string]int
	timeout       int
	dest          string
	interval      int

	apiServer string
	receiver  string
	authKey   string
)

func main() {
	proxiesStatus = make(map[string]int)

	err := config.InitConfigLogger()
	if err != nil {
		logrus.Fatalf("Init config logger error: %s", err.Error())
	}

	proxies := viper.GetStringSlice("proxies")
	timeout = viper.GetInt("timeout")
	dest = viper.GetString("dest")
	interval = viper.GetInt("interval")

	logrus.Infof("proxies: %v", proxies)
	logrus.Infof("timeout: %d ms", timeout)
	logrus.Infof("dest: %s", dest)
	logrus.Infof("interval: %d s", interval)

	apiServer = viper.GetString("email.apiServer")
	receiver = viper.GetString("email.receiver")
	authKey = viper.GetString("email.auth_key")

	logrus.Infof("email.apiServer: %s", apiServer)
	logrus.Infof("email.receiver: %s", receiver)
	logrus.Infof("email.authKey: %s", authKey)

	for _, proxy := range proxies {
		parsedProxy, err := url.Parse(proxy)
		if err != nil {
			logrus.WithFields(logrus.Fields{"proxy": proxy, "err": err.Error()}).Fatalf("parse proxy url")
		}

		logrus.Debugf("parsedProxy: %v", parsedProxy)
		client := &http.Client{
			Transport: &http.Transport{Proxy: http.ProxyURL(parsedProxy)},
			Timeout:   time.Millisecond * time.Duration(timeout),
		}

		// init status is OK
		proxiesStatus[proxy] = OK

		go check(client, proxy)
	}
	select {}
}

func check(client *http.Client, proxy string) {
	var result int
	for {
		logrus.Debugf("client.Get(%s)", dest)
	INNER:
		for i := 1; i < viper.GetInt("retry"); i++ {
			_, err := client.Get(dest)
			if err != nil {
				logrus.Debugf("retry %d error: %s", i, err.Error())
				result = Error
				time.Sleep(time.Duration(viper.GetInt("retrydelay")) * time.Second)
				continue INNER
			} else {
				result = OK
				break INNER
			}
		}

		notify(proxy, result)
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func notify(proxy string, status int) {
	var subject string
	if status == OK {
		subject += " [ Work ] "
	} else {
		subject += " [ Not work ] "
	}
	subject += proxy
	logrus.Debugf(subject)

	//send mail on when status changs
	if status != proxiesStatus[proxy] {
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
			logrus.Errorf("Send mail error: %s", err)
			return
		}
		logrus.Info("Email sent")

	}
	proxiesStatus[proxy] = status
}
