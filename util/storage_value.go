package util

type Storage interface {
	Load(key any) (value any, ok bool)
	Store(key, value any)
	LoadOrStore(key, value any) (actual any, loaded bool)
	LoadAndDelete(key any) (value any, loaded bool)
	Delete(any)
	Swap(key, value any) (previous any, loaded bool)
	CompareAndSwap(key, old, new any) (swapped bool)
	CompareAndDelete(key, old any) (deleted bool)
	Range(func(key, value any) (shouldContinue bool))
	Clear()
}

type StorageValue[T any] interface {
	Get(storage Storage) (T, bool)
	Set(storage Storage, value T)
}

func NewStorageValue[T any](optionalKey ...any) StorageValue[T] {
	var key any
	if len(optionalKey) > 0 {
		key = optionalKey[0]
	} else {
		key = new(struct{})
	}
	return &storageValue[T]{key}
}

type storageValue[T any] struct {
	key any
}

func (c *storageValue[T]) Get(storage Storage) (T, bool) {
	res, ok := storage.Load(c.key)
	if ok {
		return res.(T), ok
	}
	return *(new(T)), false
}

func (c *storageValue[T]) Set(storage Storage, value T) {
	storage.Store(c.key, value)
}

type TypedStorage[K, V any] interface {
	Load(key K) (value V, ok bool)
	Store(key K, value V)
	LoadOrStore(key K, value V) (actual V, loaded bool)
	LoadAndDelete(key K) (value V, loaded bool)
	Delete(K)
	Swap(key K, value V) (previous V, loaded bool)
	CompareAndSwap(key K, old, new V) (swapped bool)
	CompareAndDelete(key K, old V) (deleted bool)
	Range(func(key K, value V) (shouldContinue bool))
	Clear()
}
