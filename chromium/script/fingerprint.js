(function (){

    var utils = {};
    utils.makeNativeString = (name = '') => {
        return (Function.toString + '').replace('toString', name || '')
    }
    utils.patchToString = (obj, str = '') => {
        const handler = {
            apply: function (target, ctx) {
                // This fixes e.g. `HTMLMediaElement.prototype.canPlayType.toString + ""`
                if (ctx === Function.prototype.toString) {
                    return utils.makeNativeString('toString')
                }
                // `toString` targeted at our proxied Object detected
                if (ctx === obj) {
                    // We either return the optional string verbatim or derive the most desired result automatically
                    return str || utils.makeNativeString(obj.name)
                }
                // Check if the toString protype of the context is the same as the global prototype,
                // if not indicates that we are doing a check across different windows., e.g. the iframeWithdirect` test case
                const hasSameProto = Object.getPrototypeOf(
                    Function.prototype.toString
                ).isPrototypeOf(ctx.toString) // eslint-disable-line no-prototype-builtins
                if (!hasSameProto) {
                    // Pass the call on to the local Function.prototype.toString instead
                    return ctx.toString()
                }
                return target.call(ctx)
            }
        }

        const toStringProxy = new Proxy(
            Function.prototype.toString,
            utils.stripProxyFromErrors(handler)
        )
        utils.replaceProperty(Function.prototype, 'toString', {
            value: toStringProxy
        })
    }
    utils.stripProxyFromErrors = (handler = {}) => {
        const newHandler = {
            setPrototypeOf: function (target, proto) {
                if (proto === null)
                    throw new TypeError('Cannot convert object to primitive value')
                if (Object.getPrototypeOf(target) === Object.getPrototypeOf(proto)) {
                    throw new TypeError('Cyclic __proto__ value')
                }
                return Reflect.setPrototypeOf(target, proto)
            }
        }
        // We wrap each trap in the handler in a try/catch and modify the error stack if they throw
        const traps = Object.getOwnPropertyNames(handler)
        traps.forEach(trap => {
            newHandler[trap] = function () {
                try {
                    // Forward the call to the defined proxy handler
                    return handler[trap].apply(this, arguments || [])
                } catch (err) {
                    // Stack traces differ per browser, we only support chromium based ones currently
                    if (!err || !err.stack || !err.stack.includes(`at `)) {
                        throw err
                    }

                    // When something throws within one of our traps the Proxy will show up in error stacks
                    // An earlier implementation of this code would simply strip lines with a blacklist,
                    // but it makes sense to be more surgical here and only remove lines related to our Proxy.
                    // We try to use a known "anchor" line for that and strip it with everything above it.
                    // If the anchor line cannot be found for some reason we fall back to our blacklist approach.

                    const stripWithBlacklist = (stack, stripFirstLine = true) => {
                        const blacklist = [
                            `at Reflect.${trap} `, // e.g. Reflect.get or Reflect.apply
                            `at Object.${trap} `, // e.g. Object.get or Object.apply
                            `at Object.newHandler.<computed> [as ${trap}] ` // caused by this very wrapper :-)
                        ]
                        return (
                            err.stack
                                .split('\n')
                                // Always remove the first (file) line in the stack (guaranteed to be our proxy)
                                .filter((line, index) => !(index === 1 && stripFirstLine))
                                // Check if the line starts with one of our blacklisted strings
                                .filter(line => !blacklist.some(bl => line.trim().startsWith(bl)))
                                .join('\n')
                        )
                    }

                    const stripWithAnchor = (stack, anchor) => {
                        const stackArr = stack.split('\n')
                        anchor = anchor || `at Object.newHandler.<computed> [as ${trap}] ` // Known first Proxy line in chromium
                        const anchorIndex = stackArr.findIndex(line =>
                            line.trim().startsWith(anchor)
                        )
                        if (anchorIndex === -1) {
                            return false // 404, anchor not found
                        }
                        // Strip everything from the top until we reach the anchor line
                        // Note: We're keeping the 1st line (zero index) as it's unrelated (e.g. `TypeError`)
                        stackArr.splice(1, anchorIndex)
                        return stackArr.join('\n')
                    }

                    // Special cases due to our nested toString proxies
                    err.stack = err.stack.replace(
                        'at Object.toString (',
                        'at Function.toString ('
                    )
                    if ((err.stack || '').includes('at Function.toString (')) {
                        err.stack = stripWithBlacklist(err.stack, false)
                        throw err
                    }

                    // Try using the anchor method, fallback to blacklist if necessary
                    err.stack = stripWithAnchor(err.stack) || stripWithBlacklist(err.stack)

                    throw err // Re-throw our now sanitized error
                }
            }
        })
        return newHandler
    }
    utils.replaceProperty = (obj, propName, descriptorOverrides = {}) => {
        return Object.defineProperty(obj, propName, {
            // Copy over the existing descriptors (writable, enumerable, configurable, etc)
            ...(Object.getOwnPropertyDescriptor(obj, propName) || {}),
            // Add our overrides (e.g. value, get())
            ...descriptorOverrides
        })
    }
    utils.redirectToString = (proxyObj, originalObj) => {
        const handler = {
            apply: function (target, ctx) {
                // This fixes e.g. `HTMLMediaElement.prototype.canPlayType.toString + ""`
                if (ctx === Function.prototype.toString) {
                    return utils.makeNativeString('toString')
                }

                // `toString` targeted at our proxied Object detected
                if (ctx === proxyObj) {
                    const fallback = () =>
                        originalObj && originalObj.name
                            ? utils.makeNativeString(originalObj.name)
                            : utils.makeNativeString(proxyObj.name)

                    // Return the toString representation of our original object if possible
                    return originalObj + '' || fallback()
                }

                if (typeof ctx === 'undefined' || ctx === null) {
                    return target.call(ctx)
                }

                // Check if the toString protype of the context is the same as the global prototype,
                // if not indicates that we are doing a check across different windows., e.g. the iframeWithdirect` test case
                const hasSameProto = Object.getPrototypeOf(
                    Function.prototype.toString
                ).isPrototypeOf(ctx.toString) // eslint-disable-line no-prototype-builtins
                if (!hasSameProto) {
                    // Pass the call on to the local Function.prototype.toString instead
                    return ctx.toString()
                }

                return target.call(ctx)
            }
        }

        const toStringProxy = new Proxy(
            Function.prototype.toString,
            utils.stripProxyFromErrors(handler)
        )
        utils.replaceProperty(Function.prototype, 'toString', {
            value: toStringProxy
        })
    }
    utils.replaceWithProxy = (obj, propName, handler) => {
        const originalObj = obj[propName]
        const proxyObj = new Proxy(obj[propName], utils.stripProxyFromErrors(handler))

        utils.replaceProperty(obj, propName, { value: proxyObj })
        utils.redirectToString(proxyObj, originalObj)

        return true
    }
    utils.replaceGetterWithProxy = (obj, propName, handler) => {
        const fn = Object.getOwnPropertyDescriptor(obj, propName).get
        const fnStr = fn.toString() // special getter function string
        const proxyObj = new Proxy(fn, utils.stripProxyFromErrors(handler))

        utils.replaceProperty(obj, propName, { get: proxyObj })
        utils.patchToString(proxyObj, fnStr)

        return true
    }
    utils.cache = {
        // Used in our proxies
        Reflect: {
            get: Reflect.get.bind(Reflect),
            apply: Reflect.apply.bind(Reflect)
        },
        // Used in `makeNativeString`
        nativeToStringStr: Function.toString + '' // => `function toString() { [native code] }`
    }
    // Proxy handler templates for re-usability
    utils.makeHandler = () => ({
        // Used by simple `navigator` getter evasions
        getterValue: value => ({
            apply(target, ctx, args) {
                // Let's fetch the value first, to trigger and escalate potential errors
                // Illegal invocations like `navigator.__proto__.vendor` will throw here
                utils.cache.Reflect.apply(...arguments)
                return value
            }
        })
    })
    utils.createProxy = (pseudoTarget, handler) => {
        const proxyObj = new Proxy(pseudoTarget, utils.stripProxyFromErrors(handler))
        utils.patchToString(proxyObj)

        return proxyObj
    }
    let fingerprintValue  = Math.random()
    if (localStorage.getItem('_fvalue')) {
        fingerprintValue = localStorage.getItem('_fvalue')
    }else{
        localStorage.setItem('_fvalue', fingerprintValue)
    }
    utils.fingerprintValue = (index = 1) => {
        function randomWithSeed(seed) {
            var x = Math.sin(seed) * 10000;
            return x - Math.floor(x);
        }
        return randomWithSeed(fingerprintValue * index);
    }
    function fingerprintAutido(){
        const context = {
            "BUFFER": null,
            "getChannelData": function (e) {
                const getChannelData = e.prototype.getChannelData;
                Object.defineProperty(e.prototype, "getChannelData", {
                    "value": function () {
                        const results_1 = getChannelData.apply(this, arguments);
                        if (context.BUFFER !== results_1) {
                            context.BUFFER = results_1;
                            for (var i = 0; i < results_1.length; i += 100) {
                                let index = Math.floor(fingerprintValue * i);
                                results_1[index] = results_1[index] + fingerprintValue * 0.0000001;
                            }
                        }
                        //
                        return results_1;
                    }
                });
            },
            "createAnalyser": function (e) {
                const createAnalyser = e.prototype.__proto__.createAnalyser;
                Object.defineProperty(e.prototype.__proto__, "createAnalyser", {
                    "value": function () {
                        const results_2 = createAnalyser.apply(this, arguments);
                        const getFloatFrequencyData = results_2.__proto__.getFloatFrequencyData;
                        Object.defineProperty(results_2.__proto__, "getFloatFrequencyData", {
                            "value": function () {
                                const results_3 = getFloatFrequencyData.apply(this, arguments);
                                for (var i = 0; i < arguments[0].length; i += 100) {
                                    let index = Math.floor(fingerprintValue * i);
                                    arguments[0][index] = arguments[0][index] + fingerprintValue * 0.1;
                                }
                                //
                                return results_3;
                            }
                        });
                        //
                        return results_2;
                    }
                });
            }
        };
//
        context.getChannelData(AudioBuffer);
        context.createAnalyser(AudioContext);
        context.getChannelData(OfflineAudioContext);
        context.createAnalyser(OfflineAudioContext);

        console.log('fingerprint.autido');
    }

    function fingerprintBattery(){
        function fakeCharging() {
            return [true,true,true,true,true,true,false,false][Math.floor(fingerprintValue * 7)];
        }
        function fakeChargingTime() {
            var dt2 = Math.floor(fingerprintValue * 6100) + 1200;
            return [dt2,Infinity,0,0,0,0,0,0,Infinity,Infinity,0,0,0,0,dt2][Math.floor(fingerprintValue * 15)];
        }
        function fakeDischargingTime() {
            var dt = Math.floor(fingerprintValue * 7000) + 1150;
            return [Infinity,Infinity,Infinity,Infinity,Infinity,Infinity,Infinity,dt,dt,dt,dt][Math.floor(fingerprintValue * 9)];
        }
        function fakeLevel() {
            return [1,1,1,1,1,1,1,1,56.99999999999999,57.99999999999999,58.99999999999999,58.99999999999999,70.99999999999999,0.84,0.77,0.99,0.98,0.98,0.98,0.99,0.98,0.97,0.96,0.95,0.94,0.93,0.92,0.91,0.90,0.89,0.88,0.87,0.86,0.76,0.65,0.48,0.73,0.74,0.75][Math.floor(fingerprintValue * 28)];
        }

        const fakeChargingValue = fakeCharging();
        const fakeLevelValue = fakeLevel();
        const fakeDischargingValue = fakeDischargingTime();
        const fakeChargingTimeValue = fakeChargingTime();

        if (window['BatteryManager']){
            Object.defineProperties(BatteryManager.prototype, {
                charging: {
                    configurable: true,
                    enumerable: true,
                    get: function getCharging() {
                        return fakeChargingValue;
                    }
                },

                chargingTime: {
                    configurable: true,
                    enumerable: true,
                    get: function getChargingTime() {
                        return fakeChargingTimeValue;
                    }
                },

                dischargingTime: {
                    configurable: true,
                    enumerable: true,
                    get: function getDischargingTime() {
                        return fakeDischargingValue;
                    }
                },


                level: {
                    configurable: true,
                    enumerable: true,
                    get: function getLevel() {
                        return fakeLevelValue;
                    }
                }
            });
        }
    }
    function fingerprintCanvas(){
        function random(list) {
            let min = 0;
            let max = list.length
            return list[Math.floor(fingerprintValue * (max - min)) + min];
        }

        let rsalt = random([...Array(7).keys()].map(a => a - 3))
        let gsalt = random([...Array(7).keys()].map(a => a - 3))
        let bsalt = random([...Array(7).keys()].map(a => a - 3))
        let asalt = random([...Array(7).keys()].map(a => a - 3))

        const shift = {
            'r': rsalt,
            'g': gsalt,
            'b': bsalt,
            'a': asalt,
            //'a': Math.floor(0 * 255)+1
        };
        const toBlobOrigion = HTMLCanvasElement.prototype.toBlob;
        const toDataURLOrigion = HTMLCanvasElement.prototype.toDataURL;
        const getImageDataOrigion = CanvasRenderingContext2D.prototype.getImageData;
        const toStringOrigion = Function.prototype.toString;
        //
        var noisify = function (canvas, context) {
            const width = canvas.width, height = canvas.height;
            const imageData = getImageDataOrigion.apply(context, [0, 0, width, height]);
            for (let i = 0; i < height; i++) {
                for (let j = 0; j < width; j++) {
                    const n = ((i * (width * 4)) + (j * 4));
                    if (imageData.data[n + 0] + shift.r > 0) {
                        imageData.data[n + 0] = imageData.data[n + 0] + shift.r;
                    }
                    if (imageData.data[n + 1] + shift.r > 0) {
                        imageData.data[n + 1] = imageData.data[n + 1] + shift.g;
                    }
                    if (imageData.data[n + 2] + shift.r > 0) {
                        imageData.data[n + 2] = imageData.data[n + 2] + shift.b;
                    }
                    if (imageData.data[n + 3] + shift.r > 0) {
                        imageData.data[n + 3] = imageData.data[n + 3] + shift.a;
                    }
                }
            }
            context.putImageData(imageData, 0, 0);
        };
        //
        Object.defineProperty(HTMLCanvasElement.prototype, "toBlob", {
            "value": function toBlob(a) {
                var context = this.getContext("2d");
                if (!context) {
                    context = this.getContext("experimental-webgl", { preserveDrawingBuffer: true });
                    if (context) {
                        return toBlobOrigion.apply(this, arguments);
                    }
                }
                else {
                    noisify(this);
                    return toBlobOrigion.apply(this, arguments);
                }
            }
        });
        //
        Object.defineProperty(HTMLCanvasElement.prototype, "toDataURL", {
            "value": function toDataURL() {
                var context = this.getContext("2d");
                if (!context) {
                    context = this.getContext("experimental-webgl", { preserveDrawingBuffer: true });
                    if (context) {
                        return toDataURLOrigion.apply(this, arguments);
                    }
                }
                else {
                    noisify(this, context);
                    return toDataURLOrigion.apply(this, arguments);
                }
            }
        });
        //
        //Object.defineProperty(CanvasRenderingContext2D.prototype, "getImageData", {
        //    "value": function getImageData(a, b, c, d) {
        //        noisify(this.canvas, this);
        //        return getImageDataOrigion.apply(this, arguments);
        //    }
        //});
        Object.defineProperty(Function.prototype, "toString", {
            "value": function toString() {
                if (this.name && this.name === "toBlob") {
                    return "function toBlob() { [native code] }";
                }
                else if (this.name && this.name === "toDataURL") {
                    return "function toDataURL() { [native code] }";
                }
                else if (this.name && this.name === "getImageData") {
                    return "function getImageData() { [native code] }";
                }
                return toStringOrigion.apply(this, arguments);
            }
        });
    }
    function fingerprintClientRects(){
        var fontRandom = 1;
        var rand = {
            "noise": function () {
                var SIGN = utils.fingerprintValue(fontRandom++) < utils.fingerprintValue(fontRandom++) ? -1 : 1;
                return Math.floor(utils.fingerprintValue(fontRandom++) + SIGN * utils.fingerprintValue(fontRandom++));
            },
            "sign": function () {
                const tmp = [-1, -1, -1, -1, -1, -1, +1, -1, -1, -1];
                const index = Math.floor(utils.fingerprintValue(fontRandom++) * tmp.length);
                return tmp[index];
            },
            "value": function (d) {
                const valid = d && rand.sign() === 1;
                return valid ? d + rand.noise() : d;
            }
        };
        Object.defineProperty(HTMLElement.prototype, "getBoundingClientRect", {
            value() {
                let rects = (Object.getOwnPropertyDescriptor(Element.prototype, "getBoundingClientRect").value).call(this);

                let properties = ['height', 'width', 'x', 'y', 'top', 'right', 'bottom', 'left'];
                for (let i = 0; i < properties.length; i++) {
                    let name = properties[i];
                    let value = rects[name];

                    rects[name] = value + fingerprintValue;
                }

                return rects;
            },
        });


        Object.defineProperty(HTMLElement.prototype, "getClientRects", {
            value() {
                let list = (Object.getOwnPropertyDescriptor(Element.prototype, "getClientRects").value).call(this);
                let rects = list[0];
                // console.log('getClientRects' , JSON.stringify(rects));
                let properties = ['height', 'width', 'x', 'y', 'top', 'right', 'bottom', 'left'];
                for (let i = 0; i < properties.length; i++) {
                    let name = properties[i];
                    let value = rects[name];

                    // console.log('getClientRects set ' , name , value , ' to ' , value + fingerprintValue);
                    rects[name] = value + fingerprintValue;
                }

                return list;
            },
        });

        console.log('fingerprint.clientRects');
    }

    function fingerprintFont(){
        var fontRandom = 1;
        var rand = {
            "noise": function () {
                var SIGN = utils.fingerprintValue(fontRandom++) < utils.fingerprintValue(fontRandom++) ? -1 : 1;
                return Math.floor(utils.fingerprintValue(fontRandom++) + SIGN * utils.fingerprintValue(fontRandom++));
            },
            "sign": function () {
                const tmp = [-1, -1, -1, -1, -1, -1, +1, -1, -1, -1];
                const index = Math.floor(utils.fingerprintValue(fontRandom++) * tmp.length);
                return tmp[index];
            },
            "value": function (d) {
                const valid = d && rand.sign() === 1;
                return valid ? d + rand.noise() : d;
            }
        };
        Object.defineProperty(HTMLElement.prototype, "offsetHeight", {
            get() {
                const value = Math.floor(this.getBoundingClientRect().height);
                return rand.value(value);
            }
        });
        Object.defineProperty(HTMLElement.prototype, "offsetWidth", {
            get() {
                const value = Math.floor(this.getBoundingClientRect().width);
                return rand.value(value);
            }
        });
    }

// 随机 media设备ID
    function fingerprintMedia(){
        // 随机生成 65位设备ID
        function randomString(len,index) {
            len = len || 64;
            var $chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
            var maxPos = $chars.length;
            var pwd = '';
            for (var i = 0; i < len; i++) {
                pwd += $chars.charAt(Math.floor(fingerprintValue * index * maxPos));
            }
            return pwd;
        }

        let groupId = randomString(64, 0);
        let devices = [
            {
                "groupId": groupId,
                "deviceId": randomString(64, 1),
                "kind": "audioinput",
                "label": ""
            },
            {
                "groupId": groupId,
                "deviceId": randomString(64, 2),
                "kind": "audioinput",
                "label": "",
            },
            {
                "groupId": groupId,
                "deviceId": randomString(64, 3),
                "kind": "audiooutput",
                "label": "",
            },
            {
                "groupId": groupId,
                "deviceId": randomString(64, 4),
                "kind": "audiooutput",
                "label": "",
            }
        ];

        if (navigator.mediaDevices && navigator.mediaDevices.enumerateDevices) {
            navigator.mediaDevices.__proto__.enumerateDevices = function () {
                return new Promise((resolve, reject) => {
                    resolve(devices);
                });
            }
        }
        if (window.MediaStreamTrack && window.MediaStreamTrack.getSources) {
            window.MediaStreamTrack.getSources = function () {
                return devices;
            }
        }
    }
    function fingerprintProtect(){
        let hardwareConcurrency = parseInt(fingerprintValue * 10);
        if (hardwareConcurrency < 2) {
            hardwareConcurrency = 2;
        }
        console.log('hardwareConcurrency', hardwareConcurrency);
        utils.replaceGetterWithProxy(
            Object.getPrototypeOf(navigator),
            'hardwareConcurrency',
            utils.makeHandler().getterValue(hardwareConcurrency)
        )
    }

    function fingerprintWebgl(){
        const VENDORS = [
            'ARM',
            'ATI Technologies Inc.',
            'Google Inc.',
            'Intel',
            'Intel Inc.',
            'Nvidia Corporation',
            'Qualcomm',
        ]
        const RENDERERS = [
            'Intel(R) HD Graphics 6000',
            'Intel(R) Iris(TM) Graphics 6100',
            'Intel(R) Iris(TM) Plus Graphics 640',
            'Intel Iris Pro OpenGL Engine',
            'Intel HD Graphics 4000 OpenGL Engine',
            'Google SwiftShader',
            'ANGLE (Intel(R) HD Graphics 620 Direct3D11 vs_5_0 ps_5_0)',
            'Intel HD Graphics 5000 OpenGL Engine',
            'ANGLE (Intel(R) HD Graphics 520 Direct3D11 vs_5_0 ps_5_0)',
            'Intel Iris OpenGL Engine',
            'ANGLE (Intel(R) HD Graphics Direct3D11 vs_5_0 ps_5_0)',
            'ANGLE (Intel(R) HD Graphics 4600 Direct3D11 vs_5_0 ps_5_0)',
            'ANGLE (Intel(R) HD Graphics 530 Direct3D11 vs_5_0 ps_5_0)',
            'ANGLE (Intel(R) HD Graphics 630 Direct3D11 vs_5_0 ps_5_0)',
            'ANGLE (Intel(R) UHD Graphics 620 Direct3D11 vs_5_0 ps_5_0)',
            'Intel(R) Iris(TM) Graphics 650',
            'ANGLE (Intel(R) HD Graphics 5500 Direct3D11 vs_5_0 ps_5_0)',
            'ANGLE (Intel(R) HD Graphics Family Direct3D11 vs_5_0 ps_5_0)',
            'Mesa DRI Intel(R) HD Graphics 400 (Braswell)',
            'ANGLE (Intel(R) HD Graphics 4000 Direct3D11 vs_5_0 ps_5_0)',
            'Intel(R) Iris(TM) Graphics 550',
            'AMD Radeon Pro 560 OpenGL Engine',
            'ANGLE (NVIDIA GeForce GTX 1060 6GB Direct3D11 vs_5_0 ps_5_0)',
            'Intel(R) Iris(TM) Plus Graphics 655',
            'NVIDIA GeForce GT 750M OpenGL Engine',
            'AMD Radeon Pro 555 OpenGL Engine',
            'NVIDIA GeForce GT 650M OpenGL Engine',
            'Mesa DRI Intel(R) Bay Trail',
            'ANGLE (NVIDIA GeForce GTX 1050 Ti Direct3D11 vs_5_0 ps_5_0)',
            'ANGLE (NVIDIA GeForce GTX 1070 Direct3D11 vs_5_0 ps_5_0)',
            'AMD Radeon R9 M370X OpenGL Engine',
            'Intel(R) Iris(TM) Graphics 540',
            'ANGLE (Intel(R) HD Graphics Direct3D9Ex vs_3_0 ps_3_0)',
            'ANGLE (NVIDIA GeForce GTX 970 Direct3D11 vs_5_0 ps_5_0)',
            'ANGLE (Intel(R) HD Graphics Direct3D11 vs_4_1 ps_4_1)',
            'ANGLE (NVIDIA GeForce GTX 1060 3GB Direct3D11 vs_5_0 ps_5_0)',
            'ANGLE (NVIDIA GeForce GTX 960 Direct3D11 vs_5_0 ps_5_0)',
            'ANGLE (Intel(R) UHD Graphics 630 Direct3D11 vs_5_0 ps_5_0)',
            'ANGLE (NVIDIA GeForce GTX 750 Ti Direct3D11 vs_5_0 ps_5_0)',
            'Intel HD Graphics 3000 OpenGL Engine',
            'ANGLE (Intel(R) HD Graphics 3000 Direct3D11 vs_4_1 ps_4_1)',
            'ANGLE (Intel(R) HD Graphics 4400 Direct3D11 vs_5_0 ps_5_0)'
        ]
        var random = {
            "value": function () { return  fingerprintValue},
            "newvalue": function (m, n) {
                return Math.floor(random.value() * n * 2) + m - n;
            },
            "item": function (e) {
                var rand = e.length * random.value();
                return e[Math.floor(rand)];
            },
            "array": function (e) {
                var rand = random.item(e);
                return new Int32Array([rand, rand]);
            },
            "items": function (e, n) {
                var length = e.length;
                var result = new Array(n);
                var taken = new Array(length);
                if (n > length) n = length;
                //
                while (n--) {
                    var i = Math.floor(random.value() * length);
                    result[n] = e[i in taken ? taken[i] : i];
                    taken[i] = --length in taken ? taken[length] : length;
                }
                //
                return result;
            }
        };

        function safeOverwrite(obj, prop, newVal) {
            let props = Object.getOwnPropertyDescriptor(obj, prop);
            props["value"] = newVal;

            return props;
        }

        let changeMap = {};

        let vendor = random.item(VENDORS);
        let renderer = random.item(RENDERERS);
        let paramChanges = {
            // vendor
            37445: vendor,
            // renderer
            37446: renderer,
            7938: random.item(['WebGL 2.0 (OpenGL ES 3.0 Chromium)', 'WebGL 1.0 (OpenGL ES 2.0 Chromium)']),
            35724: random.item(['WebGL GLSL ES 3.00 (OpenGL ES GLSL ES 3.0 Chromium)', 'WebGL GLSL ES 2.00 (OpenGL ES GLSL ES 2.0 Chromium)']),
            32937: random.item([8, 4]),
            35978: random.item([120, 64]),
            35968: random.item([4, 30]),
            35374: random.item([24, 32]),
            34024: random.newvalue(16384, 64),
            3379: random.newvalue(16384, 64),
            34076: random.newvalue(16384, 64),
            35658: random.newvalue(16384, 64),
            35377: random.newvalue(212992, 1024),
            35379: random.newvalue(200704, 1024)
        };
        changeMap = Object.assign(changeMap, paramChanges);

        var frame = window;

        ["WebGLRenderingContext", "WebGL2RenderingContext"].forEach(function (ctx) {
            if (!frame[ctx]) return;

            // Modify getParameter
            let oldParam = frame[ctx].prototype.getParameter;
            Object.defineProperty(frame[ctx].prototype, "getParameter",
                safeOverwrite(frame[ctx].prototype, "getParameter", function (param) {
                    if (changeMap[param])
                        return changeMap[param];
                    else
                        return oldParam.apply(this, arguments);
                })
            );

            // Modify bufferData (this updates the image hash)
            let oldBuffer = frame[ctx].prototype.bufferData;
            Object.defineProperty(frame[ctx].prototype, "bufferData",
                safeOverwrite(frame[ctx].prototype, "bufferData", function () {
                    for (let i = 0; i < arguments[1].length; i++) {
                        arguments[1][i] += 0.1 * random.value() * arguments[1][i];
                    }
                    return oldBuffer.apply(this, arguments);
                })
            );
        });
    }

    function fingerprintNavigatorPlugins(){
        function generateMagicArray(
                dataArray = [],
                proto = MimeTypeArray.prototype,
                itemProto = MimeType.prototype,
                itemMainProp = 'type'
            ) {
                // Quick helper to set props with the same descriptors vanilla is using
                const defineProp = (obj, prop, value) =>
                    Object.defineProperty(obj, prop, {
                        value,
                        writable: false,
                        enumerable: false, // Important for mimeTypes & plugins: `JSON.stringify(navigator.mimeTypes)`
                        configurable: true
                    })

                // Loop over our fake data and construct items
                const makeItem = data => {
                    const item = {}
                    for (const prop of Object.keys(data)) {
                        if (prop.startsWith('__')) {
                            continue
                        }
                        defineProp(item, prop, data[prop])
                    }
                    return patchItem(item, data)
                }

                const patchItem = (item, data) => {
                    let descriptor = Object.getOwnPropertyDescriptors(item)

                    // Special case: Plugins have a magic length property which is not enumerable
                    // e.g. `navigator.plugins[i].length` should always be the length of the assigned mimeTypes
                    if (itemProto === Plugin.prototype) {
                        descriptor = {
                            ...descriptor,
                            length: {
                                value: data.__mimeTypes.length,
                                writable: false,
                                enumerable: false,
                                configurable: true // Important to be able to use the ownKeys trap in a Proxy to strip `length`
                            }
                        }
                    }

                    // We need to spoof a specific `MimeType` or `Plugin` object
                    const obj = Object.create(itemProto, descriptor)

                    // Virtually all property keys are not enumerable in vanilla
                    const blacklist = [...Object.keys(data), 'length', 'enabledPlugin']
                    return new Proxy(obj, {
                        ownKeys(target) {
                            return Reflect.ownKeys(target).filter(k => !blacklist.includes(k))
                        },
                        getOwnPropertyDescriptor(target, prop) {
                            if (blacklist.includes(prop)) {
                                return undefined
                            }
                            return Reflect.getOwnPropertyDescriptor(target, prop)
                        }
                    })
                }

                const magicArray = []

                // Loop through our fake data and use that to create convincing entities
                dataArray.forEach(data => {
                    magicArray.push(makeItem(data))
                })

                // Add direct property access  based on types (e.g. `obj['application/pdf']`) afterwards
                magicArray.forEach(entry => {
                    defineProp(magicArray, entry[itemMainProp], entry)
                })

                // This is the best way to fake the type to make sure this is false: `Array.isArray(navigator.mimeTypes)`
                const magicArrayObj = Object.create(proto, {
                    ...Object.getOwnPropertyDescriptors(magicArray),

                    // There's one ugly quirk we unfortunately need to take care of:
                    // The `MimeTypeArray` prototype has an enumerable `length` property,
                    // but headful Chrome will still skip it when running `Object.getOwnPropertyNames(navigator.mimeTypes)`.
                    // To strip it we need to make it first `configurable` and can then overlay a Proxy with an `ownKeys` trap.
                    length: {
                        value: magicArray.length,
                        writable: false,
                        enumerable: false,
                        configurable: true // Important to be able to use the ownKeys trap in a Proxy to strip `length`
                    }
                })

                function generateFunctionMocks(
                    proto,
                    itemMainProp,
                    dataArray
                ) {
                    return {
                        /** Returns the MimeType object with the specified index. */
                        item: utils.createProxy(proto.item, {
                            apply(target, ctx, args) {
                                if (!args.length) {
                                    throw new TypeError(
                                        `Failed to execute 'item' on '${
                                            proto[Symbol.toStringTag]
                                        }': 1 argument required, but only 0 present.`
                                    )
                                }
                                // Special behavior alert:
                                // - Vanilla tries to cast strings to Numbers (only integers!) and use them as property index lookup
                                // - If anything else than an integer (including as string) is provided it will return the first entry
                                const isInteger = args[0] && Number.isInteger(Number(args[0])) // Cast potential string to number first, then check for integer
                                // Note: Vanilla never returns `undefined`
                                return (isInteger ? dataArray[Number(args[0])] : dataArray[0]) || null
                            }
                        }),
                        /** Returns the MimeType object with the specified name. */
                        namedItem: utils.createProxy(proto.namedItem, {
                            apply(target, ctx, args) {
                                if (!args.length) {
                                    throw new TypeError(
                                        `Failed to execute 'namedItem' on '${
                                            proto[Symbol.toStringTag]
                                        }': 1 argument required, but only 0 present.`
                                    )
                                }
                                return dataArray.find(mt => mt[itemMainProp] === args[0]) || null // Not `undefined`!
                            }
                        }),
                        /** Does nothing and shall return nothing */
                        refresh: proto.refresh
                            ? utils.createProxy(proto.refresh, {
                                apply(target, ctx, args) {
                                    return undefined
                                }
                            })
                            : undefined
                    }
                }
                // Generate our functional function mocks :-)
                const functionMocks = generateFunctionMocks(
                    proto,
                    itemMainProp,
                    magicArray
                )

                // We need to overlay our custom object with a JS Proxy
                const magicArrayObjProxy = new Proxy(magicArrayObj, {
                    get(target, key = '') {
                        // Redirect function calls to our custom proxied versions mocking the vanilla behavior
                        if (key === 'item') {
                            return functionMocks.item
                        }
                        if (key === 'namedItem') {
                            return functionMocks.namedItem
                        }
                        if (proto === PluginArray.prototype && key === 'refresh') {
                            return functionMocks.refresh
                        }
                        // Everything else can pass through as normal
                        return utils.cache.Reflect.get(...arguments)
                    },
                    ownKeys(target) {
                        // There are a couple of quirks where the original property demonstrates "magical" behavior that makes no sense
                        // This can be witnessed when calling `Object.getOwnPropertyNames(navigator.mimeTypes)` and the absense of `length`
                        // My guess is that it has to do with the recent change of not allowing data enumeration and this being implemented weirdly
                        // For that reason we just completely fake the available property names based on our data to match what regular Chrome is doing
                        // Specific issues when not patching this: `length` property is available, direct `types` props (e.g. `obj['application/pdf']`) are missing
                        const keys = []
                        const typeProps = magicArray.map(mt => mt[itemMainProp])
                        typeProps.forEach((_, i) => keys.push(`${i}`))
                        typeProps.forEach(propName => keys.push(propName))
                        return keys
                    },
                    getOwnPropertyDescriptor(target, prop) {
                        if (prop === 'length') {
                            return undefined
                        }
                        return Reflect.getOwnPropertyDescriptor(target, prop)
                    }
                })

                return magicArrayObjProxy
            }
        function generateMimeTypeArray(mimeTypesData){
            return generateMagicArray(
                mimeTypesData,
                MimeTypeArray.prototype,
                MimeType.prototype,
                'type'
            )
        }
        function generatePluginArray(pluginsData){
            return generateMagicArray(
                pluginsData,
                PluginArray.prototype,
                Plugin.prototype,
                'name'
            )
        }
        const data = {
            "mimeTypes": [
                {
                    "type": "application/pdf",
                    "suffixes": "pdf",
                    "description": "",
                    "__pluginName": "Chrome PDF Viewer"
                },
                {
                    "type": "application/x-google-chrome-pdf",
                    "suffixes": "pdf",
                    "description": "Portable Document Format",
                    "__pluginName": "Chrome PDF Plugin"
                },
                {
                    "type": "application/x-nacl",
                    "suffixes": "",
                    "description": "Native Client Executable",
                    "__pluginName": "Native Client"
                },
                {
                    "type": "application/x-pnacl",
                    "suffixes": "",
                    "description": "Portable Native Client Executable",
                    "__pluginName": "Native Client"
                }
            ],
            "plugins": [
                {
                    "name": "Chrome PDF Plugin",
                    "filename": "internal-pdf-viewer",
                    "description": "Portable Document Format",
                    "__mimeTypes": ["application/x-google-chrome-pdf"],
                    "must": true
                },
                {
                    "name": "Chrome PDF Viewer",
                    "filename": "internal-pdf-viewer",
                    "description": "Portable Document Format",
                    "__mimeTypes": ["application/pdf"],
                    "must": true
                },
                {
                    "name": "Chromium PDF Viewer",
                    "filename": "internal-pdf-viewer",
                    "description": "Portable Document Format",
                    "__mimeTypes": ["application/pdf"]
                },
                {
                    "name": "Microsoft Edge PDF Viewer",
                    "filename": "internal-pdf-viewer",
                    "description": "Portable Document Format",
                    "__mimeTypes": ["application/pdf"]
                },
                {
                    "name": "WebKit built-in PDF",
                    "filename": "internal-pdf-viewer",
                    "description": "Portable Document Format",
                    "__mimeTypes": ["application/pdf"]
                },
                {
                    "name": "Native Client",
                    "filename": "internal-nacl-plugin",
                    "description": "",
                    "__mimeTypes": ["application/x-nacl", "application/x-pnacl"]
                }
            ]
        }

        let newPlugins = [];
        for (let i = 0; i < data.plugins.length; i++) {
            let plugin = data.plugins[i];
            if (plugin.name.includes("[random]")){
                plugin.name = plugin.name.replace("[random]", Math.random().toString(36).substr(2));
            }
            if (plugin.must){
                newPlugins.push(plugin);
            } else {
                if (fingerprintValue < 0.5){
                    newPlugins.push(plugin);
                }
            }
        }

        const newData = {
            mimeTypes: data.mimeTypes,
            plugins: newPlugins
        }
        const mimeTypes = generateMimeTypeArray(newData.mimeTypes)
        const plugins = generatePluginArray(newData.plugins)

        // Plugin and MimeType cross-reference each other, let's do that now
        // Note: We're looping through `data.plugins` here, not the generated `plugins`
        for (const pluginData of newData.plugins) {
            pluginData.__mimeTypes.forEach((type, index) => {
                plugins[pluginData.name][index] = mimeTypes[type]

                Object.defineProperty(plugins[pluginData.name], type, {
                    value: mimeTypes[type],
                    writable: false,
                    enumerable: false, // Not enumerable
                    configurable: true
                })
                Object.defineProperty(mimeTypes[type], 'enabledPlugin', {
                    value:
                        type === 'application/x-pnacl'
                            ? mimeTypes['application/x-nacl'].enabledPlugin // these reference the same plugin, so we need to re-use the Proxy in order to avoid leaks
                            : new Proxy(plugins[pluginData.name], {}), // Prevent circular references
                    writable: false,
                    enumerable: false, // Important: `JSON.stringify(navigator.plugins)`
                    configurable: true
                })
            })
        }
        const patchNavigator = (name, value) =>
            utils.replaceProperty(Object.getPrototypeOf(navigator), name, {
                get() {
                    return value
                }
            })

        patchNavigator('mimeTypes', mimeTypes)
        patchNavigator('plugins', plugins)
    }

    fingerprintAutido();
    console.log('fingerprint fingerprintAutido======');
    fingerprintBattery();
    console.log('fingerprint fingerprintBattery======');
    fingerprintCanvas();
    console.log('fingerprint fingerprintCanvas======');
    fingerprintClientRects();
    console.log('fingerprint fingerprintClientRects======');
    fingerprintFont();
    console.log('fingerprint fingerprintFont======');
    fingerprintMedia();
    console.log('fingerprint fingerprintMedia======');
    fingerprintProtect();
    console.log('fingerprint fingerprintProtect======');
    // fingerprintWebgl();
    console.log('fingerprint fingerprintWebgl======');
    fingerprintNavigatorPlugins();
    console.log('fingerprint fingerprintNavigatorPlugins======');

    if (navigator.webdriver === false) {
        // Post Chrome 89.0.4339.0 and already good
    } else if (navigator.webdriver === undefined) {
        // Pre Chrome 89.0.4339.0 and already good
    } else {
        // Pre Chrome 88.0.4291.0 and needs patching
        delete Object.getPrototypeOf(navigator).webdriver
    }

})();
