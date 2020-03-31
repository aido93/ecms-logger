# Current status: development!

# What is it?

This is robust logger for [Echo web framework](https://echo.labstack.com) that sends access logs to [Clickhouse](https://clickhouse.tech).
It adds metadata for every request:

- Request GeoIP2 via MaxMind Database
- User Sessions

# Why not using just nginx?
It does not support user sessions with redis. Also there are can be different problems with determining real client IP if nginx is behind another proxy.

# Example config
```yaml
# logger.yaml
maxmind:
  # path to MaxMind Cities database
  db: /maxmind/GeoLite2-City.mmdb
  source:
    # Can be X-Forwarded-For, X-Real-Ip, CF-Connecting-IP
    header: CF-Connecting-IP
    # Or
    #remoteAddr: true
clickhouse:
  # maxQueueSize must be more than batchSize
  batchSize: 100
  maxQueueSize: 150
  table: user_mgmt_actions
  period:   10s
  reserve:
    dir: /access-log
    rotate:
      maxFiles: 10
      maxSize:  200k
  connection:
    host:      127.0.0.1
    port:      9000
    user:      default
    port:      default
    altHosts:  []
    connLimit: 3
    idleLimit: 1
    timeout:   1s
session:
  cookie:  X-Authorization
```

# Example usage

```

```
