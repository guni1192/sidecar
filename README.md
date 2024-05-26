# sidecar

## usage

```
% ./bin/sidecar --pre-exec "python3 -m http.server 8000" --healthcheck --retries 10 curl localhost:8000
<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01//EN" "http://www.w3.org/TR/html4/strict.dtd">
```
