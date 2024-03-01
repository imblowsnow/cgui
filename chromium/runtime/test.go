package runtime

type Test struct {
}

func (t Test) Test(name string) string {
	return "hello " + name
}
