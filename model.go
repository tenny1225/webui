package webui


type Message struct {
	Type string
	FuncName []string
	Params []interface{}
	Id string
	Data interface{}
}
