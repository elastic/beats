Diagnostic Contexts
===================

Go documentation: https://godoc.org/github.com/urso/diag

Package diag provides a diagnostic context that can be used to record
contextual information about the current scope, that needs to be reported.
Diagnostic contexts can be used for logging, to add context when wrapping
errors, or to pass additional information with data/events/objects passed
through multiple subsystems.

Contexts are represented as trees. A Context has a 'before'-Context and an
'after'-Context. The order of contexts define the shadowing of fields in
case multiple contexts reported different values for the same field.  All
fields in the 'after'-Context always overwrite fields in the current context
node and fields in the 'before'-Context. Fields in the 'before'-Context are
always shadowed by the current Context node and the 'after'-Context.

The diag package differentiates between standardized and user defined fields.
Although diag does not define any standardized fields, libraries and users are
encouraged to create constructors for standardized fields.
Standardized fields ensure that consistent field names and types are used by programmers.
Constructors can add some type-safety, for post-processing or storing a
diagnostic context state in databases or other storage systems that require
a schema.

One can define a standardized "Host" field like this:

    package myfields

    func Host(name string) diag.Field {
        return diag.Field{Standardized: true, Key: "host", Value: diag.ValString(name)}
    }

The fields can be used with a context like this:

    ctx.AddField(myfields.Host("localhost"))

