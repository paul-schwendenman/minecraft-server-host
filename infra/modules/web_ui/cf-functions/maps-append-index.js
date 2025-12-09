function handler(event) {
    var req = event.request;
    var uri = req.uri;

    // If it ends with '/', append index.html
    if (uri.endsWith('/')) {
        req.uri = uri + 'index.html';
        return req;
    }

    // If it has no file extension (no '.' after last '/'), treat as a folder
    var lastSlash = uri.lastIndexOf('/');
    var lastDot = uri.lastIndexOf('.');
    if (lastDot < 0 || lastDot < lastSlash) {
        req.uri = uri.replace(/\/?$/, '/') + 'index.html';
    }

    return req;
}
