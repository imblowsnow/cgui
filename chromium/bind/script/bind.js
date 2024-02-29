(function (){
    window._cgui_runtime = window._cgui_runtime || {}
    window._cgui = window._cgui || {
        callbacks: new Map(),
        deliverError: function (name, seq, message) {
            const error = new Error(message);
            _cgui.callbacks.get(seq).reject(error);
            _cgui.callbacks.delete(seq);
        },
        deliverResult: function (name, seq, result) {
            _cgui.callbacks.get(seq).resolve(result);
            _cgui.callbacks.delete(seq);
        },
        wrapBinding(type, name, originName){
            _cgui_runtime[originName] = function (args){
                if (typeof args != 'string') {
                    return Promise.reject(
                        new Error(
                            'function takes exactly one argument, this argument should be string'
                        )
                    );
                }

                const seq = Date.now() + Math.random().toString(36).substr(2);

                window[name](JSON.stringify({ type, seq, args }));

                return new Promise((resolve, reject) => {
                    _cgui.callbacks.set(seq, { resolve, reject });
                });
            };

        }
    }
})()
