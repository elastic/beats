// spellchecker: disable

var elf_symbol_type_lookup = {
    "NOTYPE": "no type",
    "FUNC": "function",
    "TLS": "thread local symbol",
    "OBJECT": "object"
};

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

    var normalizeImports = function (evt) {
        console.debug("vtELF.normalizeImports");

        var imports = evt.Get("file.elf.imports");
        var normal_imports = Array();

        if (import != null) {
            console.debug("imports[" + imports.length + "]: \n" + JSON.stringify(imports, undefined, 2));

            /* The goal is to normalize import list to the following
             * structure:
             * {
             *  "name": "MY_SYMBOL", (keyword)
             *  "type": "function", (keyword, lowercased; normalized to function, object, notype, thread local symbol
             *  "library_name": "kernel32.dll"
             * }
             *
             * NOTE: VT doesn't return resolved library_name for ELF files, so this will be omitted
             */
            for (var i = 0; i < imports.length; i++) {
                var sym_type = imports[i].type;
                if (sym_type in elf_symbol_type_lookup) {
                    imports[i].type = elf_symbol_type_lookup[sym_type];
                }
            }

            evt.Delete("file.elf.imports");
            evt.Put("file.elf.imports", imports);
        }

    };

    var normalizeExports = function (evt) {
        console.debug("vtELF.normalizeExports");

        var exports = evt.Get("file.elf.exports");
        var normal_exports = Array();

        if (import != null) {
            console.debug("exports[" + exports.length + "]: \n" + JSON.stringify(exports, undefined, 2));

            /* The goal is to normalize import list to the following
             * structure:
             * {
             *  "name": "MY_SYMBOL", (keyword)
             *  "type": "function", (keyword, lowercased; normalized to function, object, notype, thread local symbol
             *  "library_name": "kernel32.dll"
             * }
             *
             * NOTE: VT doesn't return resolved library_name for ELF files, so this will be omitted
             */
            for (var i = 0; i < exports.length; i++) {
                var sym_type = exports[i].type;
                if (sym_type in elf_symbol_type_lookup) {
                    exports[i].type = elf_symbol_type_lookup[sym_type];
                }
            }

            evt.Delete("file.elf.exports");
            evt.Put("file.elf.exports", exports);
        }

    };

    var processMessage = new processor.Chain()
        .Add(function (evt) {
            correctSpelling(evt);
            enumeratePackers(evt);
            normalizeImports(evt);
            normalizeExports(evt);
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
