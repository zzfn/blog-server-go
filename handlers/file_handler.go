package handlers

import (
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"os"
)

type FileHandler struct {
	BaseHandler
}

func (fh *FileHandler) UploadFile(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	path := c.FormValue("path")
	if err != nil {
		log.Error(err)
		return c.Status(400).JSON(fiber.Map{"error": "Failed to read file from request"})
	}
	OssEndpoint := os.Getenv("OSS_ENDPOINT")
	OssAccessKeyId := os.Getenv("OSS_ACCESS_KEY_ID")
	OssAccessKeySecret := os.Getenv("OSS_ACCESS_KEY_SECRET")

	// 每次请求时创建一个新的 OSS 客户端
	client, err := oss.New(OssEndpoint, OssAccessKeyId, OssAccessKeySecret, oss.UseCname(true))
	if err != nil {
		log.Error(err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create OSS client"})
	}

	bucket, err := client.Bucket("wwma")
	if err != nil {
		log.Error(err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to access OSS bucket"})
	}

	fileContent, err := file.Open()
	if err != nil {
		log.Error(err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to open file"})
	}
	objectKey := path + file.Filename
	err = bucket.PutObject(objectKey, fileContent)

	if err != nil {
		log.Error(err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to upload to OSS"})
	}

	return c.Status(200).JSON([]string{OssEndpoint + objectKey})
}
