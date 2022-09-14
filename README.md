# GO 轻量级Web UI

以下是例子
```$xslt
package main

import (
	"github.com/tenny1225/webui"
)

type X struct {
	w webui.Window
}

func (*X) Test(a int64) int64 {
	return a * 2
}
func (x *X) Gds() {
	x.w.Eval(`document.getElementById("xz").style.color="#ff3300";`,nil)
}
func main() {
    //最后一个参数指定一个本地的文件夹
	w := webui.NewWindow("xz", 300,300,400, 300, "./html")
	w.Run(func() {
		w.Navigation("xz.html")
		w.Bind(&X{w})
	})

}
```