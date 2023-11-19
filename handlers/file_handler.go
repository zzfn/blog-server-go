package handlers

import (
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"net/url"
	"os"
	"path"
	"strings"
)

type FileHandler struct {
	BaseHandler
}

func (fh *FileHandler) UploadFile(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	objectPath := c.FormValue("objectPath")
	if err != nil {
		log.Error(err)
		return c.Status(400).JSON(fiber.Map{"error": "Failed to read file from request"})
	}
	OssEndpoint := os.Getenv("OSS_ENDPOINT")
	OssAccessKeyId := os.Getenv("OSS_ACCESS_KEY_ID")
	OssAccessKeySecret := os.Getenv("OSS_ACCESS_KEY_SECRET")
	OssBucket := os.Getenv("OSS_BUCKET")

	// 每次请求时创建一个新的 OSS 客户端
	client, err := oss.New(OssEndpoint, OssAccessKeyId, OssAccessKeySecret, oss.UseCname(true))
	if err != nil {
		log.Error(err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create OSS client"})
	}

	bucket, err := client.Bucket(OssBucket)
	if err != nil {
		log.Error(err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to access OSS bucket"})
	}

	fileContent, err := file.Open()
	if err != nil {
		log.Error(err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to open file"})
	}
	defer fileContent.Close()
	objectKey := path.Join(objectPath, file.Filename)
	err = bucket.PutObject(objectKey, fileContent)

	if err != nil {
		log.Error(err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to upload to OSS"})
	}
	u, err := url.Parse(OssEndpoint)
	if err != nil {
		log.Error(err)
	}
	trimmedObjectKey := strings.TrimPrefix(objectKey, "/")
	u.Path = path.Join(u.Path, trimmedObjectKey)
	finalUrl := u.String()
	return c.Status(200).JSON([]string{finalUrl})
}
func (fh *FileHandler) ListFile(c *fiber.Ctx) error {
	OssEndpoint := os.Getenv("OSS_ENDPOINT")
	OssAccessKeyId := os.Getenv("OSS_ACCESS_KEY_ID")
	OssAccessKeySecret := os.Getenv("OSS_ACCESS_KEY_SECRET")
	OssBucket := os.Getenv("OSS_BUCKET")

	// 每次请求时创建一个新的 OSS 客户端
	client, err := oss.New(OssEndpoint, OssAccessKeyId, OssAccessKeySecret, oss.UseCname(true))
	if err != nil {
		log.Error(err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to create OSS client"})
	}

	bucket, err := client.Bucket(OssBucket)
	if err != nil {
		log.Error(err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to access OSS bucket"})
	}
	prefix := c.Query("prefix")
	marker := c.Query("marker")
	delimiter := c.Query("delimiter")
	// 列举文件。
	lsRes, err := bucket.ListObjects(oss.Marker(marker), oss.Delimiter(delimiter), oss.Prefix(prefix))
	if err != nil {
		log.Error(111, err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to list objects"})
	}
	log.Info("lsRes", lsRes)
	return c.Status(200).JSON(lsRes)
}
