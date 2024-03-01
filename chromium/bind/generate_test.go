package bind

import (
	"github.com/imblowsnow/cgui/chromium/runtime"
	"testing"
)

func TestGenerate(t *testing.T) {
	Generate("frontend", []interface{}{
		runtime.WebviewTest{},
	})
}
