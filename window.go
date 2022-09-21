package webui

import (
	"bytes"
	"fmt"
	"golang.org/x/net/websocket"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

type Window interface {
	HandleFunc(p string, f func(w http.ResponseWriter, r *http.Request))
	Run(fun func())
	RunAndBindPort(port string, fun func())
	Close()
	Navigation(htmlPath string)
	BindWithName(key string, obj interface{})
	Bind(obj interface{})
	RemoveBind(key string)
	Eval(js string, f interface{})
}

func NewWindow(title string, x, y, w, h int64, staticPath string) Window {
	return &window{
		title:         title,
		w:             w,
		h:             h,
		x:             x,
		y:             y,
		staticPath:    staticPath,
		closeChannel:  make(chan int, 1),
		wlWaitChannel: make(chan int, 1),
	}
}

type window struct {
	title         string
	x             int64
	y             int64
	w             int64
	h             int64
	staticPath    string
	closeChannel  chan int
	currentUrl    string
	addr          string
	inited        bool
	wl            *webSocketListener
	wlWaitChannel chan int
}

func (w *window) HandleFunc(p string, f func(w http.ResponseWriter, r *http.Request)) {
	http.HandleFunc(p, f)
}
func (w *window) Run(fun func()) {
	w.RunAndBindPort("8000", fun)
}
func (w *window) RunAndBindPort(addr string, fun func()) {
	if strings.HasPrefix(addr, ":") {
		addr = addr[1:]
	}
	w.addr = addr
	go w.startHttpService(addr)
	<-time.Tick(time.Second)
	if fun!=nil{
		go fun()
	}
	<-w.closeChannel
}
func (win *window) startHttpService(addr string) {
	http.Handle("/ws", websocket.Handler(func(wc *websocket.Conn) {
		win.wl = NewWebSocketListener(wc)
		win.wlWaitChannel <- 1
		win.wl.listen()
	}))
	http.HandleFunc("/file", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		p := r.FormValue("p")
		fn := path.Base(p)
		f, e := os.Open(p)
		if e != nil {
			w.WriteHeader(500)
			w.Write([]byte(e.Error()))
			return
		}
		defer f.Close()
		w.WriteHeader(200)
		w.Header().Add("Content-Disposition", "attachment;filename="+fn)
		io.Copy(w, f)
	})
	http.HandleFunc("/html/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		f := path.Base(r.URL.Path)
		var buf []byte
		if f == DEFAULT_HTML_NAME {
			html := fmt.Sprintf(DEFAULT_HTML,
				win.title,
				win.currentUrl,
				win.addr)
			buf = []byte(html)
		} else if f == NOT_FOUND_CHROME_HTML_NAME {
			buf = []byte(NOT_FOUND_CHROME_HTML)
		} else {
			var e error
			fn := path.Join(win.staticPath, f)
			buf, e = os.ReadFile(fn)
			if e != nil {
				w.Write([]byte(e.Error()))
				return
			}
		}

		w.Write(buf)
	})
	//http.HandleFunc("/remote", func(w http.ResponseWriter, r *http.Request) {
	//	r.ParseForm()
	//	uri := r.FormValue("url")
	//	rep,e:=http.Get(uri)
	//
	//	if e!=nil{
	//		w.Write([]byte(e.Error()))
	//		return
	//	}
	//	doc,e:=goquery.NewDocumentFromReader(rep.Body)
	//	if e!=nil{
	//		w.Write([]byte(e.Error()))
	//		return
	//	}
	//	URL,e:=url.Parse(uri)
	//	if e!=nil{
	//		w.Write([]byte(e.Error()))
	//		return
	//	}
	//
	//	node:=doc.Find("link")
	//	for i:=0;i<node.Length();i++{
	//		for n,attr:=range node.Get(i).Attr{
	//			if attr.Key=="href"&&!strings.HasPrefix(attr.Val,"http://")&&!strings.HasPrefix(attr.Val,"https://"){
	//
	//				attr.Val=URL.Scheme+"://"+URL.Host+attr.Val
	//
	//				node.Get(i).Attr[n]=attr
	//			}
	//		}
	//
	//	}
	//	node=doc.Find("script")
	//	for i:=0;i<node.Length();i++{
	//		for n,attr:=range node.Get(i).Attr{
	//			if attr.Key=="src"&&!strings.HasPrefix(attr.Val,"http://")&&!strings.HasPrefix(attr.Val,"https://"){
	//				attr.Val=URL.Scheme+"://"+URL.Host+attr.Val
	//				node.Get(i).Attr[n]=attr
	//			}
	//		}
	//
	//	}
	//
	//	node=doc.Find("img")
	//	for i:=0;i<node.Length();i++{
	//		for n,attr:=range node.Get(i).Attr{
	//			if attr.Key=="src"&&!strings.HasPrefix(attr.Val,"http://")&&!strings.HasPrefix(attr.Val,"https://"){
	//				attr.Val=URL.Scheme+"://"+URL.Host+attr.Val
	//				node.Get(i).Attr[n]=attr
	//			}
	//		}
	//
	//	}
	//
	//	node=doc.Find("a")
	//	for i:=0;i<node.Length();i++{
	//		for n,attr:=range node.Get(i).Attr{
	//			if attr.Key=="href"&&!strings.HasPrefix(attr.Val,"http://")&&!strings.HasPrefix(attr.Val,"https://"){
	//				attr.Val=URL.Scheme+"://"+URL.Host+attr.Val
	//				node.Get(i).Attr[n]=attr
	//			}
	//		}
	//
	//	}
	//	html,e:=doc.Html()
	//	fmt.Println(html)
	//	if e!=nil{
	//		w.Write([]byte(e.Error()))
	//		return
	//	}
	//	w.Write([]byte(html))
	//})
	http.ListenAndServe(":"+addr, nil)
}
func (w *window) startChrome(url string) {
	commandReader := bytes.NewBuffer([]byte{})
	go func() {
		defer w.Close()
		bash, args := GetLocalChromeBash(w.x, w.y, w.w, w.h, url)
		if bash == "" {
			openDefaultWebView(fmt.Sprintf("http://localhost:%s/html/%s", w.addr, NOT_FOUND_CHROME_HTML_NAME))
			time.Sleep(time.Second * 5)
			return
		}
		cmd := exec.Command(bash, args...)
		cmd.Stdout = commandReader
		cmd.Start()
		cmd.Wait()
	}()
	commandReader.Next(10)
}
func (w *window) Close() {
	w.closeChannel <- 1
}

func (w *window) Navigation(htmlPath string) {
	initChannel := make(chan int, 1)
	go func() {
		if !w.inited {
			w.inited = true
			w.startChrome(fmt.Sprintf("http://localhost:%s/html/%s", w.addr, DEFAULT_HTML_NAME))
		}
		if strings.HasPrefix(htmlPath,"http://")||strings.HasPrefix(htmlPath,"https://"){
			w.currentUrl =  htmlPath
		}else{
			w.currentUrl = fmt.Sprintf("http://localhost:%s/html/%s", w.addr, htmlPath)
		}
		if w.wl == nil {
			<-w.wlWaitChannel
		}
		w.wl.navigation(`document.getElementById("iframe").src="`+w.currentUrl+`"`, func() {
			initChannel <- 1
		})

	}()
	<-initChannel
}

func (s *window) BindWithName(key string, obj interface{}) {
	if s.wl != nil {
		s.wl.putRequestHandler(key, obj)
	}
}
func (s *window) Bind(obj interface{}) {
	if s.wl != nil {
		s.wl.putRequestHandler("", obj)
	}
}
func (s *window) RemoveBind(key string) {
	if s.wl != nil {
		s.wl.removeRequestHandler(key)
	}
}

func (s *window) Eval(js string, f interface{}) {
	s.wl.eval(js, f)
}
