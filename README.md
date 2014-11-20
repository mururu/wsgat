# wsgat

WebSocket cat. wsgat is heavily inspired by [wscat](https://github.com/einaros/ws/tree/master/wscat).

## Usage

```
$ wsgat connect ws://echo.websocket.org:80
connected (press CTRL+C to quit)
> hi
  < hi
> yo
  < yo
>
```

```
$ wsgat listen 80
listening on port 80 (press CTRL+C to quit)
  client connected
  < hi
  < yo
```
