package env

import (
	"os"
	"testing"
)

func TestBaseConfig(t *testing.T) {
	const podName = "pod"
	_ = os.Setenv("POD_NAME", podName)
	c := BaseConfigReaderBuilder.Build().Read()
	t.Run("常規驗證", func(t *testing.T) {
		if c.PodName != podName {
			t.Error("unexpected podName")
		}
	})
}
