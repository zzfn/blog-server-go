package models

import (
	"blog-server-go/common"
	"database/sql/driver"
	"fmt"
	"strconv"
)

type SnowflakeID string

// 实现 sql.Scanner 接口，Scan 从数据库读取值到 SnowflakeID
func (s *SnowflakeID) Scan(value interface{}) error {
	bigint, ok := value.(int64)
	if !ok {
		return fmt.Errorf("Failed to scan SnowflakeID value: %v", value)
	}
	*s = SnowflakeID(strconv.FormatInt(bigint, 10))
	return nil
}

// 实现 driver.Valuer 接口，Value 返回数据库写入值
func (s SnowflakeID) Value() (driver.Value, error) {
	if len(s) == 0 {
		return nil, nil
	}

	bigint, err := strconv.ParseInt(string(s), 10, 64)
	if err != nil {
		return nil, err
	}

	return bigint, nil
}

type AutoSnowflakeID string

func (s *AutoSnowflakeID) Scan(value interface{}) error {
	bigint, ok := value.(int64)
	if !ok {
		return fmt.Errorf("Failed to scan SnowflakeID value: %v", value)
	}
	*s = AutoSnowflakeID(strconv.FormatInt(bigint, 10))
	return nil
}

// 实现 driver.Valuer 接口，Value 返回数据库写入值
func (s AutoSnowflakeID) Value() (driver.Value, error) {
	if len(s) == 0 {
		newID, _ := common.GenerateID()
		return newID, nil
	}

	bigint, err := strconv.ParseInt(string(s), 10, 64)
	if err != nil {
		return nil, err
	}

	return bigint, nil
}
func (s *AutoSnowflakeID) Set(id string) {
	*s = AutoSnowflakeID(id)
}
