(function (){
    const mode = "{mode}";

    console.log("inject common.js success!  mode=", mode, location.href);

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
    }
})()
