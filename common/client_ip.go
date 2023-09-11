package common

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/redis/go-redis/v9"
	"io"
	"net/http"
	"os"
)

type IPInfo struct {
	XForwardedFor  string `reqHeader:"x-forwarded-for"`
	XRealIp        string `reqHeader:"x-real-ip"`
	CfConnectingIp string `reqHeader:"cf-connecting-ip"`
}

// GetConnectingIp tries to get the client's real IP address using different headers.
func GetConnectingIp(c *fiber.Ctx) string {
	ipInfo := new(IPInfo)

	if err := c.ReqHeaderParser(ipInfo); err != nil {
		return "Unknown" // or return an empty string, depending on your preference
	}

	// Return the IP from the headers, prioritizing cf-connecting-ip if available
	if ipInfo.CfConnectingIp != "" {
		return ipInfo.CfConnectingIp
	} else if ipInfo.XRealIp != "" {
		return ipInfo.XRealIp
	} else if ipInfo.XForwardedFor != "" {
		return ipInfo.XForwardedFor
	}

	// If no headers provide the IP, return the IP from the Fiber context
	return c.IP()
}

type IpAddress struct {
	City     string `json:"city"`
	Province string `json:"province"`
}

func GetIpAddressInfo(client *redis.Client, ip string) (string, error) {
	var ctx = context.Background()
	exists, err := client.HExists(ctx, "address", ip).Result()
	if exists {
		// 如果存在，则从Redis中获取地址信息
		address, err := client.HGet(ctx, "address", ip).Result()
		if err == nil {
			return address, nil // 从Redis中直接返回信息
		}
	}
	AmapApiKey := os.Getenv("AMAP_API_KEY")
	url := fmt.Sprintf("https://restapi.amap.com/v3/ip?key=%s&ip=%s", AmapApiKey, ip)
	log.Info("url1:", url)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Error: Received HTTP %d", resp.StatusCode)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var info IpAddress
	err = json.Unmarshal(bodyBytes, &info)
	if err != nil {
		return "", err
	}
	if info.City != "" && info.Province != "" {
		address := fmt.Sprintf("%s, %s", info.City, info.Province)
		err = client.HSet(ctx, "address", ip, address).Err()
		if err != nil {
			return "未知地址", fmt.Errorf("error storing to Redis: %w", err)
		}
		return address, nil
	} else {
		apiKey := os.Getenv("IP_INFO_KEY")
		url2 := fmt.Sprintf("https://ipinfo.io/%s/json?token=%s", ip, apiKey)
		log.Info("url2:", url2)
		resp, err = http.Get(url2)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		var infoIpinfo struct {
			City    string `json:"city"`
			Region  string `json:"region"`
			Country string `json:"country"`
		}
		err = json.Unmarshal(bodyBytes, &infoIpinfo)
		if err != nil {
			return "", err
		}
		address := fmt.Sprintf("%s, %s, %s", infoIpinfo.City, infoIpinfo.Region, infoIpinfo.Country)
		err = client.HSet(ctx, "address", ip, address).Err()
		return address, nil
	}
}
