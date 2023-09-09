package common

import (
	"github.com/gofiber/fiber/v2/log"
	"strconv"
)

func GenerateID() (string, error) {
	node, err := NewNode(1)
	if err != nil {
		log.Errorf("Failed to create snowflake node: %v", err)
		return "", err
	}
	return strconv.FormatInt(node.Generate().Int64(), 10), nil
}
