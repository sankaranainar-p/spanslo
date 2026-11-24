# Open questions

## OQ-1: Duplicate JSON object keys in canonicalization
`core/canon.go` keeps the last value when a JSON object has a duplicate key. RFC 8785 says to reject such input. Accepted as a known limit for v0.2.0. Revisit after the tag.
