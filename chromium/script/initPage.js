(function (){
    console.log("initPage.js loaded!");
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
    }
})()
