package bind

type BindItem struct {
	MethodName string
	StructName string
	Path       string
	call       func(args string) (string, error)
}

func (b *BindItem) GetFullName() string {
	if b.StructName == "" {
		return b.Path + "." + b.MethodName
	}
	return b.Path + "." + b.StructName + "." + b.MethodName
}

type GenerateTplData struct {
	Name string
}
