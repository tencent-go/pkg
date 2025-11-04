package util

import "sync"

type LazyMap[K, V any] struct {
	m sync.Map
}

type wrapper[T any] struct {
	value      T
	initialize func() T
	once       sync.Once
}

func (w *wrapper[V]) get() V {
	if w.initialize != nil {
		w.once.Do(func() {
			w.value = w.initialize()
			w.initialize = nil
		})
	}
	return w.value
}

func (m *LazyMap[K, V]) Load(key K) (V, bool) {
	actual, loaded := m.m.Load(key)
	if !loaded {
		var zero V
		return zero, false
	}
	w := actual.(*wrapper[V])
	return w.get(), loaded
}

func (m *LazyMap[K, V]) Store(key K, value V) {
	m.m.Store(key, &wrapper[V]{
		value: value,
	})
}

func (m *LazyMap[K, V]) Delete(key K) {
	m.m.Delete(key)
}

func (m *LazyMap[K, V]) Range(f func(key K, value V) bool) {
	m.m.Range(func(key, value interface{}) bool {
		k := key.(K)
		w := value.(*wrapper[V])
		return f(k, w.get())
	})
}

func (m *LazyMap[K, V]) LoadOrStore(key K, value V) (V, bool) {
	actual, loaded := m.m.LoadOrStore(key, &wrapper[V]{
		value: value,
	})
	return actual.(*wrapper[V]).get(), loaded
}

func (m *LazyMap[K, V]) LoadOrLazyStore(key K, initialize func() V) (V, bool) {
	actual, loaded := m.m.Load(key)
	if loaded {
		w := actual.(*wrapper[V])
		return w.get(), true
	}
	w := &wrapper[V]{
		initialize: initialize,
	}
	actual, _ = m.m.LoadOrStore(key, w)
	return actual.(*wrapper[V]).get(), false
}

func (m *LazyMap[K, V]) LoadAndDelete(key K) (V, bool) {
	actual, loaded := m.m.LoadAndDelete(key)
	if !loaded {
		var zero V
		return zero, false
	}
	w := actual.(*wrapper[V])
	return w.get(), true
}

func (m *LazyMap[K, V]) Clear() {
	m.m.Clear()
}

func (m *LazyMap[K, V]) Swap(key K, value V) (V, bool) {
	previous, loaded := m.m.Swap(key, value)
	if !loaded {
		var zero V
		return zero, false
	}
	w := previous.(*wrapper[V])
	return w.get(), true
}

func (m *LazyMap[K, V]) CompareAndSwap(key K, old, new V) (swapped bool) {
	return m.m.CompareAndSwap(key, old, new)
}

func (m *LazyMap[K, V]) CompareAndDelete(key K, old V) (deleted bool) {
	return m.m.CompareAndDelete(key, old)
}
