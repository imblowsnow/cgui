package chromium

import (
	"fmt"
	"testing"
)

type TestBindJs struct {
}

func (TestBindJs) Test1(params string) {
	fmt.Println("Test1", params)
}
func (TestBindJs) Test2(params string) {
	fmt.Println("Test2", params)
}

func TestGenerateBindJs(t *testing.T) {
	var binds []interface{}
	binds = append(binds, TestBindJs{})
	GenerateBindJs(binds)
}
