// spellchecker: disable
var exif_lookup = {
    // PE files
    "CharacterSet": "character_set",
    "CodeSize": "code_size",
    "CompanyName": "company_name",
    "EntryPoint": "entry_point",
    "FileDescription": "file_description",
    "FileFlagsMask": "file_flags_mask",
    "FileOS": "file_os",
    "FileSize": "file_size",
    "FileSubtype": "file_subtype",
    "FileType": "file_type",
    "FileTypeExtension": "file_type_extension",
    "FileVersion": "file_version",
    "FileVersionNumber": "file_version_number",
    "ImageVersion": "image_version",
    "ImageFileCharacteristics": "image_file_characteristics",
    "InitializedDataSize": "initialized_data_size",
    "InternalName": "internal_name",
    "LanguageCode": "language_code",
    "LegalCopyright": "legal_copyright",
    "LinkerVersion": "linker_version",
    "MIMEType": "mime_type",
    "MachineType": "machine_type",
    "OSVersion": "os_version",
    "ObjectFileType": "object_file_type",
    "OriginalFileName": "original_file_name",
    "PEType": "pe_type",
    "ProductName": "product_name",
    "ProductVersion": "product_version",
    "ProductVersionNumber": "product_version_number",
    "Subsystem": "subsystem",
    "SubsystemVersion": "subsystem_version",
    "TimeStamp": "timestamp",
    "UninitializedDataSize": "uninitialized_data_size",
    // PDF files
    "CreateDate": "create_date",
    "Creator": "creator",
    "CreatorTool": "creator_tool",
    "DocumentID": "document_id",
    "Linearized": "linearized",
    "ModifyDate": "modify_date",
    "PDFVersion": "pdf_version",
    "PageCount": "page_count",
    "Producer": "producer",
    "XMPToolkit": "xmp_toolkit",
    // MachO & ELF files
    "CPUArchitecture": "cpu_architecture",
    "CPUByteOrder": "cpu_byte_order",
    "CPUCount": "cpu_count",
    "CPUType": "cpu_type",
    "CPUSubtype": "cpu_subtype",
    "ObjectFlags": "object_flags"
};

var type_lookup = {
    // Executables
    "peexe": "application/vnd.microsoft.portable-executable",
    "pedll": "application/vnd.microsoft.portable-executable",
    "neexe": "application/x-dosexec",
    "nedll": "application/x-dosexec",
    "mz": "application/x-dosexec",
    "msi": "application/x-msi",
    "com": "application/x-dosexec",
    "coff": "application/x-coff",
    "elf": "application/x-executable",
    "rpm": "application/x-rpm",
    "macho": "application/x-mach-o-executable",

    // Internet
    "html": "text/html",
    "xml": "application/xml",
    "flash": "application/x-shockwave-flash",
    "fla": "application/x-flash-authoring-material",
    "iecookie": "text/x-ie-cookie",
    "bittorrent": "",
    "email": "message/rfc822",
    "outlook": "application/x-outlook-message",
    "cap": "application/vnd.tcpdump.pcap",

    // Images
    "jpeg": "image/jpeg",
    "emf": "application/x-emf",
    "tiff": "image/tiff",
    "gif": "image/gif",
    "png": "image/png",
    "bmp": "image/bmp",
    "gimp": "image/x-xcf",
    "indesign": "application/x-adobe-indesign",
    "psd": "image/vnd.adobe.photoshop",
    "dib": "image/x-ms-bmp",
    "jng": "video/x-jng",
    "ico": "image/vnd.microsoft.icon",
    "fpx": "image/vnd.fpx",
    "eps": "application/postscript",
    "svg": "image/svg+xml",

    // Video & audio
    "ogg": "audio/ogg",
    "flc": "video/x-flc",
    "fli": "video/x-fli",
    "mp3": "audio/mpeg",
    "flac": "audio/x-flac",
    "wav": "audio/wav",
    "midi": "audio/midi",
    "avi": "video/x-msvideo",
    "mpeg": "video/mpeg",
    "qt": "video/quicktime",
    "asf": "application/vnd.ms-asf",
    "divx": "video/x-divx",
    "flv": "video/x-flv",
    "wma": "audio/x-ms-wma",
    "wmv": "video/x-ms-wmv",
    "rm": "application/vnd.rn-realmedia",
    "mov": "video/quicktime",
    "mp4": "video/mp4",
    "3gp": "video/3gpp",

    // Documents
    "text": "text/plain",
    "pdf": "application/pdf",
    "ps": "application/postscript",
    "doc": "application/msword",
    "docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    "rtf": "application/rtf",
    "ppt": "application/vnd.mspowerpoint",
    "pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
    "xls": "application/vnd.ms-excel",
    "xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    "odp": "application/vnd.oasis.opendocument.presentation",
    "ods": "application/vnd.oasis.opendocument.spreadsheet",
    "odt": "application/vnd.oasis.opendocument.text",
    "hwp": "application/x-hangul-word",
    "gul": "application/x-samsung-document",
    "ebook": "application/vnd.amazon.ebook",
    "latex": "application/x-latex",

    // Bundles
    "isoimage": "application/x-iso9660-image",
    "zip": "application/zip",
    "gzip": "application/gzip",
    "bzip": "application/x-bzip",
    "rzip": "application/x-rzip-compressed",
    "dzip": "application/x-dzip-compressed",
    "7zip": "application/x-7z-compressed",
    "cab": "application/vnd.ms-cab-compressed",
    "jar": "application/java-archive",
    "rar": "application/vnd.rar",
    "mscompress": "application/x-mscompress",
    "ace": "application/x-ace-compressed",
    "arc": "application/x-arc-compressed",
    "arj": "application/x-arj-compressed",
    "asd": "application/x-asd-compressed",
    "blackhole": "application/x-blackhole-compressed",
    "kgb": "application/x-kgb-compressed",

    // Code
    "script": "application/x-sh",
    "php": "text/x-php",
    "python": "text/x-python",
    "perl": "text/x-perl",
    "ruby": "text/x-ruby",
    "c": "text/x-csrc",
    "cpp": "text/x-c++src",
    "java": "text/x-java-source",
    "shell": "application/x-sh",
    "pascal": "text/x-pascal",
    "awk": "text/x-awk",
    "dyalog": "text/x-dyalog",
    "fortran": "text/x-fortran",
    "java-bytecode": "application/java-vm",

    // Apple
    "mac": "application/macbinary",
    "applesingle": "application/applefile; version=\"1\"",
    "appledouble": "application/applefile; version=\"2\"",
    "machfs": "application/x-apple-diskimage",

    // Miscellaneous
    "lnk": "application/x-windows-shortcut",
    "ttf": "font/ttf",
    "rom": "application/x-rom-image"
};

var androguard_lookup = {
    "Activities": "activities",
    "AndroguardVersion": "version",
    "AndroidApplication": "android_application",
    "AndroidApplicationError": "android_application_error",
    "AndroidApplicationInfo": "android_application_info",
    "AndroidVersionCode": "android_version_code",
    "AndroidVersionName": "android_version_name",
    "Libraries": "libraries",
    "main_activity": "main_activity",
    "MinSdkVersion": "minimum_sdk_version",
    "Package": "package",
    "Providers": "providers",
    "Receivers": "receivers",
    "RiskIndicator": "risk_indicator",
    "Services": "services",
    "StringsInformation": "strings_information",
    "TargetSdkVersion": "target_sdk_version",
    "VTAndroidInfo": "vt_android_info",
    "certificate": "certificate",
    "intent_filters": "intent_filters",
    "permission_details": "permission_details"
};

var yara_result_lookup = {
    "description": "description",
    "match_in_subfile": "match_in_subfile",
    "rule_name": "name",
    "ruleset_id": "ruleset_id",
    "ruleset_name": "ruleset",
    "reference": "source"
};

var vtAnalysis = (function () {
    var processor = require("processor");
    var console = require("console");

    var normalizeAnalysis = function (evt) {
        console.debug("vtAnalysis.normalizeAnalysis()");

        var last_analysis_results = evt.Get("virustotal.attributes.last_analysis_results");

        if (last_analysis_results != null) {
            console.debug("last_analysis_results: \n" + JSON.stringify(last_analysis_results, undefined, 2));
            var analysis_results = Array();
            Object.keys(last_analysis_results).forEach(function (key) {
                analysis_results.push(last_analysis_results[key]);
            });

            evt.Put("virustotal.analysis.results", analysis_results);
            evt.Delete("virustotal.attributes.last_analysis_results");
        }
    };

    var normalizeCommunityRules = function (evt) {
        console.debug("vtAnalysis.normalizeCommunityRules()");

        var yara_results = evt.Get("virustotal.community.yara_results");

        if (yara_results != null) {
            console.debug("yara_results: \n" + JSON.stringify(yara_results, undefined, 2));
            var normal_results = Array();
            for (var i = 0; i < yara_results.length; i++) {
                var rule = {};
                var clean_key = "";
                Object.keys(yara_results[i]).forEach(function (key) {
                    if (key in yara_result_lookup) {
                        clean_key = yara_result_lookup[key];
                    } else {
                        clean_key = key;
                    }
                    rule[clean_key] = yara_results[key];
                });
            }

            evt.Put("virustotal.community.rules", normal_results);
            evt.Delete("virustotal.community.yara_results");
        }
    };

    var snakeCaseExifTool = function (evt) {
        console.debug("vtAnalysis.snakeCaseExifTool()");

        var exifdata = evt.Get("virustotal.exiftool");
        console.debug("exifdata: \n" + JSON.stringify(exifdata, undefined, 2));

        if (exifdata != null) {
            var exif_clean = {};
            var clean_key = "";
            Object.keys(exifdata).forEach(function (key) {
                if (key in exif_lookup) {
                    clean_key = exif_lookup[key];
                } else {
                    // By default, don't change the key so that it's easy to notice and fix
                    clean_key = key;
                }
                exif_clean[clean_key] = exifdata[key];
            });

            evt.Put("virustotal.exiftool", exif_clean);
        }
    };

    var snakeCaseAndroguard = function (evt) {
        console.debug("vtAnalysis.snakeCaseAndroguard()");

        var androdata = evt.Get("virustotal.androguard");
        console.debug("androguard: \n" + JSON.stringify(androdata, undefined, 2));

        if (androdata != null) {
            var andro_clean = {};
            var clean_key = "";
            Object.keys(androdata).forEach(function (key) {
                if (key in androguard_lookup) {
                    clean_key = androguard_lookup[key];
                } else {
                    // By default, don't change the key so that it's easy to notice and fix
                    clean_key = key;
                }
                andro_clean[clean_key] = androdata[key];
            });

            evt.Put("virustotal.androguard", andro_clean);
        }
    };

    var addMimeType = function (evt) {
        console.debug("vtAnalysis.addMimeType");
        var type_tag = evt.Get("virustotal.type_tag");

        if (type_tag != null && type_tag in type_lookup) {
            evt.Put("file.mime_type", type_lookup[type_tag]);
        }
    };

    var processMessage = new processor.Chain()
        .Add(function (evt) {
            normalizeAnalysis(evt);
            normalizeCommunityRules(evt);
            snakeCaseExifTool(evt);
            snakeCaseAndroguard(evt);
            addMimeType(evt);
        })
        .Build();

    return {
        process: function (evt) {
            console.debug("vtAnalysis.process()");
            processMessage.Run(evt);
        }
    };
})();

function process(evt) {
    vtAnalysis.process(evt);
}
