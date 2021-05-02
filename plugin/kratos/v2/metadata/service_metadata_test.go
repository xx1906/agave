package metadata

import (
	"testing"
)

func TestServiceMetaData(t *testing.T) {
	var data map[string]string = make(map[string]string)
	t.Logf("data %#v", ServiceMetaData(data))
	t.Logf("data %#v", ServiceMetaData(nil))
}
