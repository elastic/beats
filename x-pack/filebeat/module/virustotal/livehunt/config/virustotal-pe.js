// spellchecker: disable
var vtPE = (function () {
    var processor = require("processor");
    var console = require("console");

    var enumeratePackers = function (evt) {
        console.debug("vtPE.splitPackers()");

        var packers = evt.Get("virustotal.packers");
        var packer_list = Array();

        if (packers != null) {
            Object.keys(packers).forEach(function (key) {
                packer_list.push(packers[key]);
            });
            evt.Put("file.pe.packers", packer_list);
        }
    };

    var enumeratePackers = function (evt) {
        console.debug("vtPE.splitPackers()");

        var packers = evt.Get("file.pe.flattened.packers");
        var packer_list = Array();

        if (packers != null) {
            Object.keys(packers).forEach(function (key) {
                packer_list.push(packers[key]);
            });
            evt.Put("file.pe.packers", packer_list);
        }
    };

    var processMessage = new processor.Chain()
        .Add(function (evt) {
            enumeratePackers(evt);
        })
        .Build();

    return {
        process: function (evt) {
            processMessage.Run(evt);
        }
    }
})();

function process(evt) {
    vtPE.process(evt);
}

