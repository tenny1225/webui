package main

import (
	"fmt"

	"github.com/tenny1225/webui"
)

type X struct {
	w webui.Window
}

func (x *X) Gds() {
	x.w.Eval(`document.getElementById("xz").style.color`, func(str string) {
		x.w.Eval(`alert("`+str+`")`, nil)
	})
}
func main() {
	w := webui.NewWindow("xz", 300, 300, 400, 500, "./html")
	w.Run(func() {
		w.Navigation("https://s.weibo.com/weibo?q=特朗普怕了")
		w.Eval("document.documentElement.innerHTML", func(str string) {
			fmt.Println(str)
			w.Close()
		})
	})

}
