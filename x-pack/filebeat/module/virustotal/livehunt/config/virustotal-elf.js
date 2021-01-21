// spellchecker: disable

var elf_symbol_type_lookup = {
    "NOTYPE": "no type",
    "FUNC": "function",
    "TLS": "thread local symbol",
    "OBJECT": "object"
};

function isLetter(c) {
    return c.toLowerCase() != c.toUpperCase();
}


var vtELF = (function () {
    var processor = require("processor");
    var console = require("console");

    var normalizeImports = function (evt) {
        console.debug("vtELF.normalizeImports");

        var imports = evt.Get("file.elf.imports");
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

        if (exports != null) {
            console.debug("exports[" + exports.length + "]: \n" + JSON.stringify(exports, undefined, 2));

            /* The goal is to normalize exports list to the following
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

    var normalizeSections = function (evt) {
        console.debug("vtELF.normalizeSections");

        var sections = evt.Get("file.elf.sections");

        // Input structure
        // {
        //     "section_type": "NULL",
        //     "virtual_address": 0,
        //     "size": 0,
        //     "physical_offset": 0,
        //     "flags": "",
        //     "name": ""
        //   },
        if (sections != null) {
            for (var i = 0; i < sections.length; i++) {
                var old_sect = sections[i];
                var new_sect = {
                    "name": old_sect.name,
                    // VT returns this field with a misspelling
                    "physical_offset": "0x" + old_sect.phisical_offset.toString(16).toUpperCase(),
                    "physical_size": old_sect.size,
                    "virtual_address": "0x" + old_sect.virtual_address.toString(16).toUpperCase(),
                    "type": old_sect.section_type
                }

                // Section flags: https://en.wikipedia.org/wiki/Executable_and_Linkable_Format#Section_header
                // 0x1 	        SHF_WRITE 	            Writable
                // 0x2 	        SHF_ALLOC 	            Occupies memory during execution
                // 0x4 	        SHF_EXECINSTR          	Executable
                // 0x10 	    SHF_MERGE 	            Might be merged
                // 0x20 	    SHF_STRINGS         	Contains null-terminated strings
                // 0x40 	    SHF_INFO_LINK          	'sh_info' contains SHT index
                // 0x80 	    SHF_LINK_ORDER      	Preserve order after combining
                // 0x100 	    SHF_OS_NONCONFORMING 	Non-standard OS specific handling required
                // 0x200 	    SHF_GROUP           	Section is member of a group
                // 0x400 	    SHF_TLS 	            Section hold thread-local data
                // 0x0ff00000 	SHF_MASKOS 	            OS-specific
                // 0xf0000000 	SHF_MASKPROC 	        Processor-specific
                // 0x4000000 	SHF_ORDERED 	        Special ordering requirement (Solaris)
                // 0x8000000 	SHF_EXCLUDE 	        Section is excluded unless referenced or allocated (Solaris)
                var flag_lookup = {
                    "W": "WRITE",
                    "A": "ALLOC",
                    "X": "EXECINSTR",
                    "M": "MERGE",
                    "S": "STRINGS",
                    "I": "INFO_LINK",
                    "T": "TLS"
                }

                console.debug("section flags[" + old_sect.flags.length + "]: \n" + old_sect.flags);

                var new_flags = [];
                for (var j = 0; j < old_sect.flags.length; j++) {

                    var flag = old_sect.flags[j];
                    console.debug("flag[" + j + "]: " + flag[j]);
                    if (flag_lookup.hasOwnProperty(flag)) {
                        new_flags.push(flag_lookup[flag]);
                    } else {
                        new_flags.push(flag);
                    }
                }

                if (new_flags.length > 0) {
                    new_sect["flags"] = new_flags;
                }

                // Replace existing section
                sections[i] = new_sect;
            }

            evt.Delete("file.elf.sections");
            evt.Put("file.elf.sections", sections);
        }
    };

    var normalizeSegments = function (evt) {
        console.debug("vtELF.normalizeSegments");

        // Input structure:
        // [ {
        //     "resources": [
        //       ".note.gnu.build-id",
        //       ".gnu.hash",
        //       ".dynsym",
        //       ".dynstr",
        //       ".gnu.version",
        //       ".gnu.version_r",
        //       ".rela.dyn",
        //       ".rela.plt",
        //       ".init",
        //       ".plt",
        //       ".plt.got",
        //       ".text",
        //       ".fini",
        //       ".rodata",
        //       ".eh_frame_hdr",
        //       ".eh_frame"
        //     ],
        //     "segment_type": "LOAD"
        //   },
        var segments = evt.Get("file.elf.segments");
        if (segments != null) {
            var new_segments = Array();
            for (var i = 0; i < segments.length; i++) {
                new_segments.push({ "type": segments[i].segment_type, "sections": segments[i].resources })

                // IDEA: could perhaps loop through all sections in this segment to calculate physical address and size
            }

            evt.Delete("file.elf.segments");
            evt.Put("file.elf.segments", new_segments);
        }
        // Desired output
        // {
        //     "file_offset": "[keyword] Address in hex string of segment location", //# Not sure about this
        //     "flags": "[keyword] flags on segment, if present",
        //     "physical_address": "[keyword] Address as hex string of segment",
        //     "physical_size": "[long] Size of segment, in bytes",
        //     "sections": "[keyword] List of section names present in this segment",
        //     "type": "[keyword] Type of segment",
        //     "virtual_address": "[keyword]"
        // }
    };

    var processMessage = new processor.Chain()
        .Add(function (evt) {
            normalizeImports(evt);
            normalizeExports(evt);
            normalizeSections(evt);
            normalizeSegments(evt);
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
