package main

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Token struct {
	Key string
	Created time.Time
}

type RetentionMap map[string]Token

func NewRetentionMap(ctx context.Context, retentionPeriod time.Duration) RetentionMap {
	rm := make(RetentionMap)

	go rm.Retention(ctx, retentionPeriod)

	return rm
}

func (rm RetentionMap) NewToken() Token {
	token := Token{
		Key: uuid.NewString(),
		Created: time.Now(),
	}

	rm[token.Key] = token
	return token
}

func (rm RetentionMap) VerifyToken(token string) bool {
	if _, ok := rm[token]; !ok {
		return false
	}
	delete(rm, token)
	return true
}

func (rm RetentionMap) Retention(ctx context.Context, retentionPeriod time.Duration) {
	ticker := time.NewTicker(400 * time.Millisecond)

	for {
		select {
		case <- ticker.C:
			for _, token := range rm {
				if (token.Created.Add(retentionPeriod).Before(time.Now())) {
					delete(rm, token.Key)
				}
			}
		case <- ctx.Done():
			return
		}
	}
}