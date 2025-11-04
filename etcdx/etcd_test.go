package etcdx

import (
	"fmt"
	"os"
	"testing"

	"github.com/tencent-go/pkg/env"
)

func TestGetClientUniqueLease(t *testing.T) {
	_ = os.Setenv("ETCD_ENDPOINTS", "ksafjkd://sakdjf.skdjfkas,http://lsdkjfas.sldkfj:333,jjjjj")
	env.PrintState()
	fmt.Print(configReader.Read().Endpoints)
}
