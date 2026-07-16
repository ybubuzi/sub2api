fetch("https://sub.kedaya.xyz/api/v1/auth/me?timezone=Asia%2FShanghai", {
  "headers": {
    "accept": "application/json, text/plain, */*",
    "accept-language": "zh",
    "authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxOTQ1LCJlbWFpbCI6ImJ1YnV6aUA4OC5jb20iLCJyb2xlIjoidXNlciIsInRva2VuX3ZlcnNpb24iOjc4OTEyMTI4MTkzNDg5MzYzNjUsImV4cCI6MTc4MzI1NDQ3NywibmJmIjoxNzgzMTY4MDc3LCJpYXQiOjE3ODMxNjgwNzd9.PSt8zjQed99NwCh8DF9kf45CkC_2F5LURRN2RpnSkmc",
    "priority": "u=1, i",
    "sec-ch-ua": "\"Microsoft Edge\";v=\"149\", \"Chromium\";v=\"149\", \"Not)A;Brand\";v=\"24\"",
    "sec-ch-ua-arch": "\"x86\"",
    "sec-ch-ua-bitness": "\"64\"",
    "sec-ch-ua-full-version": "\"149.0.4022.98\"",
    "sec-ch-ua-full-version-list": "\"Microsoft Edge\";v=\"149.0.4022.98\", \"Chromium\";v=\"149.0.7827.201\", \"Not)A;Brand\";v=\"24.0.0.0\"",
    "sec-ch-ua-mobile": "?0",
    "sec-ch-ua-model": "\"\"",
    "sec-ch-ua-platform": "\"Windows\"",
    "sec-ch-ua-platform-version": "\"19.0.0\"",
    "sec-fetch-dest": "empty",
    "sec-fetch-mode": "cors",
    "sec-fetch-site": "same-origin",
    "cookie": "__stripe_mid=adbfc38e-e0bc-48fe-ba2c-f4e45930e2492273dc; cf_clearance=0VP4HZIwe.NV4q2A_.gl.3EZ3uCcD7m2BYuscH9Iql8-1783168026-1.2.1.1-6NZIJatL.CWSb8l4zW9qTAoySlxeDpL9AV7SL1guoMimQlPnIDyqmaKYt443asxrDTrFzCIqvrBBPmmGT07JTAQ8siokx7mAeAR4.czpyJWS.4QKcp3.KpRLmG94eWHmbSmuj_G0j5i22uxHaHWTxTA5mxzIkzKpYBJJNz.ZunkO3FcWliZ3n_iq4VCSxnUezavCkDyyM1jX9cixPe7ld2SyXOcu83m7OQkaakbWGpuTmGhVAw.9syigPfJ6PvtGjSFhnm_6dR2T0W2x_iEM14mD05Uu2v.CKCcuu_pq9RsrlzgiZQBiAWmcc0O.vPfk0etXQNq3LVv51J0skHyEjnqwpJfmJWn9.Lr4Cti8hzcXh6WjUC10ajyKXm3miRrlf16y6o.jKHbnRbRr8TNcnhHKsY0ipoCz5nFkG6PE8zYBymQhBTYlKs6V7XkFpwgqcA4cNX__aoL71mSUzTFJM164Lkg3uLqhEc6ZS0mxV.65LNNI38vjXzGR3AxthAu9; __stripe_sid=a9cec47e-7dd9-4e5d-bb0b-84928061c84d79c428",
    "Referer": "https://sub.kedaya.xyz/dashboard"
  },
  "body": null,
  "method": "GET"
}).then(response => response.json()).then(data => {
  console.log(data);
}).catch(error => {
  console.error('Error:', error);
});