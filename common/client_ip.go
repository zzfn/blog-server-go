package common

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/redis/go-redis/v9"
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
	ipInfo := &IPInfo{}
	if err := c.ReqHeaderParser(ipInfo); err != nil {
		return "Unknown"
	}
	if ip := ipInfo.CfConnectingIp; ip != "" {
		return ip
	}

	return "Unknown"
}

type IpAddress struct {
	City     string `json:"city"`
	Province string `json:"province"`
}

type IpInfoResponse struct {
	City    string `json:"city"`
	Region  string `json:"region"`
	Country string `json:"country"`
	Org     string `json:"org"`
}

func fetchFromAmap(ip, apiKey string) (string, error) {
	url := fmt.Sprintf("https://restapi.amap.com/v3/ip?key=%s&ip=%s", apiKey, ip)
	log.Info("Fetching from Amap", url)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Amap Error: Received HTTP %d", resp.StatusCode)
	}

	var info IpAddress
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", err
	}

	if info.City != "" && info.Province != "" {
		return fmt.Sprintf("%s, %s", info.City, info.Province), nil
	}
	return "", fmt.Errorf("Incomplete data from Amap")
}

func fetchFromIpinfo(ip, apiKey string) (string, error) {
	url := fmt.Sprintf("https://ipinfo.io/%s/json?token=%s", ip, apiKey)
	log.Info("Fetching from Ipinfo: ", url)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var info IpInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", err
	}
	if info.City != "" && info.Region != "" && info.Country != "" {
		return fmt.Sprintf("%s, %s, %s", info.City, info.Region, info.Country), nil
	} else {
		return "", fmt.Errorf("Incomplete data from Ipinfo")
	}
}

func GetIpAddressInfo(client *redis.Client, ip string) (string, error) {
	var ctx = context.Background()
	address, err := client.HGet(ctx, "address", ip).Result()
	if err == nil {
		return address, nil
	}

	AmapApiKey := os.Getenv("AMAP_API_KEY")
	address, err = fetchFromAmap(ip, AmapApiKey)
	if err == nil {
		client.HSet(ctx, "address", ip, address)
		return address, nil
	}

	IpinfoApiKey := os.Getenv("IP_INFO_KEY")
	address, err = fetchFromIpinfo(ip, IpinfoApiKey)
	if err == nil {
		client.HSet(ctx, "address", ip, address)
		return address, nil
	}

	return "未知地址", err
}
