console.log("go.js loaded!");
const mode = "{mode}";

if (mode !== "dev") {
    // 拦截右击菜单
    document.oncontextmenu = function (event) {
        event.preventDefault()
    }

    // 拦截F12弹窗
    document.onkeydown = function (event) {
        if (event.keyCode === 123) {
            event.preventDefault()
        }
    }
}else{
    // 检测后台服务是否启动
    setInterval(() => {
        fetch("http://127.0.0.1/sub-jstogo", {
            method: "POST",
            headers: {
                "Content-Type": "application/json"
            },
            body: JSON.stringify({
                call: "status"
            })
        }).then((res) => {
            if (res.status === 200) {
            }else{
                window.close()
            }
        }).catch((err) => {
            window.close()
        })
    }, 1000);
}
