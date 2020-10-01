// spellchecker: disable
var vtELF = (function () {
    var processor = require("processor");
    var console = require("console");

    var correctSpelling = function (evt) {
        console.debug("vtELF.correctSpelling()");

        var section_list = evt.Get("file.elf.sections");

        if (section_list != null) {
            console.debug("section_list[" + section_list.length + "]: \n" + JSON.stringify(section_list, undefined, 2));
            for (var i = 0; i < section_list.length; i++) {
                if ('phisical_offset' in section_list[i]) {
                    section_list[i].physical_offset = section_list[i].phisical_offset;
                    delete section_list[i].phisical_offset;
                }
            }

            evt.Put("file.elf.sections", section_list);
        }
    };

    var enumeratePackers = function (evt) {
        console.debug("vtELF.splitPackers()");

        var packers = evt.Get("virustotal.packers");
        var packer_list = Array();

        if (packers != null) {
            console.debug("packers[" + packers.length + "]: \n" + JSON.stringify(packers, undefined, 2));

            Object.keys(packers).forEach(function (key) {
                packer_list.push(packers[key]);
            });
            evt.Put("file.elf.packers", packer_list);
        }
    };

    var processMessage = new processor.Chain()
        .Add(function (evt) {
            correctSpelling(evt);
            enumeratePackers(evt);
        })
        .Build();

    return {
        process: function (evt) {
            console.debug("vtELF.process()");
            processMessage.Run(evt);
        }
    }
})();

function process(evt) {
    vtELF.process(evt);
}
