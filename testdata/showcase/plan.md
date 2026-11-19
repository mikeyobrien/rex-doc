---
title: Login fix plan
status: ready for review
---

The agent traced the failed sign-ins to an expired gateway certificate. Here's the plan, with the [incident notes](notes.md) linked.

> [!IMPORTANT]
> Rotating the certificate drops active sessions for about 30 seconds.

## Steps

- [x] Reproduce the failure in staging
- [x] Confirm the certificate expired at 09:00 UTC
- [ ] Rotate the certificate during the 18:00 window
- [ ] Watch sign-in errors for an hour :rex:

```sh
openssl x509 -in gateway.pem -noout -enddate
```

| Region | Users affected | Status |
|:-------|---------------:|:------:|
| EU     | 4,200          | fixing |
| US     | 0              | fine   |

![Sign-in errors before and after the fix](dashboard.png)
