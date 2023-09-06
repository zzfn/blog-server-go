package common

import (
	"fmt"
	"github.com/gofiber/fiber/v2/log"
)

type InitializationError struct {
	Module string
	Err    error
}

func (e *InitializationError) Error() string {
	return fmt.Sprintf("failed to initialize %s: %s", e.Module, e.Err)
}

func HandleError(err error, message string) {
	if err != nil {
		log.Error("%s: %s", message, err)
	}
}
