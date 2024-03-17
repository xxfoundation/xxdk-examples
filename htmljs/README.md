# reactjs

An xxdk example written for plain old html using javascript.

## Running This Example

Open the `index.html` and it will attempt to run. Depending on your
browser, the underlying web assembly environment may not have enough
access to run properly. It is recommended to start a python webserver
in this directory:

```
python3 -m http.server
```

This will start up a webserver on the local host that serves the
index.html file. That file has all the javascript built in to run the
entire example.

The script will download several scripts from content distribution
networks. We recommend hosting these locally especially to speed up
your testing.

## How This Example Was Built

We extracted the code from the reactjs example for the design. Then we
added the necessary `<script>` tags. In particular, we added `Dexie`
for indexedDB and `base64js` to handle base64 to Uint8Array
conversions.
