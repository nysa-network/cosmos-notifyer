package ctxlogger

import (
	"context"

	"github.com/sirupsen/logrus"
)

type ContextKey string
type ContextData map[string]interface{}

const (
	dataKey = ContextKey("data")
)

// WithValue inject data value inside the context
func WithValue(ctx context.Context, key string, value interface{}) context.Context {
	if data, ok := ctx.Value(dataKey).(ContextData); ok {
		data[key] = value
		return context.WithValue(ctx, dataKey, data)
	}
	return context.WithValue(ctx, dataKey, ContextData{
		key: value,
	})
}

// Logger create a logrus.NewEntry based on the StandardLogger
func Logger(ctx context.Context) *logrus.Entry {
	l := logrus.WithContext(ctx)
	if ctx == nil {
		return l
	}

	if data, ok := ctx.Value(dataKey).(ContextData); ok {
		for k, v := range data {
			l = l.WithField(k, v)
		}
	}

	return l
}
