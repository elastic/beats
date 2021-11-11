package bundle

var FilePolicies = map[string]string{
    "compliance/CIS.1.2.1.rego": `


    finding = {"evaluation": evaluation, "rule_name": rule_name, "fields": fields, "tags": tags} {
    input.filename == "etcd.yaml"
    fileMode := input.fileMode
    pattern := "0?(0|1|2|3|4|5|6)(0|1|2|3|4)(0|1|2|3|4)"
    rule_evaluation := regex.match(pattern, filemode)

    # set result
    evaluation := calculate_result(rule_evaluation)
    fields := [{ "key": "filemode", "value": filemode }]
    rule_name := "Ensure that the etcd pod specification file permissions are set to 644 or more restrictive"
    tags := ["CIS 1.1.7"]
}


    calculate_result(evaluation) = "passed" {
    evaluation
} else = "violation"
`,
}
