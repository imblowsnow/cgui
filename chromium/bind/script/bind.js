(function (){
    window._cgui = window._cgui || {}
    window._cexposed = window._cexposed || {
        callbacks: new Map(),
        deliverError: function (name, seq, message) {
            const error = new Error(message);
            _cexposed.callbacks.get(seq).reject(error);
            _cexposed.callbacks.delete(seq);
        },
        deliverResult: function (name, seq, result) {
            _cexposed.callbacks.get(seq).resolve(result);
            _cexposed.callbacks.delete(seq);
        },
        wrapBinding(type, name, originName){
            this.setNestedProperty(window._cgui, originName, function (args){
                if (typeof args != 'string') {
                    args = JSON.stringify(args);
                }

                const seq = Date.now() + Math.random().toString(36).substr(2);

                window[name](JSON.stringify({ type, seq, args }));

                return new Promise((resolve, reject) => {
                    _cexposed.callbacks.set(seq, { resolve, reject });
                });
            });
        },
        setNestedProperty(obj, path, value) {
            const pathParts = path.split('.');
            const lastPart = pathParts.pop();

            let currentPart = obj;
            for (const part of pathParts) {
                if (!(part in currentPart)) {
                    currentPart[part] = {};
                }
                currentPart = currentPart[part];
            }

            currentPart[lastPart] = value;
        }
    }
})()
