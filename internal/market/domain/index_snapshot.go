package domain

import (
	"errors"
	"strings"
)

// IndexSnapshot 是某个交易日的指数收盘快照。
type IndexSnapshot struct {
	Code          string
	Name          string
	TradeDate     string
	ObservedAt    string
	Close         string
	Change        string
	ChangePercent string
}

// Validate 检查指数快照的最小领域约束。
func (snapshot IndexSnapshot) Validate() error {
	for name, value := range map[string]string{
		"code": snapshot.Code, "name": snapshot.Name, "trade date": snapshot.TradeDate,
		"observed at": snapshot.ObservedAt, "close": snapshot.Close, "change": snapshot.Change,
		"change percent": snapshot.ChangePercent,
	} {
		if strings.TrimSpace(value) == "" {
			return errors.New("index snapshot " + name + " is required")
		}
	}
	return nil
}
