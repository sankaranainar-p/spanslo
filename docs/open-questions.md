# Open questions

## OQ-1: Duplicate JSON object keys in canonicalization
`core/canon.go` keeps the last value when a JSON object has a duplicate key. RFC 8785 says to reject such input. Accepted as a known limit for v0.2.0. Revisit after the tag.

## OQ-2: Fault event name is hardcoded
The name `spanslo.fault` is declared in `spec/semconv-mapping.yaml` (`fault_event.name`, mapping_version 9) but core still keeps a constant (`FaultEventName` in `core/turnmismatch.go`). A test (`core/faultname_test.go`) fails if they differ. Load the name from the mapping in v0.3.0.
