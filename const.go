package webui


const DEFAULT_HTML=`
<html>
<title>%s</title>
<body style="margin:0;padding:0;background-color:#ffffff;">
<iframe src="%s" style="border:medium none;width:100vw;height:100vh;" frameborder="0" id="iframe" style="margin:0;padding:0;"></iframe>
</body>
</html>
<script>
    var wsServer = 'ws://127.0.0.1:%s/ws';
    var websocket;
    var requestMap = {};
    var iframe;
    window.onload = function () {
        iframe = document.getElementById("iframe");
        startWebsocket();
    }
    function startWebsocket() {
        websocket = new WebSocket(wsServer);
        websocket.onopen = function (evt) {
            console.log(evt)
        };
        websocket.onclose = function (evt) {
            setTimeout(() => {
                startWebsocket();
            }, 1000)
        };
        websocket.onmessage = function (evt) {
            let msg = JSON.parse(evt.data);
            console.log(msg)
            if (msg.Type == "Navigation") {
                let r = eval(msg.Data);
                setTimeout(() => {
                    websocket.send2({Id: msg.Id, Data: r});
                }, 1000)

            } else if (msg.Type == "Javascript") {
                let r = iframe.contentWindow.eval(msg.Data);
                websocket.send2({Id: msg.Id, Data: r});
            } else if (msg.Type == "MethodBind") {
                console.log("MethodBind", msg.FuncName)
                let fs = msg.FuncName;
                if (fs.length != 2) {
                    return;
                }
                iframe.contentWindow[fs[0]] = {};
                iframe.contentWindow[fs[0]][fs[1]] = function () {
                    let data = {FuncName: fs, Params: []};
                    let callback = null;
                    if (arguments.length > 0) {
                        for (let k in arguments) {
                            if (k == arguments.length - 1 && typeof arguments[k] === "function") {
                                callback = arguments[k];
                            } else {
                                data.Params.push(arguments[k])
                            }

                        }
                    }
                    websocket.send2(data, callback)
                }
            } else if (msg.Type == "RemoveBind") {
                let fs = msg.FuncName;
                if (fs.length != 2) {
                    return;
                }
                iframe.contentWindow[fs[0]] = {};
            } else if (requestMap[msg.Id]) {
                requestMap[msg.Id](msg.Data)
            }
        };

        websocket.onerror = function (evt, e) {
            console.log('Error occured: ' + evt.data);
        };
        websocket.send2 = function (data, f) {
            console.log("send2", data)
            if (!data.Id) data.Id = Date.now() + "";
            if (f) requestMap[data.Id] = f;
            websocket.send(JSON.stringify(data));
        }
    }

</script>
`
const NOT_FOUND_CHROME_HTML  = `
<h1>Chrome not found on this computer,please install it first!</h1>
`
const DEFAULT_HTML_NAME = "DEFAULT.html"
const NOT_FOUND_CHROME_HTML_NAME = "NOT_FOUND_CHROME.html"
