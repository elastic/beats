// spellchecker: disable
var vtPE = (function () {
    var processor = require("processor");
    var console = require("console");

    var normalizeImports = function (evt) {
        console.debug("vtPE.normalizeImports");

        var imports = evt.Get("file.pe.imports");
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
             */
            var norm_imports = Array();
            for (var i = 0; i < imports.length; i++) {
                var libname = imports[i].library_name;
                for (var j = 0; i < imports[i].imported_functions.length; j++) {
                    norm_imports.push(
                        {
                            "name": imports[i].imported_functions[j],
                            "type": "function",
                            "library_name": libname
                        }
                    );
                }
            }

            evt.Delete("file.pe.imports");
            evt.Put("file.pe.imports", norm_imports);
        }
    }
};

var normalizeExports = function (evt) {
    console.debug("vtELF.normalizeExports: TBD");
};


var processMessage = new processor.Chain()
    .Add(function (evt) {
        normalizeImports(evt);
    })
    .Build();

return {
    process: function (evt) {
        processMessage.Run(evt);
    }
}
}) ();

function process(evt) {
    vtPE.process(evt);
}

