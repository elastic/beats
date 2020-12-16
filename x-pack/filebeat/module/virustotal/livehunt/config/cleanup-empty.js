// spellchecker: disable

var params = { field: "" };

function register(scriptParams) {
    params = scriptParams;
}

// Adapted from https://stackoverflow.com/a/679937/6654930
function isEmpty(obj) {
    for (var prop in obj) {
        if (obj.hasOwnProperty(prop))
            return false;
    }
    return true;
}

function process(evt) {
    console.debug("cleanup.cleanEmptyList");

    if (field == "") {
        console.debug("Empty field parameter. Skipping.");
        return;
    }

    var field_data = evt.Get(params.field);
    // This works for empty objects, arrays, and null
    if (typeof field_data == "object" && isEmpty(field_data)) {
        console.debug("Field " + params.field + " is empty. Deleting.");
        evt.Delete(params.field);
    }
}

