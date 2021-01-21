// spellchecker: disable
var vtPE = (function () {
    var processor = require("processor");
    var console = require("console");

    var normalizeImports = function (evt) {
        console.debug("vtPE.normalizeImports");

        var imports = evt.Get("file.pe.imports");
        var normal_imports = Array();

        if (imports != null) {
            console.debug("imports[" + imports.length + "]: \n" + JSON.stringify(imports, undefined, 2));

            /* The goal is to normalize import list to the following
             * structure:
             * {
             *  "name": "MY_SYMBOL", (keyword)
             *  "type": "function", (keyword, lowercased; normalized to function, object, notype, thread local symbol
             *  "library_name": "kernel32.dll"
             * }
             */
            var norm_imports = Array();
            for (var i = 0; i < imports.length; i++) {
                var libname = imports[i].library_name;
                for (var j = 0; j < imports[i].imported_functions.length; j++) {
                    norm_imports.push(
                        {
                            "name": imports[i].imported_functions[j],
                            "type": "function",
                            "library_name": libname
                        }
                    );
                }
            }

            console.debug("normalized imports[" + norm_imports.length + "]: \n" + JSON.stringify(norm_imports, undefined, 2));
            evt.Delete("file.pe.imports");
            evt.Put("file.pe.imports", norm_imports);
        }
    };

    var normalizeExports = function (evt) {
        console.debug("vtPE.normalizeExports");

        var exports = evt.Get("file.pe.exports");
        if (exports != null) {
            console.debug("exports[" + exports.length + "]: \n" +
                JSON.stringify(exports, undefined, 2));

            /* The goal is to normalize export list to the following
             * structure:
             * {
             *  "name": "MY_SYMBOL", (keyword)
             *  "type": "function", (keyword, lowercased; normalized to function, object, notype, thread local symbol
             *  "library_name": "kernel32.dll"
             * }
             */
            var norm_exports = Array();
            for (var i = 0; i < exports.length; i++) {
                norm_exports.push(
                    {
                        "name": exports[i],
                        "type": "function",
                    }
                );
            }

            evt.Delete("file.pe.exports");
            evt.Put("file.pe.exports", norm_exports);
        }
    };

    var normalizeSections = function (evt) {
        console.debug("vtPE.normalizeSections");

        var sections = evt.Get("file.pe.sections");

        // original sections entry: [{
        //     "chi2": 144106.34,
        //     "virtual_address": 8192,
        //     "entropy": 5.29,
        //     "name": ".text",
        //     "flags": "rx",
        //     "raw_size": 5632,
        //     "virtual_size": 5316,
        //     "md5": "9002a963c87901397a986c3333d09627"
        //   },...]
        if (sections != null) {
            console.debug("sections[" + sections.length + "]: \n" +
                JSON.stringify(sections, undefined, 2));

            // {
            //     name: "Name of code section",
            //     physical_offset: "[keyword] Offset of the section from the beginning of the segment, in hex",
            //     physical_size: "[long] Size of the code section in the file in bytes",
            //     virtual_address: "[keyword] relative virtual memory address when loaded",
            //     virtual_size: "[long] Size of the section in bytes when loaded into memory",
            //     flags: "[keyword] List of flag values as strings for this section",
            //     type: "[keyword] Section type as string, if applicable",
            //     segment_name: "[keyword] Name of segment for this section, if applicable",
            //     entropy: "[float] shannon entropy calculated from section content in bits per byte of information",
            //     chi2: "[float]"
            // }
            var normal_sections = Array();
            for (var i = 0; i < sections.length; i++) {
                var norm_sect = {
                    "name": sections[i].name,
                    "physical_size": sections[i].raw_size,
                    "virtual_address": "0x" + sections[i].virtual_address.toString(16).toUpperCase(),
                    "virtual_size": sections[i].virtual_size,
                    "flags": sections[i].flags,
                    "entropy": sections[i].entropy,
                    "chi2": sections[i].chi2
                };

                // Allow for different hashes in the future
                var hashes = {};
                if (sections[i].hasOwnProperty("md5")) {
                    hashes["md5"] = sections[i].md5;
                }

                if (hashes != {}) {
                    norm_sect["hash"] = hashes;
                }

                normal_sections.push(norm_sect);
            }
        }

    };

    var processMessage = new processor.Chain()
        .Add(function (evt) {
            normalizeImports(evt);
            normalizeExports(evt);
            // normalizeSections(evt);
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

