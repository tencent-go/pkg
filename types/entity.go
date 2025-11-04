package types

import (
	"github.com/sirupsen/logrus"
	"time"
)

type Aggregate interface {
	GetID() ID
	SetID(ID)
	GetVersion() int64
	SetVersion(int64)
	GetCreatedAt() int64
	SetCreatedAt(int64)
	GetUpdatedAt() int64
	SetUpdatedAt(int64)
}

type Entity struct {
	ID        ID    `json:"id" bson:"_id"`
	CreatedAt int64 `json:"createdAt" bson:"createdAt"`
	UpdatedAt int64 `json:"updatedAt" bson:"updatedAt"`
	Version   int64 `json:"version" bson:"version"`
}

func NewEntity() Entity {
	return Entity{
		ID:        NewID(),
		CreatedAt: time.Now().UnixMilli(),
		UpdatedAt: time.Now().UnixMilli(),
		Version:   1,
	}
}

func (b *Entity) GetCreatedAt() int64 {
	return b.CreatedAt
}

func (b *Entity) GetUpdatedAt() int64 {
	return b.UpdatedAt
}

func (b *Entity) SetCreatedAt(timestamp int64) {
	switch {
	case timestamp >= 1e12 && timestamp < 1e13:
		// 13 位的毫秒级时间戳
		b.CreatedAt = timestamp
	case timestamp >= 1e9 && timestamp < 1e10:
		// 10 位的秒级时间戳，转换为毫秒
		b.CreatedAt = timestamp * 1000
	default:
		// 不符合预期的长度
		logrus.Panicf("invalid timestamp: %d", timestamp)
	}
}

func (b *Entity) SetUpdatedAt(timestamp int64) {
	switch {
	case timestamp >= 1e12 && timestamp < 1e13:
		// 13 位的毫秒级时间戳
		b.UpdatedAt = timestamp
	case timestamp >= 1e9 && timestamp < 1e10:
		// 10 位的秒级时间戳，转换为毫秒
		b.UpdatedAt = timestamp * 1000
	default:
		// 不符合预期的长度
		logrus.Panicf("invalid timestamp: %d", timestamp)
	}
}

func (b *Entity) GetID() ID {
	return b.ID
}

func (b *Entity) SetID(id ID) {
	b.ID = id
}

func (b *Entity) SetVersion(v int64) {
	b.Version = v
}

func (b *Entity) GetVersion() int64 {
	return b.Version
}
