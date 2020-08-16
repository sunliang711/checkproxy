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

type proxySetting struct {
	Name string
	Url  string
}

func main() {
	proxiesStatus = make(map[string]int)

	err := config.InitConfigLogger()
	if err != nil {
		logrus.Fatalf("Init config logger error: %s", err.Error())
	}

	// proxies2 := viper.Get("proxies2")
	// fmt.Printf("proxies2: %v", proxies2)
	// for _, pxy := range proxies2.([]interface{}) {
	// 	fmt.Printf("pxy: name: %v url: %v", pxy["name"], pxy["url"])
	// }
	var ps []*proxySetting
	err = viper.UnmarshalKey("proxies", &ps)
	if err != nil {
		logrus.Fatalf("unmarshal proxies error: %v", err)
	}

	// proxies := viper.GetStringSlice("proxies")
	timeout = viper.GetInt("basic.timeout")
	dest = viper.GetString("basic.dest")
	interval = viper.GetInt("basic.interval")

	logrus.Infof("proxies: %v", ps)
	logrus.Infof("timeout: %d ms", timeout)
	logrus.Infof("dest: %s", dest)
	logrus.Infof("interval: %d s", interval)

	apiServer = viper.GetString("email.apiServer")
	receiver = viper.GetString("email.receiver")
	authKey = viper.GetString("email.auth_key")

	logrus.Infof("email.apiServer: %s", apiServer)
	logrus.Infof("email.receiver: %s", receiver)
	logrus.Infof("email.authKey: %s", authKey)

	for _, proxy := range ps {
		parsedProxy, err := url.Parse(proxy.Url)
		if err != nil {
			logrus.WithFields(logrus.Fields{"proxy": proxy, "err": err.Error()}).Fatalf("parse proxy url")
		}

		logrus.Debugf("parsedProxy: %v", parsedProxy)
		client := &http.Client{
			Transport: &http.Transport{Proxy: http.ProxyURL(parsedProxy)},
			Timeout:   time.Millisecond * time.Duration(timeout),
		}

		// init status is OK
		proxiesStatus[proxy.Url] = OK

		go check(client, proxy)
	}
	select {}
}

func check(client *http.Client, proxy *proxySetting) {
	var result int
	for {
		logrus.Debugf("client.Get(%s)", dest)
	INNER:
		for i := 1; i < viper.GetInt("basic.retry"); i++ {
			_, err := client.Get(dest)
			if err != nil {
				logrus.Debugf("retry %d error: %s", i, err.Error())
				result = Error
				time.Sleep(time.Duration(viper.GetInt("basic.retrydelay")) * time.Second)
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

func notify(proxy *proxySetting, status int) {
	var subject string
	if status == OK {
		subject += " [ Work ] "
	} else {
		subject += " [ Not work ] "
	}
	subject += proxy.Name
	logrus.Infof(subject)

	//send mail on when status changs
	if status != proxiesStatus[proxy.Name] {
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
	proxiesStatus[proxy.Name] = status
}
