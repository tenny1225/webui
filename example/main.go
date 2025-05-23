package main

import (
	"time"

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
		w.Navigation("xz.html")
		w.BindWithName("X", &X{w})
		time.Sleep(time.Second*2)
		w.Close()
	})

}
