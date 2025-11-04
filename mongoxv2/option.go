package mongox

import (
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UpdateOptions struct {
	IgnoreZeroValue *bool
	OptimisticLock  *bool
	*options.UpdateOptions
}

func (o *UpdateOptions) SetIgnoreZeroValue(b bool) *UpdateOptions {
	o.IgnoreZeroValue = &b
	return o
}

func (o *UpdateOptions) SetOptimisticLock(b bool) *UpdateOptions {
	o.OptimisticLock = &b
	return o
}

func Update() *UpdateOptions {
	return &UpdateOptions{
		UpdateOptions: options.Update(),
	}
}

type FindOneAndUpdateOptions struct {
	IgnoreZeroValue *bool
	*options.FindOneAndUpdateOptions
}

func (o *FindOneAndUpdateOptions) SetIgnoreZeroValue(b bool) *FindOneAndUpdateOptions {
	o.IgnoreZeroValue = &b
	return o
}

func FindOneAndUpdate() *FindOneAndUpdateOptions {
	return &FindOneAndUpdateOptions{
		FindOneAndUpdateOptions: options.FindOneAndUpdate(),
	}
}

type ChangeStreamOptions struct {
	*options.ChangeStreamOptions
	ConsumerName  *string
	PersistCursor *bool
}

func (o *ChangeStreamOptions) SetConsumerName(s string) *ChangeStreamOptions {
	o.ConsumerName = &s
	return o
}

func (o *ChangeStreamOptions) SetPersistCursor(b bool) *ChangeStreamOptions {
	o.PersistCursor = &b
	return o
}

func ChangeStream() *ChangeStreamOptions {
	return &ChangeStreamOptions{
		ChangeStreamOptions: options.ChangeStream(),
	}
}
