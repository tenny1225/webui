package webui

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/google/uuid"
	"golang.org/x/net/websocket"
)

type webSocketListener struct {
	conn            *websocket.Conn
	requestCallback map[string]interface{}
	requestHandler  *sync.Map
}

func NewWebSocketListener(conn *websocket.Conn) *webSocketListener {
	wb := &webSocketListener{
		conn:            conn,
		requestCallback: make(map[string]interface{}),
		requestHandler:  &sync.Map{},
	}
	return wb
}
func (w *webSocketListener) putRequestHandler(k string, i interface{}) {

	typeType := reflect.TypeOf(i)
	if typeType.Kind() == reflect.Ptr {
		if k == "" {
			k = typeType.Elem().Name()
		}
		w.requestHandler.Store(k, i)
		for i := 0; i < typeType.NumMethod(); i++ {
			m := Message{}
			m.Type = "MethodBind"
			m.FuncName = []string{k, typeType.Method(i).Name}
			fmt.Println("MethodBind", m.FuncName)
			go w.send(m, nil)
		}
	}

}
func (w *webSocketListener) removeRequestHandler(k string) {
	if v, ok := w.requestHandler.Load(k); ok {
		typeType := reflect.TypeOf(v)
		for i := 0; i < typeType.NumMethod(); i++ {
			m := Message{}
			m.Type = "RemoveBind"
			m.FuncName = []string{k, typeType.Method(i).Name}
			go w.send(m, nil)
		}
		w.requestHandler.Delete(k)
	}
}
func (w *webSocketListener) listen() {
	content := ""
	for {
		var replay string
		if e := websocket.Message.Receive(w.conn, &replay); e != nil {
			fmt.Println("err", e)
			break
		}
		content += replay
		//fmt.Println(content)
		//os.WriteFile("test.jon",[]byte(content),)
		var m Message
		if e := json.Unmarshal([]byte(content), &m); e == nil {
			content = ""
			if fun, ok := w.requestCallback[m.Id]; ok {
				reflectType := reflect.TypeOf(fun)
				if reflectType.Kind() != reflect.Func {
					continue
				}
				if reflectType.NumIn() == 0 {
					valueType := reflect.ValueOf(fun)
					go valueType.Call(nil)
					continue
				}
				if reflectType.NumIn() != 1 {
					continue
				}
				obj := reflect.New(reflectType.In(0)).Elem()
				valueType := reflect.ValueOf(fun)
				is := false
				//fmt.Println("m.Data.(type)", m.Data)
				switch m.Data.(type) {
				case string:
					fmt.Println("string")
					obj.SetString(m.Data.(string))
					is = true
				case float64:
					{
						if reflectType.Kind() <= reflect.Float64 {
							is = true
							fl := m.Data.(float64)
							switch reflectType.Kind() {
							case reflect.Int:
								obj.SetInt(int64(fl))
							case reflect.Int8:
								obj.SetInt(int64(fl))
							case reflect.Int16:
								obj.SetInt(int64(fl))
							case reflect.Int32:
								obj.SetInt(int64(fl))
							case reflect.Int64:
								obj.SetInt(int64(fl))
							case reflect.Uint:
								obj.SetUint(uint64(fl))
							case reflect.Uint8:
								obj.SetUint(uint64(fl))
							case reflect.Uint16:
								obj.SetUint(uint64(fl))
							case reflect.Uint32:
								obj.SetUint(uint64(fl))
							case reflect.Uint64:
								obj.SetUint(uint64(fl))
							case reflect.Uintptr:
								obj.SetUint(uint64(fl))
							case reflect.Float32:
								obj.SetFloat(float64(fl))
							case reflect.Float64:
								obj.SetFloat(float64(fl))
							default:
								is = false
							}
						}
					}
				case map[string]interface{}:
				case map[interface{}]interface{}:
				}
				if is {
					go valueType.Call([]reflect.Value{reflect.ValueOf(m.Data)})
				}

			} else {
				if m.FuncName == nil || len(m.FuncName) != 2 {
					continue
				}
				if value, ok := w.requestHandler.Load(m.FuncName[0]); ok {

					valueType := reflect.ValueOf(value)
					typeType := reflect.TypeOf(value)
					for i := 0; i < valueType.NumMethod(); i++ {
						if typeType.Method(i).Name == m.FuncName[1] {
							if valueType.Method(i).Type().NumIn() == len(m.Params) {
								is := true
								var params []reflect.Value
								if m.Params != nil && len(m.Params) > 0 {
									params = make([]reflect.Value, 0)
									for j, v := range m.Params {
										val := reflect.ValueOf(v)
										in := valueType.Method(i).Type().In(j)
										fmt.Println(in.Kind(), val.Kind())
										if in.Kind() != val.Kind() {
											if val.Kind() <= reflect.Float64 && val.Kind() >= in.Kind() {
												newValue := reflect.New(in).Elem()
												switch in.Kind() {
												case reflect.Int:
													newValue.SetInt(int64(val.Float()))
												case reflect.Int8:
													newValue.SetInt(int64(val.Float()))
												case reflect.Int16:
													newValue.SetInt(int64(val.Float()))
												case reflect.Int32:
													newValue.SetInt(int64(val.Float()))
												case reflect.Int64:
													newValue.SetInt(int64(val.Float()))
												case reflect.Uint:
													newValue.SetUint(uint64(val.Float()))
												case reflect.Uint8:
													newValue.SetUint(uint64(val.Float()))
												case reflect.Uint16:
													newValue.SetUint(uint64(val.Float()))
												case reflect.Uint32:
													newValue.SetUint(uint64(val.Float()))
												case reflect.Uint64:
													newValue.SetUint(uint64(val.Float()))
												case reflect.Uintptr:
													newValue.SetUint(uint64(val.Float()))
												case reflect.Float32:
													newValue.SetFloat(float64(val.Float()))
												case reflect.Float64:
													newValue.SetFloat(float64(val.Float()))
												}
												params = append(params, newValue)
												continue
											}
											is = false
										} else {
											params = append(params, val)
										}

									}
								}
								if is {
									values := valueType.Method(i).Call(params)
									if values != nil && len(values) > 0 {
										if len(values) == 0 {
											m.Data = values[0].Interface()
										} else {
											result := make([]interface{}, 0)
											for _, v := range values {
												result = append(result, v.Interface())
											}
											m.Data = result
										}
										w.send(m, nil)
									}
								}
							}
							break
						}
					}

				}

			}
		} else {
			fmt.Println("e", e)
		}

	}
}
func (w *webSocketListener) eval(js string, fun interface{}) error {
	m := Message{}
	m.Type = "Javascript"
	m.Data = js
	return w.send(m, fun)
}
func (w *webSocketListener) navigation(js string, fun interface{}) error {
	m := Message{}
	m.Type = "Navigation"
	m.Data = js
	return w.send(m, fun)
}
func (w *webSocketListener) send(m Message, fun interface{}) error {
	if m.Id == "" {
		m.Id = uuid.NewString()
	}
	if fun != nil {
		w.requestCallback[m.Id] = fun
	}
	return websocket.JSON.Send(w.conn, m)
}
