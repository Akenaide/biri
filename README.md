# Biri

Package that provide http.Client that will use a http proxy.

## Quickstart

```go
  biri.ProxyStart()
  
  proxy := biri.GetClient()
  resp, _ := proxy.Client.Get(url)
  biri.Done()
```

## Manage proxy by yourself

You can readd a proxy after a successful request with:
```go
  proxy.Readd()
 ```
 
 or you can ban it if the request did not worked:
```go
  proxy.Ban()
```

## Configuration

Here the basic default configuration:
```go
  var Config = &config{
    proxyWebpage:           "https://free-proxy-list.net/",
    PingServer:             "https://www.google.com/",
    TickMinuteDuration:     3,
    numberAvailableProxies: 30,
    Verbose:                1,
    Timeout:                10,
    AnonymousLevel:         []string{"elite proxy", "transparent"},
  }
```
