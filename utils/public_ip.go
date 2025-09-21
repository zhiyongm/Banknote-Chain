package utils

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"time"
)

func GetPublicIP() string {
	var ip string
	for {
		// 发送 GET 请求到 api.ipify.org
		resp, err := http.Get("https://ipinfo.io/ip")
		if err != nil {
			log.Fatalf("无法发送请求: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}
		// 确保在函数结束时关闭响应体
		defer resp.Body.Close()

		// 检查 HTTP 状态码
		if resp.StatusCode != http.StatusOK {
			log.Fatalf("收到错误的 HTTP 状态码: %s", resp.Status)
			time.Sleep(2 * time.Second)
			continue
		}

		// 读取响应体内容
		ip_bytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("无法读取响应内容: %v", err)
			time.Sleep(2 * time.Second)
			continue

		}
		ip = string(ip_bytes)
		break
		// 打印公网 IP
	}

	fmt.Printf("你的公网 IP 地址是: %s\n", ip)

	return ip
}
