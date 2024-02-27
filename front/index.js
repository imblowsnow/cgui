console.log('front/index.js');

fetch('https://customer.xiaohongshu.com/api/cas/generateQrCode', {
    method: 'POST',
    headers: {
        'Content-Type': 'application/json'
    },
    body: '{"subsystem":"business"}'
})
