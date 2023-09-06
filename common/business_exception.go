package common

type BusinessException struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (be *BusinessException) Error() string {
	return be.Message
}
func NewBusinessException(code int, message string) *BusinessException {
	return &BusinessException{
		Code:    code,
		Message: message,
	}
}
