var params = {
    messagePat: /^\["(.+?)"] ?|^\[([^"]+?)] ?/,
    splitPat: /] \[/,
    kvPat: /^(".+?"|[^"]+?)=(".+?"|[^"]+?)$/
};

function process(event) {
    var fileset = event.Get("fileset.name")
    var keyPrefix = "tidb." + fileset + "."

    // get the body([message] and [k-v]s)
    var raw = event.Get("tidb.body")
    if (raw === null) {
        return event
    }

    // get message
    var messageMatch = params.messagePat.exec(raw)
    if (messageMatch === null ) {
        return event
    }
    var message = messageMatch[1]
    event.Put("message", message)

    // get k-vs
    var kvString = raw.substring(messageMatch[0].length).trim()
    if (kvString.length <= 0) {
        event.Delete("tidb.body")
        return event
    }
    var kvStringList = kvString.substring(1, kvString.length - 1).split(params.splitPat)
    for (var i = 0; i < kvStringList.length; i++) {
        var kvMatch = params.kvPat.exec(kvStringList[i])
        if (kvMatch === null ) {
            return event
        }
        var k = kvMatch[1]
        var v = kvMatch[2]
        if (k.lastIndexOf("\"", 0) === 0 && k.lastIndexOf("\"") === k.length - 1) {
            k = k.substring(1, k.length - 1)
        }
        if (v.lastIndexOf("\"", 0) === 0 && v.lastIndexOf("\"") === v.length - 1) {
            v = v.substring(1, v.length - 1)
        }
        event.Put(keyPrefix + k, v)
    }

    event.Delete("tidb.body")
    return event
}
