package handlers

import (
	"bytes"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"io"
)

type FileHandler struct {
	BaseHandler
	ossClient  *oss.Client
	bucketName string
}

func (fh *FileHandler) UploadFile(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		log.Error(err)
		return c.Status(400).JSON(fiber.Map{"error": "Failed to read file from request"})
	}

	// 每次请求时创建一个新的 OSS 客户端
	client, err := oss.New("YourEndpoint", "YourAccessKeyId", "YourAccessKeySecret")
	if err != nil {
		log.Error(err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create OSS client"})
	}

	bucket, err := client.Bucket(fh.bucketName)
	if err != nil {
		log.Error(err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to access OSS bucket"})
	}

	fileContent, err := file.Open()
	if err != nil {
		log.Error(err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to open file"})
	}

	fileBytes, err := io.ReadAll(fileContent)
	if err != nil {
		log.Error(err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to read file bytes"})
	}

	err = bucket.PutObject(file.Filename, bytes.NewReader(fileBytes))
	if err != nil {
		log.Error(err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to upload to OSS"})
	}

	return c.Status(200).JSON(fiber.Map{"message": "File uploaded successfully to OSS"})
}
